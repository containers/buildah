package imagebuildah

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePathChecksum(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()

	tempFile, err := os.CreateTemp(tempDir, "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	// Write some data to the file
	data := []byte("Hello, world!")
	if _, err := tempFile.Write(data); err != nil {
		t.Fatalf("Failed to write data to temp file: %v", err)
	}

	// Generate the checksum for the directory
	checksum, err := generatePathChecksum(tempDir)
	if err != nil {
		t.Fatalf("Failed to generate checksum: %v", err)
	}

	digester := digest.SHA256.Digester()
	tarWriter := tar.NewWriter(digester.Hash())

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
	if err != nil {
		tarWriter.Close()
		t.Fatalf("Failed to manually generate checksum: %v", err)
	}

	tarWriter.Close()
	expectedChecksum := digester.Digest().String()

	// Compare the generated checksum to the expected checksum
	assert.Equal(t, expectedChecksum, checksum, "didn't get expected checksum over a sample directory")
}
