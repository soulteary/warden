package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRefreshDeadline(t *testing.T) {
	// 1s + 2s + 4s of retry backoff on top of the HTTP timeout.
	require.Equal(t, 37*time.Second, refreshDeadline(30*time.Second))
	require.Equal(t, 12*time.Second, refreshDeadline(5*time.Second))
	require.Equal(t, 12*time.Second, refreshDeadline(0), "no timeout configured selects the default")
	require.Equal(t, 10*time.Second, refreshDeadline(time.Second), "never below the previous fixed deadline")
}
