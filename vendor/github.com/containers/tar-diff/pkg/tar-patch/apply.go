package tar_patch

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/containers/tar-diff/pkg/common"
	"github.com/klauspost/compress/zstd"
	"io"
	"os"
	"path"
)

type DataSource interface {
	io.ReadSeeker
	io.Closer
	SetCurrentFile(file string) error
}

type FilesystemDataSource struct {
	basePath    string
	currentFile *os.File
}

// Cleans up the path lexically
// Any ".." that extends outside the first elements (or the root itself) is invalid and returns ""
func cleanPath(pathName string) string {
	// We make the path always absolute, that way path.Clean() ensure it never goes outside the top ("root") dir
	// even if its a relative path
	clean := path.Clean("/" + pathName)

	// We clean the initial slash, making all result relative (or "" which is error)
	return clean[1:]
}

func NewFilesystemDataSource(basePath string) *FilesystemDataSource {
	return &FilesystemDataSource{
		basePath:    basePath,
		currentFile: nil,
	}
}

func (f *FilesystemDataSource) Close() error {
	if f.currentFile != nil {
		err := f.currentFile.Close()
		f.currentFile = nil

		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FilesystemDataSource) Read(data []byte) (n int, err error) {
	if f.currentFile == nil {
		return 0, fmt.Errorf("No current file set")
	}
	return f.currentFile.Read(data)
}

func (f *FilesystemDataSource) SetCurrentFile(file string) error {
	if f.currentFile != nil {
		err := f.currentFile.Close()
		f.currentFile = nil
		if err != nil {
			return nil
		}
	}
	currentFile, err := os.Open(f.basePath + "/" + file)
	if err != nil {
		return err
	}
	f.currentFile = currentFile
	return nil
}

func (f *FilesystemDataSource) Seek(offset int64, whence int) (int64, error) {
	if f.currentFile == nil {
		return 0, fmt.Errorf("No current file set")
	}
	return f.currentFile.Seek(offset, whence)
}

func Apply(delta io.Reader, dataSource DataSource, dst io.Writer) error {
	buf := make([]byte, len(common.DeltaHeader))
	_, err := io.ReadFull(delta, buf)
	if err != nil {
		return err
	}
	if !bytes.Equal(buf, common.DeltaHeader[:]) {
		return fmt.Errorf("Invalid delta format")
	}

	decoder, err := zstd.NewReader(delta)
	if err != nil {
		return err
	}
	defer decoder.Close()

	r := bufio.NewReader(decoder)

	for {
		op, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		size, err := binary.ReadUvarint(r)
		if err != nil {
			return err
		}

		switch op {
		case common.DeltaOpData:
			_, err = io.CopyN(dst, r, int64(size))
			if err != nil {
				return err
			}
		case common.DeltaOpOpen:
			nameBytes := make([]byte, size)
			_, err = io.ReadFull(r, nameBytes)
			if err != nil {
				return err
			}
			name := string(nameBytes)
			cleanName := cleanPath(name)
			if len(cleanName) == 0 {
				return fmt.Errorf("Invalid source name '%v' in tar-diff", name)
			}
			err := dataSource.SetCurrentFile(cleanName)
			if err != nil {
				return err
			}
		case common.DeltaOpCopy:
			_, err = io.CopyN(dst, dataSource, int64(size))
			if err != nil {
				return err
			}
		case common.DeltaOpAddData:
			addBytes := make([]byte, size)
			_, err = io.ReadFull(r, addBytes)
			if err != nil {
				return err
			}

			addBytes2 := make([]byte, size)
			_, err = io.ReadFull(dataSource, addBytes2)
			if err != nil {
				return err
			}

			for i := uint64(0); i < size; i++ {
				addBytes[i] = addBytes[i] + addBytes2[i]
			}
			if _, err := dst.Write(addBytes); err != nil {
				return err
			}

		case common.DeltaOpSeek:
			_, err = dataSource.Seek(int64(size), 0)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unexpected delta op %d", op)
		}
	}

	return nil
}
