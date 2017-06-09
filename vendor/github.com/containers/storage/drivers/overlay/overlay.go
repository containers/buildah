// +build linux

package overlay

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"

	"github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/containers/storage/pkg/directory"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/mount"
	"github.com/containers/storage/pkg/parsers"
	"github.com/containers/storage/pkg/parsers/kernel"

	"github.com/opencontainers/selinux/go-selinux/label"
)

var (
	// untar defines the untar method
	untar = chrootarchive.UntarUncompressed
)

// This backend uses the overlay union filesystem for containers
// with diff directories for each layer.

// This version of the overlay driver requires at least kernel
// 4.0.0 in order to support mounting multiple diff directories.

// Each container/image has at least a "diff" directory and "link" file.
// If there is also a "lower" file when there are diff layers
// below  as well as "merged" and "work" directories. The "diff" directory
// has the upper layer of the overlay and is used to capture any
// changes to the layer. The "lower" file contains all the lower layer
// mounts separated by ":" and ordered from uppermost to lowermost
// layers. The overlay itself is mounted in the "merged" directory,
// and the "work" dir is needed for overlay to work.

// The "link" file for each layer contains a unique string for the layer.
// Under the "l" directory at the root there will be a symbolic link
// with that unique string pointing the "diff" directory for the layer.
// The symbolic links are used to reference lower layers in the "lower"
// file and on mount. The links are used to shorten the total length
// of a layer reference without requiring changes to the layer identifier
// or root directory. Mounts are always done relative to root and
// referencing the symbolic links in order to ensure the number of
// lower directories can fit in a single page for making the mount
// syscall. A hard upper limit of 128 lower layers is enforced to ensure
// that mounts do not fail due to length.

const (
	linkDir   = "l"
	lowerFile = "lower"
	maxDepth  = 128

	// idLength represents the number of random characters
	// which can be used to create the unique link identifer
	// for every layer. If this value is too long then the
	// page size limit for the mount command may be exceeded.
	// The idLength should be selected such that following equation
	// is true (512 is a buffer for label metadata).
	// ((idLength + len(linkDir) + 1) * maxDepth) <= (pageSize - 512)
	idLength = 26
)

// Driver contains information about the home directory and the list of active mounts that are created using this driver.
type Driver struct {
	name    string
	home    string
	uidMaps []idtools.IDMap
	gidMaps []idtools.IDMap
	ctr     *graphdriver.RefCounter
	opts    *overlayOptions
}

var backingFs = "<unknown>"

func init() {
	graphdriver.Register("overlay", InitAsOverlay)
	graphdriver.Register("overlay2", InitAsOverlay2)
}

// InitWithName returns the a naive diff driver for the overlay filesystem,
// which returns the passed-in name when asked which driver it is.
func InitWithName(name, home string, options []string, uidMaps, gidMaps []idtools.IDMap) (graphdriver.Driver, error) {
	opts, err := parseOptions(options)
	if err != nil {
		return nil, err
	}

	if err := supportsOverlay(); err != nil {
		return nil, graphdriver.ErrNotSupported
	}

	// require kernel 4.0.0 to ensure multiple lower dirs are supported
	v, err := kernel.GetKernelVersion()
	if err != nil {
		return nil, err
	}
	if kernel.CompareKernelVersion(*v, kernel.VersionInfo{Kernel: 4, Major: 0, Minor: 0}) < 0 {
		if !opts.overrideKernelCheck {
			return nil, graphdriver.ErrNotSupported
		}
		logrus.Warnf("Using pre-4.0.0 kernel for overlay, mount failures may require kernel update")
	}

	fsMagic, err := graphdriver.GetFSMagic(home)
	if err != nil {
		return nil, err
	}
	if fsName, ok := graphdriver.FsNames[fsMagic]; ok {
		backingFs = fsName
	}

	// check if they are running over btrfs, aufs, zfs, overlay, or ecryptfs
	switch fsMagic {
	case graphdriver.FsMagicBtrfs, graphdriver.FsMagicAufs, graphdriver.FsMagicZfs, graphdriver.FsMagicOverlay, graphdriver.FsMagicEcryptfs:
		logrus.Errorf("'overlay' is not supported over %s", backingFs)
		return nil, graphdriver.ErrIncompatibleFS
	}

	rootUID, rootGID, err := idtools.GetRootUIDGID(uidMaps, gidMaps)
	if err != nil {
		return nil, err
	}
	// Create the driver home dir
	if err := idtools.MkdirAllAs(path.Join(home, linkDir), 0700, rootUID, rootGID); err != nil && !os.IsExist(err) {
		return nil, err
	}

	if err := mount.MakePrivate(home); err != nil {
		return nil, err
	}

	d := &Driver{
		name:    name,
		home:    home,
		uidMaps: uidMaps,
		gidMaps: gidMaps,
		ctr:     graphdriver.NewRefCounter(graphdriver.NewFsChecker(graphdriver.FsMagicOverlay)),
		opts:    opts,
	}

	return d, nil
}

