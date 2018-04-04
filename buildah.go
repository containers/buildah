package buildah

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/docker"
)

const (
	// Package is the name of this package, used in help output and to
	// identify working containers.
	Package = "buildah"
	// Version for the Package.  Bump version in contrib/rpm/buildah.spec
	// too.
	Version = "0.16"
	// The value we use to identify what type of information, currently a
	// serialized Builder structure, we are using as per-container state.
	// This should only be changed when we make incompatible changes to
	// that data structure, as it's used to distinguish containers which
	// are "ours" from ones that aren't.
	containerType = Package + " 0.0.1"
	// The file in the per-container directory which we use to store our
	// per-container state.  If it isn't there, then the container isn't
	// one of our build containers.
	stateFile = Package + ".json"
)

const (
	// PullIfMissing is one of the values that BuilderOptions.PullPolicy
	// can take, signalling that the source image should be pulled from a
	// registry if a local copy of it is not already present.
	PullIfMissing = iota
	// PullAlways is one of the values that BuilderOptions.PullPolicy can
	// take, signalling that a fresh, possibly updated, copy of the image
	// should be pulled from a registry before the build proceeds.
	PullAlways
	// PullNever is one of the values that BuilderOptions.PullPolicy can
	// take, signalling that the source image should not be pulled from a
	// registry if a local copy of it is not already present.
	PullNever
)

// Builder objects are used to represent containers which are being used to
// build images.  They also carry potential updates which will be applied to
// the image's configuration when the container's contents are used to build an
// image.
type Builder struct {
	store storage.Store

	// Type is used to help identify a build container's metadata.  It
	// should not be modified.
	Type string `json:"type"`
	// FromImage is the name of the source image which was used to create
	// the container, if one was used.  It should not be modified.
	FromImage string `json:"image,omitempty"`
	// FromImageID is the ID of the source image which was used to create
	// the container, if one was used.  It should not be modified.
	FromImageID string `json:"image-id"`
	// Config is the source image's configuration.  It should not be
	// modified.
	Config []byte `json:"config,omitempty"`
	// Manifest is the source image's manifest.  It should not be modified.
	Manifest []byte `json:"manifest,omitempty"`

	// Container is the name of the build container.  It should not be modified.
	Container string `json:"container-name,omitempty"`
	// ContainerID is the ID of the build container.  It should not be modified.
	ContainerID string `json:"container-id,omitempty"`
	// MountPoint is the last location where the container's root
	// filesystem was mounted.  It should not be modified.
	MountPoint string `json:"mountpoint,omitempty"`
	// ProcessLabel is the SELinux process label associated with the container
	ProcessLabel string `json:"process-label,omitempty"`
	// MountLabel is the SELinux mount label associated with the container
	MountLabel string `json:"mount-label,omitempty"`

	// ImageAnnotations is a set of key-value pairs which is stored in the
	// image's manifest.
	ImageAnnotations map[string]string `json:"annotations,omitempty"`
	// ImageCreatedBy is a description of how this container was built.
	ImageCreatedBy string `json:"created-by,omitempty"`

	// Image metadata and runtime settings, in multiple formats.
	OCIv1  v1.Image       `json:"ociv1,omitempty"`
	Docker docker.V2Image `json:"docker,omitempty"`
	// DefaultMountsFilePath is the file path holding the mounts to be mounted in "host-path:container-path" format
	DefaultMountsFilePath string `json:"defaultMountsFilePath,omitempty"`
	CommonBuildOpts       *CommonBuildOptions
}

// BuilderInfo are used as objects to display container information
type BuilderInfo struct {
	Type                  string
	FromImage             string
	FromImageID           string
	Config                string
	Manifest              string
	Container             string
	ContainerID           string
	MountPoint            string
	ProcessLabel          string
	MountLabel            string
	ImageAnnotations      map[string]string
	ImageCreatedBy        string
	OCIv1                 v1.Image
	Docker                docker.V2Image
	DefaultMountsFilePath string
}

// GetBuildInfo gets a pointer to a Builder object and returns a BuilderInfo object from it.
// This is used in the inspect command to display Manifest and Config as string and not []byte.
func GetBuildInfo(b *Builder) BuilderInfo {
	return BuilderInfo{
		Type:                  b.Type,
		FromImage:             b.FromImage,
		FromImageID:           b.FromImageID,
		Config:                string(b.Config),
		Manifest:              string(b.Manifest),
		Container:             b.Container,
		ContainerID:           b.ContainerID,
		MountPoint:            b.MountPoint,
		ProcessLabel:          b.ProcessLabel,
		ImageAnnotations:      b.ImageAnnotations,
		ImageCreatedBy:        b.ImageCreatedBy,
		OCIv1:                 b.OCIv1,
		Docker:                b.Docker,
		DefaultMountsFilePath: b.DefaultMountsFilePath,
	}
}

// CommonBuildOptions are reseources that can be defined by flags for both buildah from and bud
type CommonBuildOptions struct {
	// AddHost is the list of hostnames to add to the resolv.conf
	AddHost []string
	//CgroupParent it the path to cgroups under which the cgroup for the container will be created.
	CgroupParent string
	//CPUPeriod limits the CPU CFS (Completely Fair Scheduler) period
	CPUPeriod uint64
	//CPUQuota limits the CPU CFS (Completely Fair Scheduler) quota
	CPUQuota int64
	//CPUShares (relative weight
	CPUShares uint64
	//CPUSetCPUs in which to allow execution (0-3, 0,1)
	CPUSetCPUs string
	//CPUSetMems memory nodes (MEMs) in which to allow execution (0-3, 0,1). Only effective on NUMA systems.
	CPUSetMems string
	//Memory limit
	Memory int64
	//MemorySwap limit value equal to memory plus swap.
	MemorySwap int64
	//SecruityOpts modify the way container security is running
	LabelOpts          []string
	SeccompProfilePath string
	ApparmorProfile    string
	//ShmSize is the shared memory size
	ShmSize string
	//Ulimit options
	Ulimit []string
	//Volumes to bind mount into the container
	Volumes []string
}

