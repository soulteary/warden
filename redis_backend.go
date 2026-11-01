package main

import (
	"github.com/redis/go-redis/v9"
	rediskitclient "github.com/soulteary/redis-kit/client"
)

// newRecoverableRedisClient keeps one backend identity even if the initial PING
// fails. redis-kit's verified constructor closes a failed client; a fresh lazy
// client lets subsequent operations reconnect without swapping route dependencies.
func newRecoverableRedisClient(cfg *rediskitclient.Config) (*redis.Client, error) {
	client, err := rediskitclient.NewClient(*cfg)
	if err == nil || cfg.Addr == "" {
		return client, err
	}
	// Match redis-kit's single-node option mapping. Warden currently configures
	// only address/password, but retaining all options avoids a divergent fallback.
	opts := &redis.Options{
		Addr: cfg.Addr, Username: cfg.Username, Password: cfg.Password, DB: cfg.DB,
		PoolSize: cfg.PoolSize, PipelinePoolSize: -1, MinIdleConns: cfg.MinIdleConns,
		DialTimeout: cfg.DialTimeout, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout,
		MaxRetries: cfg.MaxRetries, PoolTimeout: cfg.PoolTimeout,
		ContextTimeoutEnabled: cfg.ContextTimeoutEnabled, Dialer: cfg.Dialer,
	}
	if cfg.TLSConfig != nil {
		opts.TLSConfig = cfg.TLSConfig.Clone()
	}
	return redis.NewClient(opts), err
}
