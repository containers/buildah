//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package copier

import (
	"archive/tar"
	"io/fs"
	"syscall"
)

func copySysIDs(hdr *tar.Header, fileinfo fs.FileInfo) {
	sys := fileinfo.Sys()
	if st, ok := sys.(*syscall.Stat_t); ok {
		hdr.Uid = int(st.Uid)
		hdr.Gid = int(st.Gid)
		hdr.Uname, hdr.Gname = "", ""
	}
}

func (nsfi suppressedSysIDsFileInfo) Sys() any {
	sys := nsfi.FileInfo.Sys()
	if st, ok := sys.(*syscall.Stat_t); ok {
		copied := *st
		copied.Uid, copied.Gid = 0, 0
		return &copied
	}
	return sys
}
