package cache

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRedis struct {
	data map[string]string
	mu   sync.Mutex
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{data: make(map[string]string)}
}

func (f *fakeRedis) dialer(_ context.Context, _, _ string) (net.Conn, error) {
	clientConn, serverConn := net.Pipe()
	go f.serveConn(serverConn)
	return clientConn, nil
}

func (f *fakeRedis) serveConn(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			return
		}
	}()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	for {
		args, err := readCommand(reader)
		if err != nil {
			return
		}
		if err := f.handleCommand(args, writer); err != nil {
			return
		}
		if err := writer.Flush(); err != nil {
			return
		}
	}
}

func (f *fakeRedis) handleCommand(args []string, w *bufio.Writer) error {
	if len(args) == 0 {
		return writeError(w, "empty command")
	}

	switch strings.ToUpper(args[0]) {
	case "PING":
		return writeSimpleString(w, "PONG")
	case "CLIENT":
		return writeSimpleString(w, "OK")
	case "QUIT":
		return writeSimpleString(w, "OK")
	case "SET":
		if len(args) < 3 {
			return writeError(w, "invalid args")
		}
		f.mu.Lock()
		f.data[args[1]] = args[2]
		f.mu.Unlock()
		return writeSimpleString(w, "OK")
	case "INCR":
		if len(args) < 2 {
			return writeError(w, "invalid args")
		}
		f.mu.Lock()
		valStr := f.data[args[1]]
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			val = 0
		}
		val++
		f.data[args[1]] = strconv.FormatInt(val, 10)
		f.mu.Unlock()
		return writeInt(w, val)
	case "EXPIRE":
		return writeInt(w, 1)
	case "GET":
		if len(args) < 2 {
			return writeError(w, "invalid args")
		}
		f.mu.Lock()
		val, ok := f.data[args[1]]
		f.mu.Unlock()
		if !ok {
			return writeNil(w)
		}
		return writeBulkString(w, val)
	case "EXISTS":
		if len(args) < 2 {
			return writeError(w, "invalid args")
		}
		f.mu.Lock()
		_, ok := f.data[args[1]]
		f.mu.Unlock()
		if ok {
			return writeInt(w, 1)
		}
		return writeInt(w, 0)
	case "DEL":
		if len(args) < 2 {
			return writeError(w, "invalid args")
		}
		f.mu.Lock()
		_, ok := f.data[args[1]]
		if ok {
			delete(f.data, args[1])
		}
		f.mu.Unlock()
		if ok {
			return writeInt(w, 1)
		}
		return writeInt(w, 0)
	default:
		return writeError(w, "unknown command")
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if prefix != '*' {
		return nil, errors.New("unexpected RESP prefix")
	}

	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	count, err := strconv.Atoi(line)
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		bulkPrefix, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if bulkPrefix != '$' {
			return nil, errors.New("unexpected bulk prefix")
		}
		lenLine, err := readLine(r)
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(lenLine)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}

	return args, nil
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

func writeSimpleString(w *bufio.Writer, msg string) error {
	_, err := w.WriteString("+" + msg + "\r\n")
	return err
}

func writeError(w *bufio.Writer, msg string) error {
	_, err := w.WriteString("-ERR " + msg + "\r\n")
	return err
}

func writeInt(w *bufio.Writer, value int64) error {
	_, err := w.WriteString(":" + strconv.FormatInt(value, 10) + "\r\n")
	return err
}

func writeBulkString(w *bufio.Writer, value string) error {
	_, err := w.WriteString("$" + strconv.Itoa(len(value)) + "\r\n" + value + "\r\n")
	return err
}

func writeNil(w *bufio.Writer) error {
	_, err := w.WriteString("$-1\r\n")
	return err
}

func newFakeRedisClient(t *testing.T) *redis.Client {
	server := newFakeRedis()
	client := redis.NewClient(&redis.Options{
		Addr:   "fake",
		Dialer: server.dialer,
	})
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close redis client: %v", err)
		}
	})
	return client
}

func TestRedisUserCache_BasicFlow(t *testing.T) {
	client := newFakeRedisClient(t)
	cache := NewRedisUserCache(client)

	exists, err := cache.Exists()
	require.NoError(t, err)
	assert.False(t, exists)

	version, err := cache.GetVersion()
	require.NoError(t, err)
	assert.Equal(t, int64(0), version)

	users := []define.AllowListUser{
		{Phone: "13800138000", Mail: "user1@example.com"},
		{Phone: "13900139000", Mail: "user2@example.com"},
	}
	require.NoError(t, cache.Set(users))

	exists, err = cache.Exists()
	require.NoError(t, err)
	assert.True(t, exists)

	got, err := cache.Get()
	require.NoError(t, err)
	assert.Equal(t, users, got)

	version, err = cache.GetVersion()
	require.NoError(t, err)
	assert.Greater(t, version, int64(0))

	require.NoError(t, cache.Clear())

	exists, err = cache.Exists()
	require.NoError(t, err)
	assert.False(t, exists)

	got, err = cache.Get()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRedisUserCache_GetInvalidJSON(t *testing.T) {
	client := newFakeRedisClient(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, REDIS_CACHE_KEY, "invalid-json", REDIS_CACHE_TTL).Err())

	cache := NewRedisUserCache(client)
	users, err := cache.Get()
	assert.Error(t, err)
	assert.Nil(t, users)
}