// InitAsOverlay returns the a naive diff driver for overlay filesystem.
// If overlay filesystem is not supported on the host, graphdriver.ErrNotSupported is returned as error.
// If a overlay filesystem is not supported over a existing filesystem then error graphdriver.ErrIncompatibleFS is returned.
func InitAsOverlay(home string, options []string, uidMaps, gidMaps []idtools.IDMap) (graphdriver.Driver, error) {
	return InitWithName("overlay", home, options, uidMaps, gidMaps)
}

// InitAsOverlay2 returns the a naive diff driver for overlay filesystem.
// If overlay filesystem is not supported on the host, graphdriver.ErrNotSupported is returned as error.
// If a overlay filesystem is not supported over a existing filesystem then error graphdriver.ErrIncompatibleFS is returned.
func InitAsOverlay2(home string, options []string, uidMaps, gidMaps []idtools.IDMap) (graphdriver.Driver, error) {
	return InitWithName("overlay2", home, options, uidMaps, gidMaps)
}

type overlayOptions struct {
	overrideKernelCheck bool
	imageStores         []string
}

func parseOptions(options []string) (*overlayOptions, error) {
	o := &overlayOptions{}
	for _, option := range options {
		key, val, err := parsers.ParseKeyValueOpt(option)
		if err != nil {
			return nil, err
		}
		key = strings.ToLower(key)
		switch key {
		case "overlay.override_kernel_check", "overlay2.override_kernel_check":
			o.overrideKernelCheck, err = strconv.ParseBool(val)
			if err != nil {
				return nil, err
			}
		case "overlay.imagestore":
			// Additional read only image stores to use for lower paths
			for _, store := range strings.Split(val, ",") {
				store = filepath.Clean(store)
				if !filepath.IsAbs(store) {
					return nil, fmt.Errorf("overlay: image path %q is not absolute.  Can not be relative", store)
				}
				st, err := os.Stat(store)
				if err != nil {
					return nil, fmt.Errorf("overlay: Can't stat imageStore dir %s: %v", store, err)
				}
				if !st.IsDir() {
					return nil, fmt.Errorf("overlay: image path %q must be a directory", store)
				}
				o.imageStores = append(o.imageStores, store)
			}
		default:
			return nil, fmt.Errorf("overlay: Unknown option %s", key)
		}
	}
	return o, nil
}

func supportsOverlay() error {
	// We can try to modprobe overlay first before looking at
	// proc/filesystems for when overlay is supported
	exec.Command("modprobe", "overlay").Run()

	f, err := os.Open("/proc/filesystems")
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if s.Text() == "nodev\toverlay" {
			return nil
		}
	}
	logrus.Error("'overlay' not found as a supported filesystem on this host. Please ensure kernel is new enough and has overlay support loaded.")
	return graphdriver.ErrNotSupported
}

func (d *Driver) String() string {
	return d.name
}

// Status returns current driver information in a two dimensional string array.
// Output contains "Backing Filesystem" used in this implementation.
func (d *Driver) Status() [][2]string {
	return [][2]string{
		{"Backing Filesystem", backingFs},
	}
}

// Metadata returns meta data about the overlay driver such as
// LowerDir, UpperDir, WorkDir and MergeDir used to store data.
func (d *Driver) Metadata(id string) (map[string]string, error) {
	dir := d.dir(id)
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}

	metadata := map[string]string{
		"WorkDir":   path.Join(dir, "work"),
		"MergedDir": path.Join(dir, "merged"),
		"UpperDir":  path.Join(dir, "diff"),
	}

	lowerDirs, err := d.getLowerDirs(id)
	if err != nil {
		return nil, err
	}
	if len(lowerDirs) > 0 {
		metadata["LowerDir"] = strings.Join(lowerDirs, ":")
	}

	return metadata, nil
}

// Cleanup any state created by overlay which should be cleaned when daemon
// is being shutdown. For now, we just have to unmount the bind mounted
// we had created.
func (d *Driver) Cleanup() error {
	return mount.Unmount(d.home)
}

// CreateReadWrite creates a layer that is writable for use as a container
// file system.
func (d *Driver) CreateReadWrite(id, parent, mountLabel string, storageOpt map[string]string) error {
	return d.Create(id, parent, mountLabel, storageOpt)
}

