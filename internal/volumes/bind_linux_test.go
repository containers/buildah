package volumes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/storage/pkg/mount"
	"github.com/stretchr/testify/require"
)

func TestBindFromChroot(t *testing.T) {
	t.Parallel()
	if os.Getuid() != 0 {
		t.Skip("not running as root, assuming we can't mount or chroot")
	}
	contents1 := "file1"
	contents2 := "file2"
	rootdir := t.TempDir()
	destdir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(rootdir, "subdirectory"), 0o700), "creating bind mount source directory")
	require.NoError(t, os.WriteFile(filepath.Join(rootdir, "subdirectory", "file"), []byte(contents1), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(rootdir, "file"), []byte(contents2), 0o600))
	subdir, err := bindFromChroot(rootdir, "subdirectory", destdir)
	require.NoError(t, err, "bind mounting from a directory")
	bytes1, err := os.ReadFile(filepath.Join(subdir, "file"))
	require.NoError(t, err, "reading file from bind-mounted directory")
	subfile, err := bindFromChroot(rootdir, "file", destdir)
	require.NoError(t, err, "bind mounting from a file")
	bytes2, err := os.ReadFile(subfile)
	require.NoError(t, err, "reading file from bind mounted file")
	require.Equal(t, contents1, string(bytes1), "contents of file in bind-mounted directory")
	require.Equal(t, contents2, string(bytes2), "contents of bind-mounted file")
	require.NoError(t, mount.Unmount(subdir), "unmounting bind-mounted directory")
	require.NoError(t, mount.Unmount(subfile), "unmounting bind-mounted file")
}
