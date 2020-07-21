// +build windows

package copier

import (
	"os"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

var canChroot = false

func chroot(path string) (bool, error) {
	return false, nil
}

func getcwd() (string, error) {
	return windows.Getwd()
}

func chrMode(mode os.FileMode) uint32 {
	return windows.S_IFCHR | uint32(mode)
}

func blkMode(mode os.FileMode) uint32 {
	return windows.S_IFBLK | uint32(mode)
}

func mkdev(major, minor uint32) uint64 {
	return 0
}

func mkfifo(path string, mode uint32) error {
	return syscall.ENOSYS
}

func mknod(path string, mode uint32, dev int) error {
	return syscall.ENOSYS
}

func lutimes(isSymlink bool, path string, atime, mtime time.Time) error {
	if isSymlink {
		return nil
	}
	if atime.IsZero() || mtime.IsZero() {
		now := time.Now()
		if atime.IsZero() {
			atime = now
		}
		if mtime.IsZero() {
			mtime = now
		}
	}
	return windows.UtimesNano(path, []windows.Timespec{windows.NsecToTimespec(atime.UnixNano()), windows.NsecToTimespec(mtime.UnixNano())})
}
