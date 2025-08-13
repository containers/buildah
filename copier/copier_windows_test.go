//go:build windows

package copier

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func checkStatInfoOwnership(t *testing.T, result *StatForItem) {
	t.Helper()
	require.EqualValues(t, -1, result.UID, "expected the owning user to not be supported")
	require.EqualValues(t, -1, result.GID, "expected the owning group to not be supported")
}
