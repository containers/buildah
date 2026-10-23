package tmpdir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.podman.io/common/pkg/config"
)

func TestGetTempDir(t *testing.T) {
	// test default
	err := os.Unsetenv("TMPDIR")
	require.NoError(t, err)
	t.Setenv("CONTAINERS_CONF", "/dev/null")
	tmpdir := GetTempDir()
	assert.Equal(t, "/var/tmp", tmpdir)

	// test TMPDIR Environment
	t.Setenv("TMPDIR", "/tmp/bogus")
	tmpdir = GetTempDir()
	assert.Equal(t, tmpdir, "/tmp/bogus")
	err = os.Unsetenv("TMPDIR")
	require.NoError(t, err)

	// relative TMPDIR should be automatically converted to absolute
	t.Setenv("TMPDIR", ".")
	tmpdir = GetTempDir()
	assert.True(t, filepath.IsAbs(tmpdir), "path from GetTempDir should always be absolute")
	err = os.Unsetenv("TMPDIR")
	require.NoError(t, err)

	f, err := os.CreateTemp(t.TempDir(), "containers.conf-")
	require.NoError(t, err)
	// close and remove the temporary file at the end of the program
	defer f.Close()
	defer os.Remove(f.Name())
	data := []byte("[engine]\nimage_copy_tmp_dir=\"/mnt\"\n")
	_, err = f.Write(data)
	require.NoError(t, err)

	t.Setenv("CONTAINERS_CONF", f.Name())
	// force config reset of default containers.conf
	options := config.Options{
		SetDefault: true,
	}
	_, err = config.New(&options)
	require.NoError(t, err)
	tmpdir = GetTempDir()
	assert.Equal(t, "/mnt", tmpdir)
}

func TestGetTempDirDoesNotCreateTMPDIR(t *testing.T) {
	path := filepath.Join(t.TempDir(), "caller-managed")
	t.Setenv("TMPDIR", path)
	assert.Equal(t, path, GetTempDir())
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}
