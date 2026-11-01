package loader

import (
	"context"
	"testing"

	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
)

func TestFallbackEvidenceSurvivesSubsequentMerge(t *testing.T) {
	for _, mode := range []string{"DEFAULT", ModeLocalFirst} {
		t.Run(mode, func(t *testing.T) {
			e := &sourceEvidence{attempts: []SourceAttempt{{}, {}}}
			remote, local := newSourceBatch(e), newSourceBatch(e)
			remote.fallbackOwner, local.fallbackOwner = 0, 1
			remote.inputs[0], local.inputs[1] = true, true
			ctx := context.WithValue(context.Background(), evidenceKey{}, e)
			remoteUsers := []define.AllowListUser{{Mail: "same@example.com", UserID: "remote"}, {Mail: "remote@example.com", UserID: "only-remote"}}
			localUsers := []define.AllowListUser{{Mail: "same@example.com", UserID: "local"}}
			result := LoadResult{Users: mergeObserved(ctx, remoteUsers, localUsers, mode, remote, local)}
			e.finish(&result)
			require.Len(t, result.Users, 2)
			require.True(t, result.Attempts[0].SelectedInput)
			require.True(t, result.Attempts[1].SelectedInput)
			if mode == ModeLocalFirst {
				require.Equal(t, 1, result.Attempts[0].WinningRows)
				require.Equal(t, 1, result.Attempts[1].WinningRows)
			} else {
				require.Equal(t, 2, result.Attempts[0].WinningRows)
				require.Zero(t, result.Attempts[1].WinningRows)
			}
		})
	}
}
