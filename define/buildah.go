package define

import (
	"fmt"
	"io"
	"time"

	"github.com/containers/image/v5/types"
	"github.com/containers/ocicrypt/config"
	"github.com/opencontainers/runc/libcontainer/configs"
)

// PullPolicy takes the value PullIfMissing, PullAlways, PullIfNewer, or PullNever.
type PullPolicy int

const (
	// PullIfMissing is one of the values that BuilderOptions.PullPolicy
	// can take, signalling that the source image should be pulled from a
	// registry if a local copy of it is not already present.
	PullIfMissing PullPolicy = iota
	// PullAlways is one of the values that BuilderOptions.PullPolicy can
	// take, signalling that a fresh, possibly updated, copy of the image
	// should be pulled from a registry before the build proceeds.
	PullAlways
	// PullIfNewer is one of the values that BuilderOptions.PullPolicy
	// can take, signalling that the source image should only be pulled
	// from a registry if a local copy is not already present or if a
	// newer version the image is present on the repository.
	PullIfNewer
	// PullNever is one of the values that BuilderOptions.PullPolicy can
	// take, signalling that the source image should not be pulled from a
	// registry if a local copy of it is not already present.
	PullNever
)

// String converts a PullPolicy into a string.
func (p PullPolicy) String() string {
	switch p {
	case PullIfMissing:
		return "PullIfMissing"
	case PullAlways:
		return "PullAlways"
	case PullIfNewer:
		return "PullIfNewer"
	case PullNever:
		return "PullNever"
	}
	return fmt.Sprintf("unrecognized policy %d", p)
}

// CommonBuildOptions are resources that can be defined by flags for both buildah from and build-using-dockerfile
type CommonBuildOptions struct {
	// AddHost is the list of hostnames to add to the build container's /etc/hosts.
	AddHost []string
	// CgroupParent is the path to cgroups under which the cgroup for the container will be created.
	CgroupParent string
	// CPUPeriod limits the CPU CFS (Completely Fair Scheduler) period
	CPUPeriod uint64
	// CPUQuota limits the CPU CFS (Completely Fair Scheduler) quota
	CPUQuota int64
	// CPUShares (relative weight
	CPUShares uint64
	// CPUSetCPUs in which to allow execution (0-3, 0,1)
	CPUSetCPUs string
	// CPUSetMems memory nodes (MEMs) in which to allow execution (0-3, 0,1). Only effective on NUMA systems.
	CPUSetMems string
	// HTTPProxy determines whether *_proxy env vars from the build host are passed into the container.
	HTTPProxy bool
	// Memory is the upper limit (in bytes) on how much memory running containers can use.
	Memory int64
	// DNSSearch is the list of DNS search domains to add to the build container's /etc/resolv.conf
	DNSSearch []string
	// DNSServers is the list of DNS servers to add to the build container's /etc/resolv.conf
	DNSServers []string
	// DNSOptions is the list of DNS
	DNSOptions []string
	// MemorySwap limits the amount of memory and swap together.
	MemorySwap int64
	// LabelOpts is the a slice of fields of an SELinux context, given in "field:pair" format, or "disable".
	// Recognized field names are "role", "type", and "level".
	LabelOpts []string
	// OmitTimestamp forces epoch 0 as created timestamp to allow for
	// deterministic, content-addressable builds.
	OmitTimestamp bool
	// SeccompProfilePath is the pathname of a seccomp profile.
	SeccompProfilePath string
	// ApparmorProfile is the name of an apparmor profile.
	ApparmorProfile string
	// ShmSize is the "size" value to use when mounting an shmfs on the container's /dev/shm directory.
	ShmSize string
	// Ulimit specifies resource limit options, in the form type:softlimit[:hardlimit].
	// These types are recognized:
	// "core": maximum core dump size (ulimit -c)
	// "cpu": maximum CPU time (ulimit -t)
	// "data": maximum size of a process's data segment (ulimit -d)
	// "fsize": maximum size of new files (ulimit -f)
	// "locks": maximum number of file locks (ulimit -x)
	// "memlock": maximum amount of locked memory (ulimit -l)
	// "msgqueue": maximum amount of data in message queues (ulimit -q)
	// "nice": niceness adjustment (nice -n, ulimit -e)
	// "nofile": maximum number of open files (ulimit -n)
	// "nproc": maximum number of processes (ulimit -u)
	// "rss": maximum size of a process's (ulimit -m)
	// "rtprio": maximum real-time scheduling priority (ulimit -r)
	// "rttime": maximum amount of real-time execution between blocking syscalls
	// "sigpending": maximum number of pending signals (ulimit -i)
	// "stack": maximum stack size (ulimit -s)
	Ulimit []string
	// Volumes to bind mount into the container
	Volumes []string
}

