package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	version "github.com/soulteary/version-kit/v4"
)

type versionErrorWriter struct{}

func (versionErrorWriter) Write([]byte) (int, error) { return 0, errors.New("synthetic write failure") }

func TestVersionCommandUsesNativeVersionMetadata(t *testing.T) {
	var output bytes.Buffer
	handled, err := handleVersionCommand([]string{"--version"}, &output)
	if !handled || err != nil {
		t.Fatalf("version command: handled=%v err=%v", handled, err)
	}
	var got version.Info
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != *version.Default() || !strings.HasSuffix(output.String(), "\n") {
		t.Fatal("version output must preserve native metadata without truncation or normalization")
	}
}

func TestVersionCommandRejectsCombinationsWithoutStartupFallback(t *testing.T) {
	for _, args := range [][]string{{"--version=true"}, {"--version=false"}, {"--version", "extra"}, {"--port", "8080", "--version"}, {"--", "--version"}} {
		var output bytes.Buffer
		handled, err := handleVersionCommand(args, &output)
		if !handled || err == nil || output.Len() != 0 {
			t.Fatalf("invalid version arguments were not refused: %q", args)
		}
	}
	for _, args := range [][]string{nil, {"--port", "8080"}, {"-version"}, {"-v"}, {"--unknown"}} {
		handled, err := handleVersionCommand(args, io.Discard)
		if handled || err != nil {
			t.Fatalf("ordinary argument handling changed: %q", args)
		}
	}
}

func TestVersionCommandWriteFailureDoesNotFallThrough(t *testing.T) {
	handled, err := handleVersionCommand([]string{"--version"}, versionErrorWriter{})
	if !handled || err == nil {
		t.Fatal("output failure must remain a handled, failed version command")
	}
}

func TestVersionCommandBinaryEarlyExit(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal("Go tool required to verify the real linker metadata path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "warden-version-test")
	// These are explicit synthetic linker values, never release identity claims.
	flags := "-w -s -X github.com/soulteary/version-kit/v4.Version=version-command-test -X github.com/soulteary/version-kit/v4.Commit=synthetic-full-commit -X github.com/soulteary/version-kit/v4.BuildDate=synthetic-build-date"
	build := exec.CommandContext(ctx, goTool, "build", "-mod=readonly", "-buildvcs=false", "-trimpath", "-ldflags", flags, "-o", binary, ".")
	build.Env = []string{"CGO_ENABLED=0", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local", "GOENV=off", "GOWORK=off"}
	for _, key := range []string{"PATH", "HOME", "USERPROFILE", "SystemRoot", "WINDIR", "PATHEXT", "GOROOT", "GOMODCACHE", "GOCACHE", "TMPDIR", "TEMP", "TMP"} {
		if value, ok := os.LookupEnv(key); ok {
			build.Env = append(build.Env, key+"="+value)
		}
	}
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build version fixture: %v\n%s", err, output)
	}
	// No inherited application configuration or secrets. If startup is reached,
	// this nonexistent config must fail before any service can be initialized.
	env := []string{"CONFIG_FILE=" + filepath.Join(t.TempDir(), "absent.json"), "REDIS_ENABLED=true", "REDIS_ADDR=127.0.0.1:1"}
	run := func(args []string, stdout io.Writer) (string, error) {
		t.Helper()
		probe, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		cmd := exec.CommandContext(probe, binary, args...)
		cmd.Env = env
		cmd.Stdout = stdout
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		returnErr := cmd.Run()
		return stderr.String(), returnErr
	}
	var output bytes.Buffer
	stderr, err := run([]string{"--version"}, &output)
	if err != nil || stderr != "" {
		t.Fatalf("metadata-only command failed: %v stderr=%q", err, stderr)
	}
	var got version.Info
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal("expected only JSON metadata, without the startup banner")
	}
	if got.Version != "version-command-test" || got.Commit != "synthetic-full-commit" || got.BuildDate != "synthetic-build-date" || got.GoVersion == "" || got.Platform == "" {
		t.Fatal("real program did not report its actual synthetic linker metadata")
	}
	output.Reset()
	stderr, err = run([]string{"--version", "--port", "8080"}, &output)
	if err == nil || stderr != "warden: version command failed\n" || output.Len() != 0 {
		t.Fatal("invalid version combination must fail before application startup")
	}
	badOutput, err := os.Open(os.DevNull) // Read-only descriptor makes stdout fail.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = badOutput.Close() }()
	stderr, err = run([]string{"--version"}, badOutput)
	if err == nil || stderr != "warden: version command failed\n" {
		t.Fatal("real stdout failure must exit nonzero without entering application startup")
	}
}
