package open

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestBindFdToPath(t *testing.T) {
	t.Parallel()
	first := t.TempDir()
	sampleData := []byte("sample data")
	err := os.WriteFile(filepath.Join(first, "testfile"), sampleData, 0o600)
	require.NoError(t, err, "writing sample data to first directory")
	fd, err := unix.Open(first, unix.O_DIRECTORY, 0)
	require.NoError(t, err, "opening descriptor for first directory")
	second := t.TempDir()
	err = BindFdToPath(uintptr(fd), second)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := unix.Unmount(second, unix.MNT_DETACH)
		require.NoError(t, err, "unmounting as part of cleanup")
	})
	readBack, err := os.ReadFile(filepath.Join(second, "testfile"))
	require.NoError(t, err)
	require.Equal(t, sampleData, readBack, "expected to read back data via the bind mount")
}
