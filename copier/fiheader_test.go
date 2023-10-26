//go:build linux || darwin || freebsd

package copier

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoLookupFileInfoHeader(t *testing.T) {
	tempDir := t.TempDir()

	f1, err := os.CreateTemp(tempDir, "")
	require.NoError(t, err, "while creating first temporary file")
	defer f1.Close()
	err = f1.Chown(1, 2)
	require.NoError(t, err, "while chowning first temporary file")
	err = f1.Sync()
	require.NoError(t, err, "while syncing first temporary file")

	f2, err := os.CreateTemp(tempDir, "")
	require.NoError(t, err, "while creating second temporary file")
	defer f2.Close()
	err = f2.Chown(3, 4)
	require.NoError(t, err, "while chowning second temporary file")
	err = f2.Sync()
	require.NoError(t, err, "while syncing second temporary file")

	f1i, err := f1.Stat()
	require.NoError(t, err, "statting first temporary file")
	h1, err := noLookupFileInfoHeader(f1i, "")
	require.NoError(t, err, "generating header for first temporary file")
	f2i, err := f2.Stat()
	require.NoError(t, err, "statting second temporary file")
	h2, err := noLookupFileInfoHeader(f2i, "")
	require.NoError(t, err, "generating header for second temporary file")

	assert.NotEqual(t, 0, h1.Uid, "user owner of first temporary file should be not zero")
	assert.NotEqual(t, 0, h2.Uid, "user owner of second temporary file should be not zero")
	assert.NotEqual(t, 0, h1.Gid, "group owner of first temporary file should be not zero")
	assert.NotEqual(t, 0, h2.Gid, "group owner of second temporary file should be not zero")
	assert.Equal(t, "", h1.Uname, "user owner of first temporary file should have been left blank")
	assert.Equal(t, "", h2.Uname, "user owner of second temporary file should have been left blank")
	assert.Equal(t, "", h1.Gname, "group owner of first temporary file should have been left blank")
	assert.Equal(t, "", h2.Gname, "group owner of second temporary file should have been left blank")
}
