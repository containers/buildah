package buildah

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/buildah/util"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/pools"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/system"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	selinux "github.com/opencontainers/selinux/go-selinux"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// InitReexec is a wrapper for reexec.Init().  It should be called at
// the start of main(), and if it returns true, main() should return
// immediately.
func InitReexec() bool {
	return reexec.Init()
}

func copyStringStringMap(m map[string]string) map[string]string {
	n := map[string]string{}
	for k, v := range m {
		n[k] = v
	}
	return n
}

func copyStringSlice(s []string) []string {
	t := make([]string, len(s))
	copy(t, s)
	return t
}

func copyHistory(history []v1.History) []v1.History {
	if len(history) == 0 {
		return nil
	}
	h := make([]v1.History, 0, len(history))
	for _, entry := range history {
		created := entry.Created
		if created != nil {
			timestamp := *created
			created = &timestamp
		}
		h = append(h, v1.History{
			Created:    created,
			CreatedBy:  entry.CreatedBy,
			Author:     entry.Author,
			Comment:    entry.Comment,
			EmptyLayer: entry.EmptyLayer,
		})
	}
	return h
}

func convertStorageIDMaps(UIDMap, GIDMap []idtools.IDMap) ([]rspec.LinuxIDMapping, []rspec.LinuxIDMapping) {
	uidmap := make([]rspec.LinuxIDMapping, 0, len(UIDMap))
	gidmap := make([]rspec.LinuxIDMapping, 0, len(GIDMap))
	for _, m := range UIDMap {
		uidmap = append(uidmap, rspec.LinuxIDMapping{
			HostID:      uint32(m.HostID),
			ContainerID: uint32(m.ContainerID),
			Size:        uint32(m.Size),
		})
	}
	for _, m := range GIDMap {
		gidmap = append(gidmap, rspec.LinuxIDMapping{
			HostID:      uint32(m.HostID),
			ContainerID: uint32(m.ContainerID),
			Size:        uint32(m.Size),
		})
	}
	return uidmap, gidmap
}

func convertRuntimeIDMaps(UIDMap, GIDMap []rspec.LinuxIDMapping) ([]idtools.IDMap, []idtools.IDMap) {
	uidmap := make([]idtools.IDMap, 0, len(UIDMap))
	gidmap := make([]idtools.IDMap, 0, len(GIDMap))
	for _, m := range UIDMap {
		uidmap = append(uidmap, idtools.IDMap{
			HostID:      int(m.HostID),
			ContainerID: int(m.ContainerID),
			Size:        int(m.Size),
		})
	}
	for _, m := range GIDMap {
		gidmap = append(gidmap, idtools.IDMap{
			HostID:      int(m.HostID),
			ContainerID: int(m.ContainerID),
			Size:        int(m.Size),
		})
	}
	return uidmap, gidmap
}

// copyFileWithTar returns a function which copies a single file from outside
// of any container, or another container, into our working container, mapping
// read permissions using the passed-in ID maps, writing using the container's
// ID mappings, possibly overridden using the passed-in chownOpts
func (b *Builder) copyFileWithTar(tarIDMappingOptions *IDMappingOptions, chownOpts *idtools.IDPair, hasher io.Writer, dryRun bool) func(src, dest string) error {
	if tarIDMappingOptions == nil {
		tarIDMappingOptions = &IDMappingOptions{
			HostUIDMapping: true,
			HostGIDMapping: true,
		}
	}

	var hardlinkChecker util.HardlinkChecker
	return func(src, dest string) error {
		var f *os.File

		logrus.Debugf("copyFileWithTar(%s, %s)", src, dest)
		fi, err := os.Lstat(src)
		if err != nil {
			return errors.Wrapf(err, "error reading attributes of %q", src)
		}

		sysfi, err := system.Lstat(src)
		if err != nil {
			return errors.Wrapf(err, "error reading attributes of %q", src)
		}

		hostUID := sysfi.UID()
		hostGID := sysfi.GID()
		containerUID, containerGID, err := util.GetContainerIDs(tarIDMappingOptions.UIDMap, tarIDMappingOptions.GIDMap, hostUID, hostGID)
		if err != nil {
			return errors.Wrapf(err, "error mapping owner IDs of %q: %d/%d", src, hostUID, hostGID)
		}

		hdr, err := tar.FileInfoHeader(fi, filepath.Base(src))
		if err != nil {
			return errors.Wrapf(err, "error generating tar header for: %q", src)
		}
		chrootedDest, err := filepath.Rel(b.MountPoint, dest)
		if err != nil {
			return errors.Wrapf(err, "error generating relative-to-chroot target name for %q", dest)
		}
		hdr.Name = chrootedDest
		hdr.Uid = int(containerUID)
		hdr.Gid = int(containerGID)

		if fi.Mode().IsRegular() && hdr.Typeflag == tar.TypeReg {
			if linkname := hardlinkChecker.Check(fi); linkname != "" {
				hdr.Typeflag = tar.TypeLink
				hdr.Linkname = linkname
			} else {
				hardlinkChecker.Add(fi, chrootedDest)
				f, err = os.Open(src)
				if err != nil {
					return errors.Wrapf(err, "error opening %q to copy its contents", src)
				}
			}
		}

		if fi.Mode()&os.ModeSymlink == os.ModeSymlink && hdr.Typeflag == tar.TypeSymlink {
			hdr.Typeflag = tar.TypeSymlink
			linkName, err := os.Readlink(src)
			if err != nil {
				return errors.Wrapf(err, "error reading destination from symlink %q", src)
			}
			hdr.Linkname = linkName
		}

		pipeReader, pipeWriter := io.Pipe()
		writer := tar.NewWriter(pipeWriter)
		var copyErr error
		go func(srcFile *os.File) {
			err := writer.WriteHeader(hdr)
			if err != nil {
				logrus.Debugf("error writing header for %s: %v", srcFile.Name(), err)
				copyErr = err
			}
			if srcFile != nil {
				n, err := pools.Copy(writer, srcFile)
				if n != hdr.Size {
					logrus.Debugf("expected to write %d bytes for %s, wrote %d instead", hdr.Size, srcFile.Name(), n)
				}
				if err != nil {
					logrus.Debugf("error copying contents of %s: %v", fi.Name(), err)
					copyErr = err
				}
				if err = srcFile.Close(); err != nil {
					logrus.Debugf("error closing %s: %v", fi.Name(), err)
				}
			}
			if err = writer.Close(); err != nil {
				logrus.Debugf("error closing write pipe for %s: %v", hdr.Name, err)
			}
			pipeWriter.Close()
			pipeWriter = nil
		}(f)

		untar := b.untar(chownOpts, hasher, dryRun)
		err = untar(pipeReader, b.MountPoint)
		if err == nil {
			err = copyErr
		}
		if pipeWriter != nil {
			pipeWriter.Close()
		}
		return err
	}
}

