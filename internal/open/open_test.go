package open

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func TestOpenInChroot(t *testing.T) {
	t.Parallel()
	tmpdir := t.TempDir()
	firstContents := []byte{0, 1, 2, 3}
	secondContents := []byte{4, 5, 6, 7}
	require.NoErrorf(t, os.WriteFile(filepath.Join(tmpdir, "a"), firstContents, 0o644), "creating first test file")
	require.NoErrorf(t, os.MkdirAll(filepath.Join(tmpdir, tmpdir), 0o755), "creating test subdirectory")
	require.NoErrorf(t, os.WriteFile(filepath.Join(tmpdir, tmpdir, "a"), secondContents, 0o644), "creating second test file")

	result := inChroot(requests{
		Open: []request{
			{
				Path: filepath.Join(tmpdir, "a"),
				Mode: unix.O_RDONLY,
			},
		},
	})
	require.Empty(t, result.Err, "result from first client")
	require.Equal(t, 1, len(result.Open), "results from first client")
	require.Empty(t, result.Open[0].Err, "first (only) result from first client")
	f := os.NewFile(result.Open[0].Fd, "file from first subprocess")
	contents, err := io.ReadAll(f)
	require.NoErrorf(t, err, "reading from file from first subprocess")
	require.Equalf(t, firstContents, contents, "contents of file from first subprocess")
	f.Close()

	result = inChroot(requests{
		Root: tmpdir,
		Open: []request{
			{
				Path: filepath.Join(tmpdir, "a"),
				Mode: unix.O_RDONLY,
			},
		},
	})
	require.Empty(t, result.Err, "result from second client")
	require.Equal(t, 1, len(result.Open), "results from second client")
	require.Empty(t, result.Open[0].Err, "first (only) result from second client")
	f = os.NewFile(result.Open[0].Fd, "file from second subprocess")
	contents, err = io.ReadAll(f)
	require.NoErrorf(t, err, "reading from file from second subprocess")
	require.Equalf(t, secondContents, contents, "contents of file from second subprocess")
	f.Close()

	fd, errno, err := InChroot(tmpdir, "", filepath.Join(tmpdir, "a"), unix.O_RDONLY, 0)
	require.NoErrorf(t, err, "wrapper for opening just one item")
	require.Zero(t, errno, "errno from open file")
	f = os.NewFile(uintptr(fd), "file from third subprocess")
	require.NoErrorf(t, err, "reading from file from third subprocess")
	require.Equalf(t, secondContents, contents, "contents of file from third subprocess")
	f.Close()

	fd, errno, err = InChroot(tmpdir, "", filepath.Join(tmpdir, "b"), unix.O_RDONLY, 0)
	require.Errorf(t, err, "attempting to open a non-existent file")
	require.NotZero(t, errno, "attempting to open a non-existent file")
	require.Equal(t, -1, fd, "returned descriptor when open fails")
	f.Close()
}
