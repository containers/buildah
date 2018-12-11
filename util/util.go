package util

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/containers/image/directory"
	dockerarchive "github.com/containers/image/docker/archive"
	"github.com/containers/image/docker/reference"
	ociarchive "github.com/containers/image/oci/archive"
	"github.com/containers/image/pkg/sysregistriesv2"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/tarball"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/idtools"
	"github.com/docker/distribution/registry/api/errcode"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	minimumTruncatedIDLength = 3
	// DefaultTransport is a prefix that we apply to an image name if we
	// can't find one in the local Store, in order to generate a source
	// reference for the image that we can then copy to the local Store.
	DefaultTransport = "docker://"
)

var (
	// RegistryDefaultPathPrefix contains a per-registry listing of default prefixes
	// to prepend to image names that only contain a single path component.
	RegistryDefaultPathPrefix = map[string]string{
		"index.docker.io": "library",
		"docker.io":       "library",
	}
	// Transports contains the possible transports used for images
	Transports = map[string]string{
		dockerarchive.Transport.Name(): "",
		ociarchive.Transport.Name():    "",
		directory.Transport.Name():     "",
		tarball.Transport.Name():       "",
	}
	// DockerArchive is the transport we prepend to an image name
	// when saving to docker-archive
	DockerArchive = dockerarchive.Transport.Name()
	// OCIArchive is the transport we prepend to an image name
	// when saving to oci-archive
	OCIArchive = ociarchive.Transport.Name()
	// DirTransport is the transport for pushing and pulling
	// images to and from a directory
	DirTransport = directory.Transport.Name()
	// TarballTransport is the transport for importing a tar archive
	// and creating a filesystem image
	TarballTransport = tarball.Transport.Name()
)

// ResolveName checks if name is a valid image name, and if that name doesn't
// include a domain portion, returns a list of the names which it might
// correspond to in the set of configured registries,
// and a boolean which is true iff 1) the list of search registries was used, and 2) it was empty.
// NOTE: The "list of search registries is empty" check does not count blocked registries,
// and neither the implied "localhost" nor a possible firstRegistry are counted
func ResolveName(name string, firstRegistry string, sc *types.SystemContext, store storage.Store) ([]string, bool, error) {
	if name == "" {
		return nil, false, nil
	}

	// Maybe it's a truncated image ID.  Don't prepend a registry name, then.
	if len(name) >= minimumTruncatedIDLength {
		if img, err := store.Image(name); err == nil && img != nil && strings.HasPrefix(img.ID, name) {
			// It's a truncated version of the ID of an image that's present in local storage;
			// we need only expand the ID.
			return []string{img.ID}, false, nil
		}
	}

	// If the image includes a transport's name as a prefix, use it as-is.
	split := strings.SplitN(name, ":", 2)
	if len(split) == 2 {
		if _, ok := Transports[split[0]]; ok {
			return []string{split[1]}, false, nil
		}
	}

	name = strings.TrimPrefix(name, DefaultTransport)
	// If the image name already included a domain component, we're done.
	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, false, errors.Wrapf(err, "error parsing image name %q", name)
	}
	if named.String() == name {
		// Parsing produced the same result, so there was a domain name in there to begin with.
		return []string{name}, false, nil
	}
	if reference.Domain(named) != "" && RegistryDefaultPathPrefix[reference.Domain(named)] != "" {
		// If this domain can cause us to insert something in the middle, check if that happened.
		repoPath := reference.Path(named)
		domain := reference.Domain(named)
		tag := ""
		if tagged, ok := named.(reference.Tagged); ok {
			tag = ":" + tagged.Tag()
		}
		digest := ""
		if digested, ok := named.(reference.Digested); ok {
			digest = "@" + digested.Digest().String()
		}
		defaultPrefix := RegistryDefaultPathPrefix[reference.Domain(named)] + "/"
		if strings.HasPrefix(repoPath, defaultPrefix) && path.Join(domain, repoPath[len(defaultPrefix):])+tag+digest == name {
			// Yup, parsing just inserted a bit in the middle, so there was a domain name there to begin with.
			return []string{name}, false, nil
		}
	}

	// Figure out the list of registries.
	var registries []string
	searchRegistries, err := sysregistriesv2.FindUnqualifiedSearchRegistries(sc)
	if err != nil {
		logrus.Debugf("unable to read configured registries to complete %q: %v", name, err)
	}
	for _, registry := range searchRegistries {
		if !registry.Blocked {
			registries = append(registries, registry.URL)
		}
	}
	searchRegistriesAreEmpty := len(registries) == 0

	// Create all of the combinations.  Some registries need an additional component added, so
	// use our lookaside map to keep track of them.  If there are no configured registries, we'll
	// return a name using "localhost" as the registry name.
	candidates := []string{}
	initRegistries := []string{"localhost"}
	if firstRegistry != "" && firstRegistry != "localhost" {
		initRegistries = append([]string{firstRegistry}, initRegistries...)
	}
	for _, registry := range append(initRegistries, registries...) {
		if registry == "" {
			continue
		}
		middle := ""
		if prefix, ok := RegistryDefaultPathPrefix[registry]; ok && strings.IndexRune(name, '/') == -1 {
			middle = prefix
		}
		candidate := path.Join(registry, middle, name)
		candidates = append(candidates, candidate)
	}
	return candidates, searchRegistriesAreEmpty, nil
}