// copyWithTar returns a function which copies a directory tree from outside of
// our container or from another container, into our working container, mapping
// permissions at read-time using the container's ID maps, with ownership at
// write-time possibly overridden using the passed-in chownOpts
func (b *Builder) copyWithTar(tarIDMappingOptions *IDMappingOptions, chownOpts *idtools.IDPair, hasher io.Writer, dryRun bool) func(src, dest string) error {
	tar := b.tarPath(tarIDMappingOptions)
	return func(src, dest string) error {
		thisHasher := hasher
		if thisHasher != nil && b.ContentDigester.Hash() != nil {
			thisHasher = io.MultiWriter(thisHasher, b.ContentDigester.Hash())
		}
		if thisHasher == nil {
			thisHasher = b.ContentDigester.Hash()
		}
		untar := b.untar(chownOpts, thisHasher, dryRun)
		rc, err := tar(src)
		if err != nil {
			return errors.Wrapf(err, "error archiving %q for copy", src)
		}
		return untar(rc, dest)
	}
}

// untarPath returns a function which extracts an archive in a specified
// location into our working container, mapping permissions using the
// container's ID maps, possibly overridden using the passed-in chownOpts
func (b *Builder) untarPath(chownOpts *idtools.IDPair, hasher io.Writer, dryRun bool) func(src, dest string) error {
	convertedUIDMap, convertedGIDMap := convertRuntimeIDMaps(b.IDMappingOptions.UIDMap, b.IDMappingOptions.GIDMap)
	if dryRun {
		return func(src, dest string) error {
			thisHasher := hasher
			if thisHasher != nil && b.ContentDigester.Hash() != nil {
				thisHasher = io.MultiWriter(thisHasher, b.ContentDigester.Hash())
			}
			if thisHasher == nil {
				thisHasher = b.ContentDigester.Hash()
			}
			f, err := os.Open(src)
			if err != nil {
				return errors.Wrapf(err, "error opening %q", src)
			}
			defer f.Close()
			_, err = io.Copy(thisHasher, f)
			return err
		}
	}
	return func(src, dest string) error {
		thisHasher := hasher
		if thisHasher != nil && b.ContentDigester.Hash() != nil {
			thisHasher = io.MultiWriter(thisHasher, b.ContentDigester.Hash())
		}
		if thisHasher == nil {
			thisHasher = b.ContentDigester.Hash()
		}
		untarPathAndChown := chrootarchive.UntarPathAndChown(chownOpts, thisHasher, convertedUIDMap, convertedGIDMap)
		return untarPathAndChown(src, dest)
	}
}

// tarPath returns a function which creates an archive of a specified location,
// which is often somewhere in the container's filesystem, mapping permissions
// using the container's ID maps, or the passed-in maps if specified
func (b *Builder) tarPath(idMappingOptions *IDMappingOptions) func(path string) (io.ReadCloser, error) {
	var uidmap, gidmap []idtools.IDMap
	if idMappingOptions == nil {
		idMappingOptions = &IDMappingOptions{
			HostUIDMapping: true,
			HostGIDMapping: true,
		}
	}
	convertedUIDMap, convertedGIDMap := convertRuntimeIDMaps(idMappingOptions.UIDMap, idMappingOptions.GIDMap)
	tarMappings := idtools.NewIDMappingsFromMaps(convertedUIDMap, convertedGIDMap)
	uidmap = tarMappings.UIDs()
	gidmap = tarMappings.GIDs()
	options := &archive.TarOptions{
		Compression: archive.Uncompressed,
		UIDMaps:     uidmap,
		GIDMaps:     gidmap,
	}
	return func(path string) (io.ReadCloser, error) {
		return archive.TarWithOptions(path, options)
	}
}

