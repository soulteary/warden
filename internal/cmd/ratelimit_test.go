package cmd

import (
	"testing"
	"time"

	"github.com/soulteary/warden/internal/config"
	"github.com/soulteary/warden/internal/define"
)

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		raw     string
		want    int
		wantErr bool
	}{
		{raw: "600", want: 600},
		{raw: " 1 ", want: 1},
		{raw: "0", wantErr: true},
		{raw: "-5", wantErr: true},
		{raw: "60/min", wantErr: true},
		{raw: "1.5", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseRateLimit(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseRateLimit(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRateLimit(%q) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}

func TestParseRateLimitWindow(t *testing.T) {
	tests := []struct {
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{raw: "1m", want: time.Minute},
		{raw: "30s", want: 30 * time.Second},
		{raw: "10", want: 10 * time.Second},
		{raw: "500ms", wantErr: true},
		{raw: "0", wantErr: true},
		{raw: "-1m", wantErr: true},
		{raw: "soon", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseRateLimitWindow(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseRateLimitWindow(%q) error = %v, wantErr %v", tt.raw, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRateLimitWindow(%q) = %s, want %s", tt.raw, got, tt.want)
		}
	}
}

func TestRateLimitEnvOverridesConfigFile(t *testing.T) {
	fromFile := &config.CmdConfigData{RateLimit: 120, RateLimitWindow: 2 * time.Minute}

	t.Run("file values are kept without env", func(t *testing.T) {
		t.Setenv("RATE_LIMIT", "")
		t.Setenv("RATE_LIMIT_WINDOW", "")
		cfg := convertToConfig(fromFile)
		if cfg.RateLimit != 120 || cfg.RateLimitWindow != 2*time.Minute {
			t.Fatalf("got %d per %s, want 120 per 2m0s", cfg.RateLimit, cfg.RateLimitWindow)
		}
	})

	t.Run("env wins over file", func(t *testing.T) {
		t.Setenv("RATE_LIMIT", "900")
		t.Setenv("RATE_LIMIT_WINDOW", "30s")
		cfg := convertToConfig(fromFile)
		if cfg.RateLimit != 900 || cfg.RateLimitWindow != 30*time.Second {
			t.Fatalf("got %d per %s, want 900 per 30s", cfg.RateLimit, cfg.RateLimitWindow)
		}
	})

	t.Run("invalid env is reported, not ignored", func(t *testing.T) {
		t.Setenv("RATE_LIMIT", "lots")
		t.Setenv("RATE_LIMIT_WINDOW", "1x")
		cfg := convertToConfig(fromFile)
		if cfg.RateLimit != 120 || cfg.RateLimitWindow != 2*time.Minute {
			t.Fatalf("invalid env changed the values: %d per %s", cfg.RateLimit, cfg.RateLimitWindow)
		}
		if errs := validateRateLimit(cfg); len(errs) != 2 {
			t.Fatalf("validateRateLimit returned %d errors, want 2: %v", len(errs), errs)
		}
	})
}

func TestValidateRateLimitConfigValues(t *testing.T) {
	t.Setenv("RATE_LIMIT", "")
	t.Setenv("RATE_LIMIT_WINDOW", "")

	valid := []Config{
		{}, // zero resolves to the defaults
		{RateLimit: define.DEFAULT_RATE_LIMIT, RateLimitWindow: define.DEFAULT_RATE_LIMIT_WINDOW},
		{RateLimit: 1, RateLimitWindow: time.Second},
	}
	for i := range valid {
		if errs := validateRateLimit(&valid[i]); len(errs) != 0 {
			t.Errorf("valid[%d]: unexpected errors %v", i, errs)
		}
	}

	invalid := []Config{
		{RateLimit: -1},
		{RateLimitWindow: -time.Second},
		{RateLimitWindow: 100 * time.Millisecond},
	}
	for i := range invalid {
		if errs := validateRateLimit(&invalid[i]); len(errs) != 1 {
			t.Errorf("invalid[%d]: got %d errors, want 1: %v", i, len(errs), errs)
		}
	}
}