// ExpandNames takes unqualified names, parses them as image names, and returns
// the fully expanded result, including a tag.  Names which don't include a registry
// name will be marked for the most-preferred registry (i.e., the first one in our
// configuration).
func ExpandNames(names []string, firstRegistry string, systemContext *types.SystemContext, store storage.Store) ([]string, error) {
	expanded := make([]string, 0, len(names))
	for _, n := range names {
		var name reference.Named
		nameList, _, err := ResolveName(n, firstRegistry, systemContext, store)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing name %q", n)
		}
		if len(nameList) == 0 {
			named, err := reference.ParseNormalizedNamed(n)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing name %q", n)
			}
			name = named
		} else {
			named, err := reference.ParseNormalizedNamed(nameList[0])
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing name %q", nameList[0])
			}
			name = named
		}
		name = reference.TagNameOnly(name)
		tag := ""
		digest := ""
		if tagged, ok := name.(reference.NamedTagged); ok {
			tag = ":" + tagged.Tag()
		}
		if digested, ok := name.(reference.Digested); ok {
			digest = "@" + digested.Digest().String()
		}
		expanded = append(expanded, name.Name()+tag+digest)
	}
	return expanded, nil
}

// FindImage locates the locally-stored image which corresponds to a given name.
func FindImage(store storage.Store, firstRegistry string, systemContext *types.SystemContext, image string) (types.ImageReference, *storage.Image, error) {
	var ref types.ImageReference
	var img *storage.Image
	var err error
	names, _, err := ResolveName(image, firstRegistry, systemContext, store)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error parsing name %q", image)
	}
	for _, name := range names {
		ref, err = is.Transport.ParseStoreReference(store, name)
		if err != nil {
			logrus.Debugf("error parsing reference to image %q: %v", name, err)
			continue
		}
		img, err = is.Transport.GetStoreImage(store, ref)
		if err != nil {
			img2, err2 := store.Image(name)
			if err2 != nil {
				logrus.Debugf("error locating image %q: %v", name, err2)
				continue
			}
			img = img2
		}
		break
	}
	if ref == nil || img == nil {
		return nil, nil, errors.Wrapf(err, "error locating image with name %q", image)
	}
	return ref, img, nil
}

// AddImageNames adds the specified names to the specified image.
func AddImageNames(store storage.Store, firstRegistry string, systemContext *types.SystemContext, image *storage.Image, addNames []string) error {
	names, err := ExpandNames(addNames, firstRegistry, systemContext, store)
	if err != nil {
		return err
	}
	err = store.SetNames(image.ID, append(image.Names, names...))
	if err != nil {
		return errors.Wrapf(err, "error adding names (%v) to image %q", names, image.ID)
	}
	return nil
}

// GetFailureCause checks the type of the error "err" and returns a new
// error message that reflects the reason of the failure.
// In case err type is not a familiar one the error "defaultError" is returned.
func GetFailureCause(err, defaultError error) error {
	switch nErr := errors.Cause(err).(type) {
	case errcode.Errors:
		return err
	case errcode.Error, *url.Error:
		return nErr
	default:
		return defaultError
	}
}

// WriteError writes `lastError` into `w` if not nil and return the next error `err`
func WriteError(w io.Writer, err error, lastError error) error {
	if lastError != nil {
		fmt.Fprintln(w, lastError)
	}
	return err
}

// Runtime is the default command to use to run the container.
func Runtime() string {
	runtime := os.Getenv("BUILDAH_RUNTIME")
	if runtime != "" {
		return runtime
	}
	return DefaultRuntime
}

