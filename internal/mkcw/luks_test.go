package mkcw

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/luksy"
	"github.com/stretchr/testify/require"
)

func TestCheckLUKSPassphrase(t *testing.T) {
	passphrase, err := GenerateDiskEncryptionPassphrase()
	require.NoError(t, err)
	secondPassphrase, err := GenerateDiskEncryptionPassphrase()
	require.NoError(t, err)

	t.Run("v1", func(t *testing.T) {
		header, encrypter, blockSize, err := luksy.EncryptV1([]string{secondPassphrase, passphrase}, "")
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(t.TempDir(), "v1"))
		require.NoError(t, err)
		n, err := f.Write(header)
		require.NoError(t, err)
		require.Equal(t, len(header), n)
		wrapper := luksy.EncryptWriter(encrypter, f, blockSize)
		_, err = wrapper.Write(make([]byte, blockSize*10))
		require.NoError(t, err)
		wrapper.Close()
		f.Close()

		err = CheckLUKSPassphrase(f.Name(), passphrase)
		require.NoError(t, err)
		err = CheckLUKSPassphrase(f.Name(), secondPassphrase)
		require.NoError(t, err)
		err = CheckLUKSPassphrase(f.Name(), "nope, this is not a correct passphrase")
		require.Error(t, err)
	})

	t.Run("v2", func(t *testing.T) {
		for _, sectorSize := range []int{512, 1024, 2048, 4096} {
			t.Run(fmt.Sprintf("sectorSize=%d", sectorSize), func(t *testing.T) {
				header, encrypter, blockSize, err := luksy.EncryptV2([]string{secondPassphrase, passphrase}, "", sectorSize)
				require.NoError(t, err)
				f, err := os.Create(filepath.Join(t.TempDir(), "v2"))
				require.NoError(t, err)
				n, err := f.Write(header)
				require.NoError(t, err)
				require.Equal(t, len(header), n)
				wrapper := luksy.EncryptWriter(encrypter, f, blockSize)
				_, err = wrapper.Write(make([]byte, blockSize*10))
				require.NoError(t, err)
				wrapper.Close()
				f.Close()

				err = CheckLUKSPassphrase(f.Name(), passphrase)
				require.NoError(t, err)
				err = CheckLUKSPassphrase(f.Name(), secondPassphrase)
				require.NoError(t, err)
				err = CheckLUKSPassphrase(f.Name(), "nope, this is not one of the correct passphrases")
				require.Error(t, err)
			})
		}
	})
}
