package mkcw

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadWriteWorkloadConfig(t *testing.T) {
	// Create a temporary file to stand in for a disk image.
	temp := filepath.Join(t.TempDir(), "disk.img")
	f, err := os.OpenFile(temp, os.O_CREATE|os.O_RDWR, 0o600)
	require.NoError(t, err)
	err = f.Truncate(0x1000000)
	require.NoError(t, err)
	defer f.Close()

	// Generate a random "encoded workload config".
	workloadConfig := make([]byte, 0x100)
	n, err := rand.Read(workloadConfig)
	require.NoError(t, err)
	require.Equal(t, len(workloadConfig), n)

	// Read the size of our temporary file.
	st, err := f.Stat()
	require.NoError(t, err)
	originalSize := st.Size()

	// Should get an error, since there's no workloadConfig in there to read.
	_, err = ReadWorkloadConfigFromImage(f.Name())
	require.Error(t, err)

	// File should grow, even though we looked for an old config to overwrite.
	err = WriteWorkloadConfigToImage(f, workloadConfig, true)
	require.NoError(t, err)
	st, err = f.Stat()
	require.NoError(t, err)
	require.Greater(t, st.Size(), originalSize)
	originalSize = st.Size()

	// File shouldn't grow, even overwriting the config with a slightly larger one.
	err = WriteWorkloadConfigToImage(f, append([]byte("slightly longer"), workloadConfig...), true)
	require.NoError(t, err)
	st, err = f.Stat()
	require.NoError(t, err)
	require.Equal(t, originalSize, st.Size())
	originalSize = st.Size()

	// File should grow if we're not trying to replace an old one config with a new one.
	err = WriteWorkloadConfigToImage(f, []byte("{\"comment\":\"quite a bit shorter\"}"), false)
	require.NoError(t, err)
	st, err = f.Stat()
	require.NoError(t, err)
	require.Greater(t, st.Size(), originalSize)

	// Should read successfully.
	_, err = ReadWorkloadConfigFromImage(f.Name())
	require.NoError(t, err)
}