// BuilderOptions are used to initialize a new Builder.
type BuilderOptions struct {
	// Args define variables that users can pass at build-time to the builder
	Args map[string]string
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
	PullPolicy PullPolicy
	// Registry is a value which is prepended to the image's name, if it
	// needs to be pulled and the image name alone can not be resolved to a
	// reference to a source image.  No separator is implicitly added.
	Registry string
	// BlobDirectory is the name of a directory in which we'll attempt
	// to store copies of layer blobs that we pull down, if any.  It should
	// already exist.
	BlobDirectory string
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
	// DefaultMountsFilePath is the file path holding the mounts to be
	// mounted in "host-path:container-path" format
	DefaultMountsFilePath string
	// Isolation controls how we handle "RUN" statements and the Run()
	// method.
	Isolation Isolation
	// NamespaceOptions controls how we set up namespaces for processes that
	// we might need to run using the container's root filesystem.
	NamespaceOptions NamespaceOptions
	// ConfigureNetwork controls whether or not network interfaces and
	// routing are configured for a new network namespace (i.e., when not
	// joining another's namespace and not just using the host's
	// namespace), effectively deciding whether or not the process has a
	// usable network.
	ConfigureNetwork NetworkConfigurationPolicy
	// CNIPluginPath is the location of CNI plugin helpers, if they should be
	// run from a location other than the default location.
	CNIPluginPath string
	// CNIConfigDir is the location of CNI configuration files, if the files in
	// the default configuration directory shouldn't be used.
	CNIConfigDir string
	// ID mapping options to use if we're setting up our own user namespace.
	IDMappingOptions *IDMappingOptions
	// Capabilities is a list of capabilities to use when
	// running commands in the container.
	Capabilities    []string
	CommonBuildOpts *CommonBuildOptions
	// Format for the container image
	Format string
	// Devices are the additional devices to add to the containers
	Devices ContainerDevices
	//DefaultEnv for containers
	DefaultEnv []string
	// MaxPullRetries is the maximum number of attempts we'll make to pull
	// any one image from the external registry if the first attempt fails.
	MaxPullRetries int
	// PullRetryDelay is how long to wait before retrying a pull attempt.
	PullRetryDelay time.Duration
	// OciDecryptConfig contains the config that can be used to decrypt an image if it is
	// encrypted if non-nil. If nil, it does not attempt to decrypt an image.
	OciDecryptConfig *config.DecryptConfig
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

// NetworkConfigurationPolicy takes the value NetworkDefault, NetworkDisabled,
// or NetworkEnabled.
type NetworkConfigurationPolicy int

const (
	// NetworkDefault is one of the values that BuilderOptions.ConfigureNetwork
	// can take, signalling that the default behavior should be used.
	NetworkDefault NetworkConfigurationPolicy = iota
	// NetworkDisabled is one of the values that BuilderOptions.ConfigureNetwork
	// can take, signalling that network interfaces should NOT be configured for
	// newly-created network namespaces.
	NetworkDisabled
	// NetworkEnabled is one of the values that BuilderOptions.ConfigureNetwork
	// can take, signalling that network interfaces should be configured for
	// newly-created network namespaces.
	NetworkEnabled
)

// String formats a NetworkConfigurationPolicy as a string.
func (p NetworkConfigurationPolicy) String() string {
	switch p {
	case NetworkDefault:
		return "NetworkDefault"
	case NetworkDisabled:
		return "NetworkDisabled"
	case NetworkEnabled:
		return "NetworkEnabled"
	}
	return fmt.Sprintf("unknown NetworkConfigurationPolicy %d", p)
}

// ContainerDevices is an alias for a slice of github.com/opencontainers/runc/libcontainer/configs.Device structures.
type ContainerDevices = []configs.Device
