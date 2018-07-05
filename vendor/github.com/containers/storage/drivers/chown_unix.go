// +build !windows

package graphdriver

import (
	"fmt"
	"os"
	"syscall"

	"github.com/containers/storage/pkg/idtools"
)

func platformLChown(path string, info os.FileInfo, toHost, toContainer *idtools.IDMappings) error {
	sysinfo := info.Sys()
	if st, ok := sysinfo.(*syscall.Stat_t); ok {
		// Map an on-disk UID/GID pair from host to container
		// using the first map, then back to the host using the
		// second map.  Skip that first step if they're 0, to
		// compensate for cases where a parent layer should
		// have had a mapped value, but didn't.
		uid, gid := int(st.Uid), int(st.Gid)
		if toContainer != nil {
			pair := idtools.IDPair{
				UID: uid,
				GID: gid,
			}
			mappedUid, mappedGid, err := toContainer.ToContainer(pair)
			if err != nil {
				if (uid != 0) || (gid != 0) {
					return fmt.Errorf("error mapping host ID pair %#v for %q to container: %v", pair, path, err)
				}
				mappedUid, mappedGid = uid, gid
			}
			uid, gid = mappedUid, mappedGid
		}
		if toHost != nil {
			pair := idtools.IDPair{
				UID: uid,
				GID: gid,
			}
			mappedPair, err := toHost.ToHost(pair)
			if err != nil {
				return fmt.Errorf("error mapping container ID pair %#v for %q to host: %v", pair, path, err)
			}
			uid, gid = mappedPair.UID, mappedPair.GID
		}
		if uid != int(st.Uid) || gid != int(st.Gid) {
			// Make the change.
			if err := syscall.Lchown(path, uid, gid); err != nil {
				return fmt.Errorf("%s: chown(%q): %v", os.Args[0], path, err)
			}
		}
	}
	return nil
}
