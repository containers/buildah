// +build cgo
// +build linux

package buildah

// #include <sys/types.h>
// #include <grp.h>
// #include <pwd.h>
// #include <stdlib.h>
// #include <stdio.h>
// #include <string.h>
// typedef FILE * pFILE;
import "C"

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/unix"
)

func fopenContainerFileWork(rootdir, filename string, linkCount int) (string, C.pFILE, error) {
	// If we've recursed too far, return an error.
	if linkCount >= 64 {
		return "", nil, fmt.Errorf("too many links encountered looking up %q", filename)
	}

	// Do a bit of lexical cleanup on the filename.
	filename = filepath.Clean(filename)

	// Start the lookup at the chroot directory.
	rootfd, err := unix.Open(rootdir, unix.O_DIRECTORY|unix.O_PATH, 0)
	if err != nil {
		return "", nil, err
	}
	defer unix.Close(rootfd)

	// Start resolving the pathname.
	components := strings.Split(filepath.Clean(filename), string(os.PathSeparator))
	if len(components) > 0 && components[0] == "" {
		components = components[1:]
	}
	if len(components) == 0 {
		return "", nil, fmt.Errorf("no filename to open")
	}
	remaining := components
	fd := rootfd
	parents := []string{}

	for _, comp := range components {
		var st unix.Stat_t
		remaining = remaining[1:]

		// Open the next component, relative to the previous one.
		subfd, err := unix.Openat(fd, comp, unix.O_PATH|unix.O_NOFOLLOW, 0)
		if err != nil {
			return "", nil, err
		}
		defer unix.Close(subfd)

		// Check what sort of item this component is.
		err = unix.Fstat(subfd, &st)
		if err != nil {
			return "", nil, err
		}

		// It's a subdirectory.  That's fine.  Recompute the filename for display in case we have no more path components.
		if st.Mode&unix.S_IFDIR == unix.S_IFDIR {
			fd = subfd
			if len(remaining) == 0 {
				filename = filepath.Join(append(parents, comp)...)
				return "", nil, fmt.Errorf("%q is a directory", filepath.Join(rootdir, filename))
			}
			parents = append(parents, comp)
			continue
		}

		// It's a symbolic link.  Read it.
		if st.Mode&unix.S_IFLNK == unix.S_IFLNK {
			link := make([]byte, unix.PathMax)
			n, err := unix.Readlinkat(subfd, "", link)
			if err != nil {
				return "", nil, err
			}
			if n > len(link) {
				return "", nil, fmt.Errorf("symlink value too long")
			}
			link = link[:n]
			// Compute what the resulting path, still relative to rootdir, would look like.
			newfilename := ""
			if filepath.IsAbs(string(link)) {
				// It looks like the absolute link's destination and whatever path components were left.
				newfilename = filepath.Join(append([]string{string(link)}, remaining...)...)
			} else {
				// It looks like a name computed from the parent components, the link contents, and
				// whatever path components were left.  Prepend/ a "/" so that Clean() will treat this
				// as an absolute path, and lexically remove any leading "/.." elements.
				newfilename = filepath.Clean("/" + filepath.Join(append(append(parents, string(link)), remaining...)...))
			}
			// Start over with that resulting path as our input.
			return newfilename, nil, nil
		}

		// If it's not a directory or a link, and we still have components, it's an invalid path.
		if len(remaining) > 0 {
			return "", nil, fmt.Errorf("%q is not a directory", filepath.Join(rootdir, filepath.Join(append(parents, comp)...)))
		}

		// It's an ordinary file.  Recompute the filename for display and open the file for real, and we're done.
		fd, err = unix.Openat(fd, comp, unix.O_NOFOLLOW, 0)
		if err != nil {
			return "", nil, err
		}
		filename = filepath.Join(append(parents, comp)...)
	}

	// Actually open the file for reading.
	mode := C.CString("r")
	defer C.free(unsafe.Pointer(mode))
	f, err := C.fdopen(C.int(fd), mode)
	if f == nil || err != nil {
		return "", nil, fmt.Errorf("error opening %q/%q: %v", rootdir, filename, err)
	}
	return filename, f, nil
}

func fopenContainerFile(rootdir, filename string) (C.pFILE, error) {
	symlinksEncountered := 0
	newfilename, file, err := fopenContainerFileWork(rootdir, filename, symlinksEncountered)
	for file == nil && err == nil {
		symlinksEncountered++
		newfilename, file, err = fopenContainerFileWork(rootdir, newfilename, symlinksEncountered)
	}
	return file, err
}

var (
	lookupUser, lookupGroup sync.Mutex
)

func lookupUserInContainer(rootdir, username string) (uint64, uint64, error) {
	name := C.CString(username)
	defer C.free(unsafe.Pointer(name))

	f, err := fopenContainerFile(rootdir, "/etc/passwd")
	if err != nil {
		return 0, 0, err
	}
	defer C.fclose(f)

	lookupUser.Lock()
	defer lookupUser.Unlock()

	pwd := C.fgetpwent(f)
	for pwd != nil {
		if C.strcmp(pwd.pw_name, name) != 0 {
			pwd = C.fgetpwent(f)
			continue
		}
		return uint64(pwd.pw_uid), uint64(pwd.pw_gid), nil
	}

	return 0, 0, user.UnknownUserError(fmt.Sprintf("error looking up user %q", username))
}

func lookupGroupForUIDInContainer(rootdir string, userid uint64) (string, uint64, error) {
	f, err := fopenContainerFile(rootdir, "/etc/passwd")
	if err != nil {
		return "", 0, err
	}
	defer C.fclose(f)

	lookupUser.Lock()
	defer lookupUser.Unlock()

	pwd := C.fgetpwent(f)
	for pwd != nil {
		if uint64(pwd.pw_uid) != userid {
			pwd = C.fgetpwent(f)
			continue
		}
		return C.GoString(pwd.pw_name), uint64(pwd.pw_gid), nil
	}

	return "", 0, user.UnknownUserError(fmt.Sprintf("error looking up user with UID %d", userid))
}

func lookupGroupInContainer(rootdir, groupname string) (uint64, error) {
	name := C.CString(groupname)
	defer C.free(unsafe.Pointer(name))

	f, err := fopenContainerFile(rootdir, "/etc/group")
	if err != nil {
		return 0, err
	}
	defer C.fclose(f)

	lookupGroup.Lock()
	defer lookupGroup.Unlock()

	grp := C.fgetgrent(f)
	for grp != nil {
		if C.strcmp(grp.gr_name, name) != 0 {
			grp = C.fgetgrent(f)
			continue
		}
		return uint64(grp.gr_gid), nil
	}

	return 0, user.UnknownGroupError(fmt.Sprintf("error looking up group %q", groupname))
}