// StringInSlice returns a boolean indicating if the exact value s is present
// in the slice slice.
func StringInSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// GetHostIDs uses ID mappings to compute the host-level IDs that will
// correspond to a UID/GID pair in the container.
func GetHostIDs(uidmap, gidmap []specs.LinuxIDMapping, uid, gid uint32) (uint32, uint32, error) {
	uidMapped := true
	for _, m := range uidmap {
		uidMapped = false
		if uid >= m.ContainerID && uid < m.ContainerID+m.Size {
			uid = (uid - m.ContainerID) + m.HostID
			uidMapped = true
			break
		}
	}
	if !uidMapped {
		return 0, 0, errors.Errorf("container uses ID mappings, but doesn't map UID %d", uid)
	}
	gidMapped := true
	for _, m := range gidmap {
		gidMapped = false
		if gid >= m.ContainerID && gid < m.ContainerID+m.Size {
			gid = (gid - m.ContainerID) + m.HostID
			gidMapped = true
			break
		}
	}
	if !gidMapped {
		return 0, 0, errors.Errorf("container uses ID mappings, but doesn't map GID %d", gid)
	}
	return uid, gid, nil
}

// GetHostRootIDs uses ID mappings in spec to compute the host-level IDs that will
// correspond to UID/GID 0/0 in the container.
func GetHostRootIDs(spec *specs.Spec) (uint32, uint32, error) {
	if spec.Linux == nil {
		return 0, 0, nil
	}
	return GetHostIDs(spec.Linux.UIDMappings, spec.Linux.GIDMappings, 0, 0)
}

// getHostIDMappings reads mappings from the named node under /proc.
func getHostIDMappings(path string) ([]specs.LinuxIDMapping, error) {
	var mappings []specs.LinuxIDMapping
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading ID mappings from %q", path)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, errors.Errorf("line %q from %q has %d fields, not 3", line, path, len(fields))
		}
		cid, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing container ID value %q from line %q in %q", fields[0], line, path)
		}
		hid, err := strconv.ParseUint(fields[1], 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing host ID value %q from line %q in %q", fields[1], line, path)
		}
		size, err := strconv.ParseUint(fields[2], 10, 32)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing size value %q from line %q in %q", fields[2], line, path)
		}
		mappings = append(mappings, specs.LinuxIDMapping{ContainerID: uint32(cid), HostID: uint32(hid), Size: uint32(size)})
	}
	return mappings, nil
}

// GetHostIDMappings reads mappings for the specified process (or the current
// process if pid is "self" or an empty string) from the kernel.
func GetHostIDMappings(pid string) ([]specs.LinuxIDMapping, []specs.LinuxIDMapping, error) {
	if pid == "" {
		pid = "self"
	}
	uidmap, err := getHostIDMappings(fmt.Sprintf("/proc/%s/uid_map", pid))
	if err != nil {
		return nil, nil, err
	}
	gidmap, err := getHostIDMappings(fmt.Sprintf("/proc/%s/gid_map", pid))
	if err != nil {
		return nil, nil, err
	}
	return uidmap, gidmap, nil
}

// GetSubIDMappings reads mappings from /etc/subuid and /etc/subgid.
func GetSubIDMappings(user, group string) ([]specs.LinuxIDMapping, []specs.LinuxIDMapping, error) {
	mappings, err := idtools.NewIDMappings(user, group)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "error reading subuid mappings for user %q and subgid mappings for group %q", user, group)
	}
	var uidmap, gidmap []specs.LinuxIDMapping
	for _, m := range mappings.UIDs() {
		uidmap = append(uidmap, specs.LinuxIDMapping{
			ContainerID: uint32(m.ContainerID),
			HostID:      uint32(m.HostID),
			Size:        uint32(m.Size),
		})
	}
	for _, m := range mappings.GIDs() {
		gidmap = append(gidmap, specs.LinuxIDMapping{
			ContainerID: uint32(m.ContainerID),
			HostID:      uint32(m.HostID),
			Size:        uint32(m.Size),
		})
	}
	return uidmap, gidmap, nil
}

// ParseIDMappings parses mapping triples.
func ParseIDMappings(uidmap, gidmap []string) ([]idtools.IDMap, []idtools.IDMap, error) {
	uid, err := idtools.ParseIDMap(uidmap, "userns-uid-map")
	if err != nil {
		return nil, nil, err
	}
	gid, err := idtools.ParseIDMap(gidmap, "userns-gid-map")
	if err != nil {
		return nil, nil, err
	}
	return uid, gid, nil
}

// GetPolicyContext sets up, initializes and returns a new context for the specified policy
func GetPolicyContext(ctx *types.SystemContext) (*signature.PolicyContext, error) {
	policy, err := signature.DefaultPolicy(ctx)
	if err != nil {
		return nil, err
	}

	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, err
	}
	return policyContext, nil
}