// BuilderOptions are used to initialize a new Builder.
type BuilderOptions struct {
	// FromImage is the name of the image which should be used as the
	// starting point for the container.  It can be set to an empty value
	// or "scratch" to indicate that the container should not be based on
	// an image.
	FromImage string
	// Container is a desired name for the build container.
	Container string
	// PullPolicy decides whether or not we should pull the image that
	// we're using as a base image.  It should be PullIfMissing,
	// PullAlways, or PullNever.
	PullPolicy int
	// Registry is a value which is prepended to the image's name, if it
	// needs to be pulled and the image name alone can not be resolved to a
	// reference to a source image.  No separator is implicitly added.
	Registry string
	// Transport is a value which is prepended to the image's name, if it
	// needs to be pulled and the image name alone, or the image name and
	// the registry together, can not be resolved to a reference to a
	// source image.  No separator is implicitly added.
	Transport string
	// Mount signals to NewBuilder() that the container should be mounted
	// immediately.
	Mount bool
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
	// ReportWriter is an io.Writer which will be used to log the reading
	// of the source image from a registry, if we end up pulling the image.
	ReportWriter io.Writer
	// github.com/containers/image/types SystemContext to hold credentials
	// and other authentication/authorization information.
	SystemContext *types.SystemContext
	// DefaultMountsFilePath is the file path holding the mounts to be mounted in "host-path:container-path" format
	DefaultMountsFilePath string
	CommonBuildOpts       *CommonBuildOptions
}

// ImportOptions are used to initialize a Builder from an existing container
// which was created elsewhere.
type ImportOptions struct {
	// Container is the name of the build container.
	Container string
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
}

// ImportFromImageOptions are used to initialize a Builder from an image.
type ImportFromImageOptions struct {
	// Image is the name or ID of the image we'd like to examine.
	Image string
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
	// github.com/containers/image/types SystemContext to hold information
	// about which registries we should check for completing image names
	// that don't include a domain portion.
	SystemContext *types.SystemContext
}

// NewBuilder creates a new build container.
func NewBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	return newBuilder(store, options)
}

// ImportBuilder creates a new build configuration using an already-present
// container.
func ImportBuilder(store storage.Store, options ImportOptions) (*Builder, error) {
	return importBuilder(store, options)
}

// ImportBuilderFromImage creates a new builder configuration using an image.
// The returned object can be modified and examined, but it can not be saved
// or committed because it is not associated with a working container.
func ImportBuilderFromImage(store storage.Store, options ImportFromImageOptions) (*Builder, error) {
	return importBuilderFromImage(store, options)
}

// OpenBuilder loads information about a build container given its name or ID.
func OpenBuilder(store storage.Store, container string) (*Builder, error) {
	cdir, err := store.ContainerDirectory(container)
	if err != nil {
		return nil, err
	}
	buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
	if err != nil {
		return nil, err
	}
	b := &Builder{}
	err = json.Unmarshal(buildstate, &b)
	if err != nil {
		return nil, err
	}
	if b.Type != containerType {
		return nil, errors.Errorf("container is not a %s container", Package)
	}
	b.store = store
	b.fixupConfig()
	return b, nil
}

// OpenBuilderByPath loads information about a build container given a
// path to the container's root filesystem
func OpenBuilderByPath(store storage.Store, path string) (*Builder, error) {
	containers, err := store.Containers()
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	builderMatchesPath := func(b *Builder, path string) bool {
		return (b.MountPoint == path)
	}
	for _, container := range containers {
		cdir, err := store.ContainerDirectory(container.ID)
		if err != nil {
			return nil, err
		}
		buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
		if err != nil {
			return nil, err
		}
		b := &Builder{}
		err = json.Unmarshal(buildstate, &b)
		if err == nil && b.Type == containerType && builderMatchesPath(b, abs) {
			b.store = store
			b.fixupConfig()
			return b, nil
		}
	}
	return nil, storage.ErrContainerUnknown
}

// OpenAllBuilders loads all containers which have a state file that we use in
// their data directory, typically so that they can be listed.
func OpenAllBuilders(store storage.Store) (builders []*Builder, err error) {
	containers, err := store.Containers()
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		cdir, err := store.ContainerDirectory(container.ID)
		if err != nil {
			return nil, err
		}
		buildstate, err := ioutil.ReadFile(filepath.Join(cdir, stateFile))
		if err != nil && os.IsNotExist(err) {
			continue
		}
		b := &Builder{}
		err = json.Unmarshal(buildstate, &b)
		if err == nil && b.Type == containerType {
			b.store = store
			b.fixupConfig()
			builders = append(builders, b)
		}
	}
	return builders, nil
}

// Save saves the builder's current state to the build container's metadata.
// This should not need to be called directly, as other methods of the Builder
// object take care of saving their state.
func (b *Builder) Save() error {
	buildstate, err := json.Marshal(b)
	if err != nil {
		return err
	}
	cdir, err := b.store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(cdir, stateFile), buildstate, 0600)
}
