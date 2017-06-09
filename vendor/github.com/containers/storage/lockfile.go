package storage

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/containers/storage/pkg/stringid"
)

// A Locker represents a file lock where the file is used to cache an
// identifier of the last party that made changes to whatever's being protected
// by the lock.
type Locker interface {
	sync.Locker

	// Touch records, for others sharing the lock, that the caller was the
	// last writer.  It should only be called with the lock held.
	Touch() error

	// Modified() checks if the most recent writer was a party other than the
	// last recorded writer.  It should only be called with the lock held.
	Modified() (bool, error)

	// TouchedSince() checks if the most recent writer modified the file (likely using Touch()) after the specified time.
	TouchedSince(when time.Time) bool
}

type lockfile struct {
	mu   sync.Mutex
	file string
	fd   uintptr
	lw   string
}

var (
	lockfiles     map[string]*lockfile
	lockfilesLock sync.Mutex
)

// GetLockfile opens a lock file, creating it if necessary.  The Locker object
// return will be returned unlocked.
func GetLockfile(path string) (Locker, error) {
	lockfilesLock.Lock()
	defer lockfilesLock.Unlock()
	if lockfiles == nil {
		lockfiles = make(map[string]*lockfile)
	}
	if locker, ok := lockfiles[filepath.Clean(path)]; ok {
		return locker, nil
	}
	fd, err := unix.Open(filepath.Clean(path), os.O_RDWR|os.O_CREATE, unix.S_IRUSR|unix.S_IWUSR)
	if err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	locker := &lockfile{file: path, fd: uintptr(fd), lw: stringid.GenerateRandomID()}
	lockfiles[filepath.Clean(path)] = locker
	return locker, nil
}

func (l *lockfile) Lock() {
	lk := unix.Flock_t{
		Type:   unix.F_WRLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
	}
	l.mu.Lock()
	for unix.FcntlFlock(l.fd, unix.F_SETLKW, &lk) != nil {
		time.Sleep(10 * time.Millisecond)
	}
}

func (l *lockfile) Unlock() {
	lk := unix.Flock_t{
		Type:   unix.F_UNLCK,
		Whence: int16(os.SEEK_SET),
		Start:  0,
		Len:    0,
		Pid:    int32(os.Getpid()),
	}
	for unix.FcntlFlock(l.fd, unix.F_SETLKW, &lk) != nil {
		time.Sleep(10 * time.Millisecond)
	}
	l.mu.Unlock()
}

func (l *lockfile) Touch() error {
	l.lw = stringid.GenerateRandomID()
	id := []byte(l.lw)
	_, err := unix.Seek(int(l.fd), 0, os.SEEK_SET)
	if err != nil {
		return err
	}
	n, err := unix.Write(int(l.fd), id)
	if err != nil {
		return err
	}
	if n != len(id) {
		return unix.ENOSPC
	}
	err = unix.Fsync(int(l.fd))
	if err != nil {
		return err
	}
	return nil
}

func (l *lockfile) Modified() (bool, error) {
	id := []byte(l.lw)
	_, err := unix.Seek(int(l.fd), 0, os.SEEK_SET)
	if err != nil {
		return true, err
	}
	n, err := unix.Read(int(l.fd), id)
	if err != nil {
		return true, err
	}
	if n != len(id) {
		return true, unix.ENOSPC
	}
	lw := l.lw
	l.lw = string(id)
	return l.lw != lw, nil
}

func (l *lockfile) TouchedSince(when time.Time) bool {
	st := unix.Stat_t{}
	err := unix.Fstat(int(l.fd), &st)
	if err != nil {
		return true
	}
	touched := time.Unix(statTMtimeUnix(st))
	return when.Before(touched)
}