// untar returns a function which extracts an archive stream to a specified
// location in the container's filesystem, mapping permissions using the
// container's ID maps, possibly overridden using the passed-in chownOpts
func (b *Builder) untar(chownOpts *idtools.IDPair, hasher io.Writer, dryRun bool) func(tarArchive io.ReadCloser, dest string) error {
	convertedUIDMap, convertedGIDMap := convertRuntimeIDMaps(b.IDMappingOptions.UIDMap, b.IDMappingOptions.GIDMap)
	untarMappings := idtools.NewIDMappingsFromMaps(convertedUIDMap, convertedGIDMap)
	options := &archive.TarOptions{
		UIDMaps:   untarMappings.UIDs(),
		GIDMaps:   untarMappings.GIDs(),
		ChownOpts: chownOpts,
	}
	untar := chrootarchive.Untar
	if dryRun {
		untar = func(tarArchive io.Reader, dest string, options *archive.TarOptions) error {
			if _, err := io.Copy(ioutil.Discard, tarArchive); err != nil {
				return errors.Wrapf(err, "error digesting tar stream")
			}
			return nil
		}
	}
	originalUntar := untar
	untarWithHasher := func(tarArchive io.Reader, dest string, options *archive.TarOptions, untarHasher io.Writer) error {
		reader := tarArchive
		if untarHasher != nil {
			reader = io.TeeReader(tarArchive, untarHasher)
		}
		return originalUntar(reader, dest, options)
	}
	return func(tarArchive io.ReadCloser, dest string) error {
		thisHasher := hasher
		if thisHasher != nil && b.ContentDigester.Hash() != nil {
			thisHasher = io.MultiWriter(thisHasher, b.ContentDigester.Hash())
		}
		if thisHasher == nil {
			thisHasher = b.ContentDigester.Hash()
		}
		err := untarWithHasher(tarArchive, dest, options, thisHasher)
		if err2 := tarArchive.Close(); err2 != nil {
			if err == nil {
				err = err2
			}
		}
		return err
	}
}

// isRegistryBlocked checks if the named registry is marked as blocked
func isRegistryBlocked(registry string, sc *types.SystemContext) (bool, error) {
	reginfo, err := sysregistriesv2.FindRegistry(sc, registry)
	if err != nil {
		return false, errors.Wrapf(err, "unable to parse the registries configuration (%s)", sysregistriesv2.ConfigPath(sc))
	}
	if reginfo != nil {
		if reginfo.Blocked {
			logrus.Debugf("registry %q is marked as blocked in registries configuration %q", registry, sysregistriesv2.ConfigPath(sc))
		} else {
			logrus.Debugf("registry %q is not marked as blocked in registries configuration %q", registry, sysregistriesv2.ConfigPath(sc))
		}
		return reginfo.Blocked, nil
	}
	logrus.Debugf("registry %q is not listed in registries configuration %q, assuming it's not blocked", registry, sysregistriesv2.ConfigPath(sc))
	return false, nil
}

// isReferenceSomething checks if the registry part of a reference is insecure or blocked
func isReferenceSomething(ref types.ImageReference, sc *types.SystemContext, what func(string, *types.SystemContext) (bool, error)) (bool, error) {
	if ref != nil && ref.DockerReference() != nil {
		if named, ok := ref.DockerReference().(reference.Named); ok {
			if domain := reference.Domain(named); domain != "" {
				return what(domain, sc)
			}
		}
	}
	return false, nil
}

// isReferenceBlocked checks if the registry part of a reference is blocked
func isReferenceBlocked(ref types.ImageReference, sc *types.SystemContext) (bool, error) {
	if ref != nil && ref.Transport() != nil {
		switch ref.Transport().Name() {
		case "docker":
			return isReferenceSomething(ref, sc, isRegistryBlocked)
		}
	}
	return false, nil
}

// ReserveSELinuxLabels reads containers storage and reserves SELinux containers
// fall all existing buildah containers
func ReserveSELinuxLabels(store storage.Store, id string) error {
	if selinux.GetEnabled() {
		containers, err := store.Containers()
		if err != nil {
			return errors.Wrapf(err, "error getting list of containers")
		}

		for _, c := range containers {
			if id == c.ID {
				continue
			} else {
				b, err := OpenBuilder(store, c.ID)
				if err != nil {
					if os.IsNotExist(errors.Cause(err)) {
						// Ignore not exist errors since containers probably created by other tool
						// TODO, we need to read other containers json data to reserve their SELinux labels
						continue
					}
					return err
				}
				// Prevent different containers from using same MCS label
				if err := label.ReserveLabel(b.ProcessLabel); err != nil {
					return errors.Wrapf(err, "error reserving SELinux label %q", b.ProcessLabel)
				}
			}
		}
	}
	return nil
}
