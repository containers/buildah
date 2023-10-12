package tmpdir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/common/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTempDir(t *testing.T) {
	// test default
	err := os.Unsetenv("TMPDIR")
	require.NoError(t, err)
	err = os.Setenv("CONTAINERS_CONF", "/dev/null")
	require.NoError(t, err)
	tmpdir := GetTempDir()
	assert.Equal(t, "/var/tmp", tmpdir)

	// test TMPDIR Environment
	err = os.Setenv("TMPDIR", "/tmp/bogus")
	require.NoError(t, err)
	tmpdir = GetTempDir()
	assert.Equal(t, tmpdir, "/tmp/bogus")
	err = os.Unsetenv("TMPDIR")
	require.NoError(t, err)

	// relative TMPDIR should be automatically converted to absolute
	err = os.Setenv("TMPDIR", ".")
	require.NoError(t, err)
	tmpdir = GetTempDir()
	assert.True(t, filepath.IsAbs(tmpdir), "path from GetTempDir should always be absolute")
	err = os.Unsetenv("TMPDIR")
	require.NoError(t, err)

	f, err := os.CreateTemp("", "containers.conf-")
	require.NoError(t, err)
	// close and remove the temporary file at the end of the program
	defer f.Close()
	defer os.Remove(f.Name())
	data := []byte("[engine]\nimage_copy_tmp_dir=\"/mnt\"\n")
	_, err = f.Write(data)
	require.NoError(t, err)

	err = os.Setenv("CONTAINERS_CONF", f.Name())
	require.NoError(t, err)
	// force config reset of default containers.conf
	options := config.Options{
		SetDefault: true,
	}
	_, err = config.New(&options)
	require.NoError(t, err)
	tmpdir = GetTempDir()
	assert.Equal(t, "/mnt", tmpdir)

}
