package identity

import (
	"github.com/soulteary/warden/internal/define"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIdentityDiagnosticMaskBoundaries(t *testing.T) {
	require.Empty(t, MaskPhone(""))
	require.Equal(t, "**", MaskPhone("12"))
	require.Equal(t, "*", MaskMail("x"))
	require.Equal(t, "b**", MaskMail("bad"))
	require.Equal(t, "a*@example.com", MaskMail("a@example.com"))
	require.Empty(t, MaskUserID(""))
	require.Equal(t, "**", MaskUserID("ab"))
	result, err := ValidateAndIndexUsers([]define.AllowListUser{{Phone: "13800138000", UserID: "a"}, {Phone: "13800138000", UserID: "b"}, {Mail: "a@example.com", UserID: "c"}, {Mail: "a@example.com", UserID: "d"}, {Mail: "b@example.com", UserID: "c"}}, Options{ReportOnly: true})
	require.NoError(t, err)
	require.Equal(t, 3, result.ConflictCount)
	errMissing := &MissingUserIDError{Index: 1, MaskedIdentifier: "masked"}
	require.ErrorIs(t, errMissing, ErrMissingUserID)
	require.Contains(t, errMissing.Error(), "masked")
	users := stableSort([]define.AllowListUser{{Phone: "2", Mail: "b@example.com"}, {Phone: "1", Mail: "z@example.com"}, {Phone: "1", Mail: "a@example.com"}})
	require.Equal(t, "a@example.com", users[0].Mail)
	require.Equal(t, "z@example.com", users[1].Mail)
}
