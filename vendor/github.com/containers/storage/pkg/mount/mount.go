package mount

import (
	"sort"
	"strings"
	"time"

	"github.com/containers/storage/pkg/fileutils"
)

// GetMounts retrieves a list of mounts for the current running process.
func GetMounts() ([]*Info, error) {
	return parseMountTable()
}

// Mounted determines if a specified mountpoint has been mounted.
// On Linux it looks at /proc/self/mountinfo and on Solaris at mnttab.
func Mounted(mountpoint string) (bool, error) {
	entries, err := parseMountTable()
	if err != nil {
		return false, err
	}

	mountpoint, err = fileutils.ReadSymlinkedDirectory(mountpoint)
	if err != nil {
		return false, err
	}
	// Search the table for the mountpoint
	for _, e := range entries {
		if e.Mountpoint == mountpoint {
			return true, nil
		}
	}
	return false, nil
}

// Mount will mount filesystem according to the specified configuration, on the
// condition that the target path is *not* already mounted. Options must be
// specified like the mount or fstab unix commands: "opt1=val1,opt2=val2". See
// flags.go for supported option flags.
func Mount(device, target, mType, options string) error {
	flag, _ := ParseOptions(options)
	if flag&REMOUNT != REMOUNT {
		if mounted, err := Mounted(target); err != nil || mounted {
			return err
		}
	}
	return ForceMount(device, target, mType, options)
}

// ForceMount will mount a filesystem according to the specified configuration,
// *regardless* if the target path is not already mounted. Options must be
// specified like the mount or fstab unix commands: "opt1=val1,opt2=val2". See
// flags.go for supported option flags.
func ForceMount(device, target, mType, options string) error {
	flag, data := ParseOptions(options)
	return mount(device, target, mType, uintptr(flag), data)
}

// Unmount lazily unmounts a filesystem on supported platforms, otherwise
// does a normal unmount.
func Unmount(target string) error {
	if mounted, err := Mounted(target); err != nil || !mounted {
		return err
	}
	return ForceUnmount(target)
}

// RecursiveUnmount unmounts the target and all mounts underneath, starting with
// the deepsest mount first.
func RecursiveUnmount(target string) error {
	mounts, err := GetMounts()
	if err != nil {
		return err
	}

	// Make the deepest mount be first
	sort.Sort(sort.Reverse(byMountpoint(mounts)))

	for i, m := range mounts {
		if !strings.HasPrefix(m.Mountpoint, target) {
			continue
		}
		if err := Unmount(m.Mountpoint); err != nil && i == len(mounts)-1 {
			if mounted, err := Mounted(m.Mountpoint); err != nil || mounted {
				return err
			}
			// Ignore errors for submounts and continue trying to unmount others
			// The final unmount should fail if there ane any submounts remaining
		}
	}
	return nil
}

// ForceUnmount will force an unmount of the target filesystem, regardless if
// it is mounted or not.
func ForceUnmount(target string) (err error) {
	// Simple retry logic for unmount
	for i := 0; i < 10; i++ {
		if err = unmount(target, 0); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}
