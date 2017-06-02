// +build !cgo !linux

package buildah

import (
	"fmt"
)

func lookupUserInContainer(rootdir, username string) (uint64, uint64, error) {
	return 0, 0, errors.Wrapf("user lookup not supported")
}

func lookupGroupInContainer(rootdir, groupname string) (uint64, error) {
	return 0, errors.Wrapf("group lookup not supported")
}

func lookupGroupForUIDInContainer(rootdir string, userid uint64) (string, uint64, error) {
	return "", 0, errors.Wrapf("primary group lookup by uid not supported")
}