// Create is used to create the upper, lower, and merge directories required for overlay fs for a given id.
// The parent filesystem is used to configure these directories for the overlay.
func (d *Driver) Create(id, parent, mountLabel string, storageOpt map[string]string) (retErr error) {

	if len(storageOpt) != 0 {
		return fmt.Errorf("--storage-opt is not supported for overlay")
	}

	dir := d.dir(id)

	rootUID, rootGID, err := idtools.GetRootUIDGID(d.uidMaps, d.gidMaps)
	if err != nil {
		return err
	}
	if err := idtools.MkdirAllAs(path.Dir(dir), 0700, rootUID, rootGID); err != nil {
		return err
	}
	if err := idtools.MkdirAs(dir, 0700, rootUID, rootGID); err != nil {
		return err
	}

	defer func() {
		// Clean up on failure
		if retErr != nil {
			os.RemoveAll(dir)
		}
	}()

	if err := idtools.MkdirAs(path.Join(dir, "diff"), 0755, rootUID, rootGID); err != nil {
		return err
	}

	lid := generateID(idLength)
	if err := os.Symlink(path.Join("..", id, "diff"), path.Join(d.home, linkDir, lid)); err != nil {
		return err
	}

	// Write link id to link file
	if err := ioutil.WriteFile(path.Join(dir, "link"), []byte(lid), 0644); err != nil {
		return err
	}

	// if no parent directory, done
	if parent == "" {
		return nil
	}

	if err := idtools.MkdirAs(path.Join(dir, "work"), 0700, rootUID, rootGID); err != nil {
		return err
	}
	if err := idtools.MkdirAs(path.Join(dir, "merged"), 0700, rootUID, rootGID); err != nil {
		return err
	}

	lower, err := d.getLower(parent)
	if err != nil {
		return err
	}
	if lower != "" {
		if err := ioutil.WriteFile(path.Join(dir, lowerFile), []byte(lower), 0666); err != nil {
			return err
		}
	}

	return nil
}

func (d *Driver) getLower(parent string) (string, error) {
	parentDir := d.dir(parent)

	// Ensure parent exists
	if _, err := os.Lstat(parentDir); err != nil {
		return "", err
	}

	// Read Parent link fileA
	parentLink, err := ioutil.ReadFile(path.Join(parentDir, "link"))
	if err != nil {
		return "", err
	}
	lowers := []string{path.Join(linkDir, string(parentLink))}

	parentLower, err := ioutil.ReadFile(path.Join(parentDir, lowerFile))
	if err == nil {
		parentLowers := strings.Split(string(parentLower), ":")
		lowers = append(lowers, parentLowers...)
	}
	if len(lowers) > maxDepth {
		return "", errors.New("max depth exceeded")
	}
	return strings.Join(lowers, ":"), nil
}

func (d *Driver) dir(val string) string {
	newpath := path.Join(d.home, val)
	if _, err := os.Stat(newpath); err != nil {
		for _, p := range d.AdditionalImageStores() {
			l := path.Join(p, d.name, val)
			_, err = os.Stat(l)
			if err == nil {
				return l
			}
		}
	}
	return newpath
}

