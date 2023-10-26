package copier

import (
	"archive/tar"
	"io/fs"
)

type suppressedSysIDsFileInfo struct {
	fs.FileInfo
}

func noLookupFileInfoHeader(fileinfo fs.FileInfo, symlinkTarget string) (*tar.Header, error) {
	hdr, err := tar.FileInfoHeader(suppressedSysIDsFileInfo{fileinfo}, symlinkTarget)
	if err == nil {
		copySysIDs(hdr, fileinfo)
	}
	return hdr, err
}