func (d *Driver) getLowerDirs(id string) ([]string, error) {
	var lowersArray []string
	lowers, err := ioutil.ReadFile(path.Join(d.dir(id), lowerFile))
	if err == nil {
		for _, s := range strings.Split(string(lowers), ":") {
			lower := d.dir(s)
			lp, err := os.Readlink(lower)
			if err != nil {
				return nil, err
			}
			lowersArray = append(lowersArray, path.Clean(d.dir(path.Join("link", lp))))
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return lowersArray, nil
}

// Remove cleans the directories that are created for this id.
func (d *Driver) Remove(id string) error {
	dir := d.dir(id)
	lid, err := ioutil.ReadFile(path.Join(dir, "link"))
	if err == nil {
		if err := os.RemoveAll(path.Join(d.home, linkDir, string(lid))); err != nil {
			logrus.Debugf("Failed to remove link: %v", err)
		}
	}

	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Get creates and mounts the required file system for the given id and returns the mount path.
func (d *Driver) Get(id string, mountLabel string) (s string, err error) {
	dir := d.dir(id)
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}

	diffDir := path.Join(dir, "diff")
	lowers, err := ioutil.ReadFile(path.Join(dir, lowerFile))
	if err != nil {
		// If no lower, just return diff directory
		if os.IsNotExist(err) {
			return diffDir, nil
		}
		return "", err
	}

	newlowers := ""
	for _, l := range strings.Split(string(lowers), ":") {
		lower := ""
		newpath := path.Join(d.home, l)
		if _, err := os.Stat(newpath); err != nil {
			for _, p := range d.AdditionalImageStores() {
				lower = path.Join(p, d.name, l)
				if _, err2 := os.Stat(lower); err2 == nil {
					break
				}
				lower = ""
			}
			if lower == "" {
				return "", fmt.Errorf("Can't stat lower layer %q: %v", newpath, err)
			}
		} else {
			lower = l
		}
		if newlowers == "" {
			newlowers = lower
		} else {
			newlowers = newlowers + ":" + lower
		}
	}

	mergedDir := path.Join(dir, "merged")
	if count := d.ctr.Increment(mergedDir); count > 1 {
		return mergedDir, nil
	}
	defer func() {
		if err != nil {
			if c := d.ctr.Decrement(mergedDir); c <= 0 {
				syscall.Unmount(mergedDir, 0)
			}
		}
	}()

	workDir := path.Join(dir, "work")
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", newlowers, path.Join(id, "diff"), path.Join(id, "work"))
	mountLabel = label.FormatMountLabel(opts, mountLabel)
	if len(mountLabel) > syscall.Getpagesize() {
		return "", fmt.Errorf("cannot mount layer, mount label too large %d", len(mountLabel))
	}

	if err := mountFrom(d.home, "overlay", path.Join(id, "merged"), "overlay", mountLabel); err != nil {
		return "", fmt.Errorf("error creating overlay mount to %s: %v", mergedDir, err)
	}

	// chown "workdir/work" to the remapped root UID/GID. Overlay fs inside a
	// user namespace requires this to move a directory from lower to upper.
	rootUID, rootGID, err := idtools.GetRootUIDGID(d.uidMaps, d.gidMaps)
	if err != nil {
		return "", err
	}

	if err := os.Chown(path.Join(workDir, "work"), rootUID, rootGID); err != nil {
		return "", err
	}

	return mergedDir, nil
}

// Put unmounts the mount path created for the give id.
func (d *Driver) Put(id string) error {
	mountpoint := path.Join(d.dir(id), "merged")
	if count := d.ctr.Decrement(mountpoint); count > 0 {
		return nil
	}
	err := syscall.Unmount(mountpoint, 0)
	if err != nil {
		if _, err := ioutil.ReadFile(path.Join(d.dir(id), lowerFile)); err != nil {
			// We didn't have a "lower" directory, so we weren't mounting a "merged" directory anyway
			return nil
		}
		logrus.Debugf("Failed to unmount %s overlay: %v", id, err)
	}
	return err
}

// Exists checks to see if the id is already mounted.
func (d *Driver) Exists(id string) bool {
	_, err := os.Stat(d.dir(id))
	return err == nil
}

// ApplyDiff applies the new layer into a root
func (d *Driver) ApplyDiff(id string, parent string, diff archive.Reader) (size int64, err error) {
	applyDir := d.getDiffPath(id)

	logrus.Debugf("Applying tar in %s", applyDir)
	// Overlay doesn't need the parent id to apply the diff
	if err := untar(diff, applyDir, &archive.TarOptions{
		UIDMaps:        d.uidMaps,
		GIDMaps:        d.gidMaps,
		WhiteoutFormat: archive.OverlayWhiteoutFormat,
	}); err != nil {
		return 0, err
	}

	return d.DiffSize(id, parent)
}

func (d *Driver) getDiffPath(id string) string {
	dir := d.dir(id)

	return path.Join(dir, "diff")
}

// DiffSize calculates the changes between the specified id
// and its parent and returns the size in bytes of the changes
// relative to its base filesystem directory.
func (d *Driver) DiffSize(id, parent string) (size int64, err error) {
	return directory.Size(d.getDiffPath(id))
}

// Diff produces an archive of the changes between the specified
// layer and its parent layer which may be "".
func (d *Driver) Diff(id, parent string) (archive.Archive, error) {
	diffPath := d.getDiffPath(id)
	logrus.Debugf("Tar with options on %s", diffPath)
	return archive.TarWithOptions(diffPath, &archive.TarOptions{
		Compression:    archive.Uncompressed,
		UIDMaps:        d.uidMaps,
		GIDMaps:        d.gidMaps,
		WhiteoutFormat: archive.OverlayWhiteoutFormat,
	})
}

// Changes produces a list of changes between the specified layer
// and its parent layer. If parent is "", then all changes will be ADD changes.
func (d *Driver) Changes(id, parent string) ([]archive.Change, error) {
	// Overlay doesn't have snapshots, so we need to get changes from all parent
	// layers.
	diffPath := d.getDiffPath(id)
	layers, err := d.getLowerDirs(id)
	if err != nil {
		return nil, err
	}

	return archive.OverlayChanges(layers, diffPath)
}

// AdditionalImageStores returns additional image stores supported by the driver
func (d *Driver) AdditionalImageStores() []string {
	return d.opts.imageStores
}
