package buildah

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containernetworking/cni/libcni"
	"github.com/containers/buildah/bind"
	"github.com/containers/buildah/chroot"
	"github.com/containers/buildah/util"
	"github.com/containers/libpod/pkg/secrets"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/reexec"
	units "github.com/docker/go-units"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

const (
	// runUsingRuntimeCommand is a command we use as a key for reexec
	runUsingRuntimeCommand = Package + "-oci-runtime"
)

// TerminalPolicy takes the value DefaultTerminal, WithoutTerminal, or WithTerminal.
type TerminalPolicy int

const (
	// DefaultTerminal indicates that this Run invocation should be
	// connected to a pseudoterminal if we're connected to a terminal.
	DefaultTerminal TerminalPolicy = iota
	// WithoutTerminal indicates that this Run invocation should NOT be
	// connected to a pseudoterminal.
	WithoutTerminal
	// WithTerminal indicates that this Run invocation should be connected
	// to a pseudoterminal.
	WithTerminal
)

// String converts a TerminalPoliicy into a string.
func (t TerminalPolicy) String() string {
	switch t {
	case DefaultTerminal:
		return "DefaultTerminal"
	case WithoutTerminal:
		return "WithoutTerminal"
	case WithTerminal:
		return "WithTerminal"
	}
	return fmt.Sprintf("unrecognized terminal setting %d", t)
}

// NamespaceOption controls how we set up a namespace when launching processes.
type NamespaceOption struct {
	// Name specifies the type of namespace, typically matching one of the
	// ...Namespace constants defined in
	// github.com/opencontainers/runtime-spec/specs-go.
	Name string
	// Host is used to force our processes to use the host's namespace of
	// this type.
	Host bool
	// Path is the path of the namespace to attach our process to, if Host
	// is not set.  If Host is not set and Path is also empty, a new
	// namespace will be created for the process that we're starting.
	// If Name is specs.NetworkNamespace, if Path doesn't look like an
	// absolute path, it is treated as a comma-separated list of CNI
	// configuration names which will be selected from among all of the CNI
	// network configurations which we find.
	Path string
}

// NamespaceOptions provides some helper methods for a slice of NamespaceOption
// structs.
type NamespaceOptions []NamespaceOption

// IDMappingOptions controls how we set up UID/GID mapping when we set up a
// user namespace.
type IDMappingOptions struct {
	HostUIDMapping bool
	HostGIDMapping bool
	UIDMap         []specs.LinuxIDMapping
	GIDMap         []specs.LinuxIDMapping
}

// Isolation provides a way to specify whether we're supposed to use a proper
// OCI runtime, or some other method for running commands.
type Isolation int

const (
	// IsolationDefault is whatever we think will work best.
	IsolationDefault Isolation = iota
	// IsolationOCI is a proper OCI runtime.
	IsolationOCI
	// IsolationChroot is a more chroot-like environment: less isolation,
	// but with fewer requirements.
	IsolationChroot
	// IsolationOCIRootless is a proper OCI runtime in rootless mode.
	IsolationOCIRootless
)

// String converts a Isolation into a string.
func (i Isolation) String() string {
	switch i {
	case IsolationDefault:
		return "IsolationDefault"
	case IsolationOCI:
		return "IsolationOCI"
	case IsolationChroot:
		return "IsolationChroot"
	case IsolationOCIRootless:
		return "IsolationOCIRootless"
	}
	return fmt.Sprintf("unrecognized isolation type %d", i)
}

// RunOptions can be used to alter how a command is run in the container.
type RunOptions struct {
	// Hostname is the hostname we set for the running container.
	Hostname string
	// Isolation is either IsolationDefault, IsolationOCI, IsolationChroot, or IsolationOCIRootless.
	Isolation Isolation
	// Runtime is the name of the runtime to run.  It should accept the
	// same arguments that runc does, and produce similar output.
	Runtime string
	// Args adds global arguments for the runtime.
	Args []string
	// NoPivot adds the --no-pivot runtime flag.
	NoPivot bool
	// Mounts are additional mount points which we want to provide.
	Mounts []specs.Mount
	// Env is additional environment variables to set.
	Env []string
	// User is the user as whom to run the command.
	User string
	// WorkingDir is an override for the working directory.
	WorkingDir string
	// Shell is default shell to run in a container.
	Shell string
	// Cmd is an override for the configured default command.
	Cmd []string
	// Entrypoint is an override for the configured entry point.
	Entrypoint []string
	// NamespaceOptions controls how we set up the namespaces for the process.
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
	// Terminal provides a way to specify whether or not the command should
	// be run with a pseudoterminal.  By default (DefaultTerminal), a
	// terminal is used if os.Stdout is connected to a terminal, but that
	// decision can be overridden by specifying either WithTerminal or
	// WithoutTerminal.
	Terminal TerminalPolicy
	// TerminalSize provides a way to set the number of rows and columns in
	// a pseudo-terminal, if we create one, and Stdin/Stdout/Stderr aren't
	// connected to a terminal.
	TerminalSize *specs.Box
	// The stdin/stdout/stderr descriptors to use.  If set to nil, the
	// corresponding files in the "os" package are used as defaults.
	Stdin  io.Reader `json:"-"`
	Stdout io.Writer `json:"-"`
	Stderr io.Writer `json:"-"`
	// Quiet tells the run to turn off output to stdout.
	Quiet bool
	// AddCapabilities is a list of capabilities to add to the default set.
	AddCapabilities []string
	// DropCapabilities is a list of capabilities to remove from the default set,
	// after processing the AddCapabilities set.  If a capability appears in both
	// lists, it will be dropped.
	DropCapabilities []string
}

// DefaultNamespaceOptions returns the default namespace settings from the
// runtime-tools generator library.
func DefaultNamespaceOptions() (NamespaceOptions, error) {
	options := NamespaceOptions{
		{Name: string(specs.CgroupNamespace), Host: true},
		{Name: string(specs.IPCNamespace), Host: true},
		{Name: string(specs.MountNamespace), Host: true},
		{Name: string(specs.NetworkNamespace), Host: true},
		{Name: string(specs.PIDNamespace), Host: true},
		{Name: string(specs.UserNamespace), Host: true},
		{Name: string(specs.UTSNamespace), Host: true},
	}
	g, err := generate.New("linux")
	if err != nil {
		return options, errors.Wrapf(err, "error generating new 'linux' runtime spec")
	}
	spec := g.Config
	if spec.Linux != nil {
		for _, ns := range spec.Linux.Namespaces {
			options.AddOrReplace(NamespaceOption{
				Name: string(ns.Type),
				Path: ns.Path,
			})
		}
	}
	return options, nil
}

// Find the configuration for the namespace of the given type.  If there are
// duplicates, find the _last_ one of the type, since we assume it was appended
// more recently.
func (n *NamespaceOptions) Find(namespace string) *NamespaceOption {
	for i := range *n {
		j := len(*n) - 1 - i
		if (*n)[j].Name == namespace {
			return &((*n)[j])
		}
	}
	return nil
}

// AddOrReplace either adds or replaces the configuration for a given namespace.
func (n *NamespaceOptions) AddOrReplace(options ...NamespaceOption) {
nextOption:
	for _, option := range options {
		for i := range *n {
			j := len(*n) - 1 - i
			if (*n)[j].Name == option.Name {
				(*n)[j] = option
				continue nextOption
			}
		}
		*n = append(*n, option)
	}
}

func addRlimits(ulimit []string, g *generate.Generator) error {
	var (
		ul  *units.Ulimit
		err error
	)

	for _, u := range ulimit {
		if ul, err = units.ParseUlimit(u); err != nil {
			return errors.Wrapf(err, "ulimit option %q requires name=SOFT:HARD, failed to be parsed", u)
		}

		g.AddProcessRlimits("RLIMIT_"+strings.ToUpper(ul.Name), uint64(ul.Hard), uint64(ul.Soft))
	}
	return nil
}

func addHosts(hosts []string, w io.Writer) error {
	buf := bufio.NewWriter(w)
	for _, host := range hosts {
		values := strings.SplitN(host, ":", 2)
		if len(values) != 2 {
			return errors.Errorf("unable to parse host entry %q: incorrect format", host)
		}
		if values[0] == "" {
			return errors.Errorf("hostname in host entry %q is empty", host)
		}
		if values[1] == "" {
			return errors.Errorf("IP address in host entry %q is empty", host)
		}
		fmt.Fprintf(buf, "%s\t%s\n", values[1], values[0])
	}
	return buf.Flush()
}

func addHostsToFile(hosts []string, filename string) error {
	if len(hosts) == 0 {
		return nil
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return errors.Wrapf(err, "error creating hosts file %q", filename)
	}
	defer file.Close()
	return addHosts(hosts, file)
}

func addCommonOptsToSpec(commonOpts *CommonBuildOptions, g *generate.Generator) error {
	// Resources - CPU
	if commonOpts.CPUPeriod != 0 {
		g.SetLinuxResourcesCPUPeriod(commonOpts.CPUPeriod)
	}
	if commonOpts.CPUQuota != 0 {
		g.SetLinuxResourcesCPUQuota(commonOpts.CPUQuota)
	}
	if commonOpts.CPUShares != 0 {
		g.SetLinuxResourcesCPUShares(commonOpts.CPUShares)
	}
	if commonOpts.CPUSetCPUs != "" {
		g.SetLinuxResourcesCPUCpus(commonOpts.CPUSetCPUs)
	}
	if commonOpts.CPUSetMems != "" {
		g.SetLinuxResourcesCPUMems(commonOpts.CPUSetMems)
	}

	// Resources - Memory
	if commonOpts.Memory != 0 {
		g.SetLinuxResourcesMemoryLimit(commonOpts.Memory)
	}
	if commonOpts.MemorySwap != 0 {
		g.SetLinuxResourcesMemorySwap(commonOpts.MemorySwap)
	}

	// cgroup membership
	if commonOpts.CgroupParent != "" {
		g.SetLinuxCgroupsPath(commonOpts.CgroupParent)
	}

	// Other process resource limits
	if err := addRlimits(commonOpts.Ulimit, g); err != nil {
		return err
	}

	logrus.Debugf("Resources: %#v", commonOpts)
	return nil
}

func (b *Builder) setupMounts(mountPoint string, spec *specs.Spec, bundlePath string, optionMounts []specs.Mount, bindFiles map[string]string, builtinVolumes, volumeMounts []string, shmSize string, namespaceOptions NamespaceOptions) error {
	// Start building a new list of mounts.
	var mounts []specs.Mount
	haveMount := func(destination string) bool {
		for _, mount := range mounts {
			if mount.Destination == destination {
				// Already have something to mount there.
				return true
			}
		}
		return false
	}

	ipc := namespaceOptions.Find(string(specs.IPCNamespace))
	hostIPC := ipc == nil || ipc.Host
	net := namespaceOptions.Find(string(specs.NetworkNamespace))
	hostNetwork := net == nil || net.Host
	user := namespaceOptions.Find(string(specs.UserNamespace))
	hostUser := user == nil || user.Host

	// Copy mounts from the generated list.
	mountCgroups := true
	specMounts := []specs.Mount{}
	for _, specMount := range spec.Mounts {
		// Override some of the mounts from the generated list if we're doing different things with namespaces.
		if specMount.Destination == "/dev/shm" {
			specMount.Options = []string{"nosuid", "noexec", "nodev", "mode=1777", "size=" + shmSize}
			if hostIPC && !hostUser {
				if _, err := os.Stat("/dev/shm"); err != nil && os.IsNotExist(err) {
					logrus.Debugf("/dev/shm is not present, not binding into container")
					continue
				}
				specMount = specs.Mount{
					Source:      "/dev/shm",
					Type:        "bind",
					Destination: "/dev/shm",
					Options:     []string{bind.NoBindOption, "rbind", "nosuid", "noexec", "nodev"},
				}
			}
		}
		if specMount.Destination == "/dev/mqueue" {
			if hostIPC && !hostUser {
				if _, err := os.Stat("/dev/mqueue"); err != nil && os.IsNotExist(err) {
					logrus.Debugf("/dev/mqueue is not present, not binding into container")
					continue
				}
				specMount = specs.Mount{
					Source:      "/dev/mqueue",
					Type:        "bind",
					Destination: "/dev/mqueue",
					Options:     []string{bind.NoBindOption, "rbind", "nosuid", "noexec", "nodev"},
				}
			}
		}
		if specMount.Destination == "/sys" {
			if hostNetwork && !hostUser {
				mountCgroups = false
				if _, err := os.Stat("/sys"); err != nil && os.IsNotExist(err) {
					logrus.Debugf("/sys is not present, not binding into container")
					continue
				}
				specMount = specs.Mount{
					Source:      "/sys",
					Type:        "bind",
					Destination: "/sys",
					Options:     []string{bind.NoBindOption, "rbind", "nosuid", "noexec", "nodev", "ro"},
				}
			}
		}
		specMounts = append(specMounts, specMount)
	}

	// Add a mount for the cgroups filesystem, unless we're already
	// recursively bind mounting all of /sys, in which case we shouldn't
	// bother with it.
	sysfsMount := []specs.Mount{}
	if mountCgroups {
		sysfsMount = []specs.Mount{{
			Destination: "/sys/fs/cgroup",
			Type:        "cgroup",
			Source:      "cgroup",
			Options:     []string{bind.NoBindOption, "nosuid", "noexec", "nodev", "relatime", "ro"},
		}}
	}

	// Get the list of files we need to bind into the container.
	bindFileMounts, err := runSetupBoundFiles(bundlePath, bindFiles)
	if err != nil {
		return err
	}

	// After this point we need to know the per-container persistent storage directory.
	cdir, err := b.store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return errors.Wrapf(err, "error determining work directory for container %q", b.ContainerID)
	}

	// Figure out which UID and GID to tell the secrets package to use
	// for files that it creates.
	rootUID, rootGID, err := util.GetHostRootIDs(spec)
	if err != nil {
		return err
	}

	// Get the list of secrets mounts.
	secretMounts := secrets.SecretMountsWithUIDGID(b.MountLabel, cdir, b.DefaultMountsFilePath, cdir, int(rootUID), int(rootGID))

	// Add temporary copies of the contents of volume locations at the
	// volume locations, unless we already have something there.
	copyWithTar := b.copyWithTar(nil, nil)
	builtins, err := runSetupBuiltinVolumes(b.MountLabel, mountPoint, cdir, copyWithTar, builtinVolumes, int(rootUID), int(rootGID))
	if err != nil {
		return err
	}

	// Get the list of explicitly-specified volume mounts.
	volumes, err := runSetupVolumeMounts(spec.Linux.MountLabel, volumeMounts, optionMounts)
	if err != nil {
		return err
	}

	// Add them all, in the preferred order, except where they conflict with something that was previously added.
	for _, mount := range append(append(append(append(append(volumes, builtins...), secretMounts...), bindFileMounts...), specMounts...), sysfsMount...) {
		if haveMount(mount.Destination) {
			// Already mounting something there, no need to bother with this one.
			continue
		}
		// Add the mount.
		mounts = append(mounts, mount)
	}

	// Set the list in the spec.
	spec.Mounts = mounts
	return nil
}

func runSetupBoundFiles(bundlePath string, bindFiles map[string]string) (mounts []specs.Mount, err error) {
	for dest, src := range bindFiles {
		options := []string{"rbind"}
		if strings.HasPrefix(src, bundlePath) {
			options = append(options, bind.NoBindOption)
		}
		mounts = append(mounts, specs.Mount{
			Source:      src,
			Destination: dest,
			Type:        "bind",
			Options:     options,
		})
	}
	return mounts, nil
}

func runSetupBuiltinVolumes(mountLabel, mountPoint, containerDir string, copyWithTar func(srcPath, dstPath string) error, builtinVolumes []string, rootUID, rootGID int) ([]specs.Mount, error) {
	var mounts []specs.Mount
	hostOwner := idtools.IDPair{UID: rootUID, GID: rootGID}
	// Add temporary copies of the contents of volume locations at the
	// volume locations, unless we already have something there.
	for _, volume := range builtinVolumes {
		subdir := digest.Canonical.FromString(volume).Hex()
		volumePath := filepath.Join(containerDir, "buildah-volumes", subdir)
		srcPath := filepath.Join(mountPoint, volume)
		initializeVolume := false
		// If we need to, initialize the volume path's initial contents.
		if _, err := os.Stat(volumePath); err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrapf(err, "failed to stat %q for volume %q", volumePath, volume)
			}
			logrus.Debugf("setting up built-in volume at %q", volumePath)
			if err = os.MkdirAll(volumePath, 0755); err != nil {
				return nil, errors.Wrapf(err, "error creating directory %q for volume %q", volumePath, volume)
			}
			if err = label.Relabel(volumePath, mountLabel, false); err != nil {
				return nil, errors.Wrapf(err, "error relabeling directory %q for volume %q", volumePath, volume)
			}
			initializeVolume = true
		}
		stat, err := os.Stat(srcPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrapf(err, "failed to stat %q for volume %q", srcPath, volume)
			}
			if err = idtools.MkdirAllAndChownNew(srcPath, 0755, hostOwner); err != nil {
				return nil, errors.Wrapf(err, "error creating directory %q for volume %q", srcPath, volume)
			}
			if stat, err = os.Stat(srcPath); err != nil {
				return nil, errors.Wrapf(err, "failed to stat %q for volume %q", srcPath, volume)
			}
		}
		if initializeVolume {
			if err = os.Chmod(volumePath, stat.Mode().Perm()); err != nil {
				return nil, errors.Wrapf(err, "failed to chmod %q for volume %q", volumePath, volume)
			}
			if err = os.Chown(volumePath, int(stat.Sys().(*syscall.Stat_t).Uid), int(stat.Sys().(*syscall.Stat_t).Gid)); err != nil {
				return nil, errors.Wrapf(err, "error chowning directory %q for volume %q", volumePath, volume)
			}
			if err = copyWithTar(srcPath, volumePath); err != nil && !os.IsNotExist(errors.Cause(err)) {
				return nil, errors.Wrapf(err, "error populating directory %q for volume %q using contents of %q", volumePath, volume, srcPath)
			}
		}
		// Add the bind mount.
		mounts = append(mounts, specs.Mount{
			Source:      volumePath,
			Destination: volume,
			Type:        "bind",
			Options:     []string{"bind"},
		})
	}
	return mounts, nil
}

func runSetupVolumeMounts(mountLabel string, volumeMounts []string, optionMounts []specs.Mount) ([]specs.Mount, error) {
	var mounts []specs.Mount

	parseMount := func(host, container string, options []string) (specs.Mount, error) {
		var foundrw, foundro, foundz, foundZ bool
		var rootProp string
		for _, opt := range options {
			switch opt {
			case "rw":
				foundrw = true
			case "ro":
				foundro = true
			case "z":
				foundz = true
			case "Z":
				foundZ = true
			case "private", "rprivate", "slave", "rslave", "shared", "rshared":
				rootProp = opt
			}
		}
		if !foundrw && !foundro {
			options = append(options, "rw")
		}
		if foundz {
			if err := label.Relabel(host, mountLabel, true); err != nil {
				return specs.Mount{}, errors.Wrapf(err, "relabeling %q failed", host)
			}
		}
		if foundZ {
			if err := label.Relabel(host, mountLabel, false); err != nil {
				return specs.Mount{}, errors.Wrapf(err, "relabeling %q failed", host)
			}
		}
		if rootProp == "" {
			options = append(options, "private")
		}
		return specs.Mount{
			Destination: container,
			Type:        "bind",
			Source:      host,
			Options:     options,
		}, nil
	}
	// Bind mount volumes specified for this particular Run() invocation
	for _, i := range optionMounts {
		logrus.Debugf("setting up mounted volume at %q", i.Destination)
		mount, err := parseMount(i.Source, i.Destination, append(i.Options, "rbind"))
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}
	// Bind mount volumes given by the user when the container was created
	for _, i := range volumeMounts {
		var options []string
		spliti := strings.Split(i, ":")
		if len(spliti) > 2 {
			options = strings.Split(spliti[2], ",")
		}
		options = append(options, "rbind")
		mount, err := parseMount(spliti[0], spliti[1], options)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
}

// addNetworkConfig copies files from host and sets them up to bind mount into container
func (b *Builder) addNetworkConfig(rdir, hostPath string, chownOpts *idtools.IDPair) (string, error) {
	copyFileWithTar := b.copyFileWithTar(chownOpts, nil)

	cfile := filepath.Join(rdir, filepath.Base(hostPath))

	if err := copyFileWithTar(hostPath, cfile); err != nil {
		return "", errors.Wrapf(err, "error copying %q for container %q", cfile, b.ContainerID)
	}

	if err := label.Relabel(cfile, b.MountLabel, false); err != nil {
		return "", errors.Wrapf(err, "error relabeling %q in container %q", cfile, b.ContainerID)
	}

	return cfile, nil
}

func setupMaskedPaths(g *generate.Generator) {
	for _, mp := range []string{
		"/proc/acpi",
		"/proc/kcore",
		"/proc/keys",
		"/proc/latency_stats",
		"/proc/timer_list",
		"/proc/timer_stats",
		"/proc/sched_debug",
		"/proc/scsi",
		"/sys/firmware",
	} {
		g.AddLinuxMaskedPaths(mp)
	}
}

func setupReadOnlyPaths(g *generate.Generator) {
	for _, rp := range []string{
		"/proc/asound",
		"/proc/bus",
		"/proc/fs",
		"/proc/irq",
		"/proc/sys",
		"/proc/sysrq-trigger",
	} {
		g.AddLinuxReadonlyPaths(rp)
	}
}

func setupCapAdd(g *generate.Generator, caps ...string) error {
	for _, cap := range caps {
		if err := g.AddProcessCapabilityBounding(cap); err != nil {
			return errors.Wrapf(err, "error adding %q to the bounding capability set", cap)
		}
		if err := g.AddProcessCapabilityEffective(cap); err != nil {
			return errors.Wrapf(err, "error adding %q to the effective capability set", cap)
		}
		if err := g.AddProcessCapabilityInheritable(cap); err != nil {
			return errors.Wrapf(err, "error adding %q to the inheritable capability set", cap)
		}
		if err := g.AddProcessCapabilityPermitted(cap); err != nil {
			return errors.Wrapf(err, "error adding %q to the permitted capability set", cap)
		}
		if err := g.AddProcessCapabilityAmbient(cap); err != nil {
			return errors.Wrapf(err, "error adding %q to the ambient capability set", cap)
		}
	}
	return nil
}

func setupCapDrop(g *generate.Generator, caps ...string) error {
	for _, cap := range caps {
		if err := g.DropProcessCapabilityBounding(cap); err != nil {
			return errors.Wrapf(err, "error removing %q from the bounding capability set", cap)
		}
		if err := g.DropProcessCapabilityEffective(cap); err != nil {
			return errors.Wrapf(err, "error removing %q from the effective capability set", cap)
		}
		if err := g.DropProcessCapabilityInheritable(cap); err != nil {
			return errors.Wrapf(err, "error removing %q from the inheritable capability set", cap)
		}
		if err := g.DropProcessCapabilityPermitted(cap); err != nil {
			return errors.Wrapf(err, "error removing %q from the permitted capability set", cap)
		}
		if err := g.DropProcessCapabilityAmbient(cap); err != nil {
			return errors.Wrapf(err, "error removing %q from the ambient capability set", cap)
		}
	}
	return nil
}

func setupCapabilities(g *generate.Generator, firstAdds, firstDrops, secondAdds, secondDrops []string) error {
	g.ClearProcessCapabilities()
	if err := setupCapAdd(g, util.DefaultCapabilities...); err != nil {
		return err
	}
	if err := setupCapAdd(g, firstAdds...); err != nil {
		return err
	}
	if err := setupCapDrop(g, firstDrops...); err != nil {
		return err
	}
	if err := setupCapAdd(g, secondAdds...); err != nil {
		return err
	}
	return setupCapDrop(g, secondDrops...)
}

func setupTerminal(g *generate.Generator, terminalPolicy TerminalPolicy, terminalSize *specs.Box) {
	switch terminalPolicy {
	case DefaultTerminal:
		onTerminal := terminal.IsTerminal(unix.Stdin) && terminal.IsTerminal(unix.Stdout) && terminal.IsTerminal(unix.Stderr)
		if onTerminal {
			logrus.Debugf("stdio is a terminal, defaulting to using a terminal")
		} else {
			logrus.Debugf("stdio is not a terminal, defaulting to not using a terminal")
		}
		g.SetProcessTerminal(onTerminal)
	case WithTerminal:
		g.SetProcessTerminal(true)
	case WithoutTerminal:
		g.SetProcessTerminal(false)
	}
	if terminalSize != nil {
		g.SetProcessConsoleSize(terminalSize.Width, terminalSize.Height)
	}
}

func setupNamespaces(g *generate.Generator, namespaceOptions NamespaceOptions, idmapOptions IDMappingOptions, policy NetworkConfigurationPolicy) (configureNetwork bool, configureNetworks []string, configureUTS bool, err error) {
	// Set namespace options in the container configuration.
	configureUserns := false
	specifiedNetwork := false
	for _, namespaceOption := range namespaceOptions {
		switch namespaceOption.Name {
		case string(specs.UserNamespace):
			configureUserns = false
			if !namespaceOption.Host && namespaceOption.Path == "" {
				configureUserns = true
			}
		case string(specs.NetworkNamespace):
			specifiedNetwork = true
			configureNetwork = false
			if !namespaceOption.Host && (namespaceOption.Path == "" || !filepath.IsAbs(namespaceOption.Path)) {
				if namespaceOption.Path != "" && !filepath.IsAbs(namespaceOption.Path) {
					configureNetworks = strings.Split(namespaceOption.Path, ",")
					namespaceOption.Path = ""
				}
				configureNetwork = (policy != NetworkDisabled)
			}
		case string(specs.UTSNamespace):
			configureUTS = false
			if !namespaceOption.Host && namespaceOption.Path == "" {
				configureUTS = true
			}
		}
		if namespaceOption.Host {
			if err := g.RemoveLinuxNamespace(namespaceOption.Name); err != nil {
				return false, nil, false, errors.Wrapf(err, "error removing %q namespace for run", namespaceOption.Name)
			}
		} else if err := g.AddOrReplaceLinuxNamespace(namespaceOption.Name, namespaceOption.Path); err != nil {
			if namespaceOption.Path == "" {
				return false, nil, false, errors.Wrapf(err, "error adding new %q namespace for run", namespaceOption.Name)
			}
			return false, nil, false, errors.Wrapf(err, "error adding %q namespace %q for run", namespaceOption.Name, namespaceOption.Path)
		}
	}

	// If we've got mappings, we're going to have to create a user namespace.
	if len(idmapOptions.UIDMap) > 0 || len(idmapOptions.GIDMap) > 0 || configureUserns {
		if err := g.AddOrReplaceLinuxNamespace(specs.UserNamespace, ""); err != nil {
			return false, nil, false, errors.Wrapf(err, "error adding new %q namespace for run", string(specs.UserNamespace))
		}
		hostUidmap, hostGidmap, err := util.GetHostIDMappings("")
		if err != nil {
			return false, nil, false, err
		}
		for _, m := range idmapOptions.UIDMap {
			g.AddLinuxUIDMapping(m.HostID, m.ContainerID, m.Size)
		}
		if len(idmapOptions.UIDMap) == 0 {
			for _, m := range hostUidmap {
				g.AddLinuxUIDMapping(m.ContainerID, m.ContainerID, m.Size)
			}
		}
		for _, m := range idmapOptions.GIDMap {
			g.AddLinuxGIDMapping(m.HostID, m.ContainerID, m.Size)
		}
		if len(idmapOptions.GIDMap) == 0 {
			for _, m := range hostGidmap {
				g.AddLinuxGIDMapping(m.ContainerID, m.ContainerID, m.Size)
			}
		}
		if !specifiedNetwork {
			if err := g.AddOrReplaceLinuxNamespace(specs.NetworkNamespace, ""); err != nil {
				return false, nil, false, errors.Wrapf(err, "error adding new %q namespace for run", string(specs.NetworkNamespace))
			}
			configureNetwork = (policy != NetworkDisabled)
		}
	} else {
		if err := g.RemoveLinuxNamespace(specs.UserNamespace); err != nil {
			return false, nil, false, errors.Wrapf(err, "error removing %q namespace for run", string(specs.UserNamespace))
		}
		if !specifiedNetwork {
			if err := g.RemoveLinuxNamespace(specs.NetworkNamespace); err != nil {
				return false, nil, false, errors.Wrapf(err, "error removing %q namespace for run", string(specs.NetworkNamespace))
			}
		}
	}
	if configureNetwork {
		for name, val := range util.DefaultNetworkSysctl {
			g.AddLinuxSysctl(name, val)
		}
	}
	return configureNetwork, configureNetworks, configureUTS, nil
}

// Search for a command that isn't given as an absolute path using the $PATH
// under the rootfs.  We can't resolve absolute symbolic links without
// chroot()ing, which we may not be able to do, so just accept a link as a
// valid resolution.
func runLookupPath(g *generate.Generator, command []string) []string {
	// Look for the configured $PATH.
	spec := g.Spec()
	envPath := ""
	for i := range spec.Process.Env {
		if strings.HasPrefix(spec.Process.Env[i], "PATH=") {
			envPath = spec.Process.Env[i]
		}
	}
	// If there is no configured $PATH, supply one.
	if envPath == "" {
		defaultPath := "/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"
		envPath = "PATH=" + defaultPath
		g.AddProcessEnv("PATH", defaultPath)
	}
	// No command, nothing to do.
	if len(command) == 0 {
		return command
	}
	// Command is already an absolute path, use it as-is.
	if filepath.IsAbs(command[0]) {
		return command
	}
	// For each element in the PATH,
	for _, pathEntry := range filepath.SplitList(envPath[5:]) {
		// if it's the empty string, it's ".", which is the Cwd,
		if pathEntry == "" {
			pathEntry = spec.Process.Cwd
		}
		// build the absolute path which it might be,
		candidate := filepath.Join(pathEntry, command[0])
		// check if it's there,
		if fi, err := os.Lstat(filepath.Join(spec.Root.Path, candidate)); fi != nil && err == nil {
			// and if it's not a directory, and either a symlink or executable,
			if !fi.IsDir() && ((fi.Mode()&os.ModeSymlink != 0) || (fi.Mode()&0111 != 0)) {
				// use that.
				return append([]string{candidate}, command[1:]...)
			}
		}
	}
	return command
}

func (b *Builder) configureUIDGID(g *generate.Generator, mountPoint string, options RunOptions) error {
	// Set the user UID/GID/supplemental group list/capabilities lists.
	user, err := b.user(mountPoint, options.User)
	if err != nil {
		return err
	}
	if err := setupCapabilities(g, b.AddCapabilities, b.DropCapabilities, options.AddCapabilities, options.DropCapabilities); err != nil {
		return err
	}
	g.SetProcessUID(user.UID)
	g.SetProcessGID(user.GID)
	for _, gid := range user.AdditionalGids {
		g.AddProcessAdditionalGid(gid)
	}

	// Remove capabilities if not running as root except Bounding set
	if user.UID != 0 {
		bounding := g.Config.Process.Capabilities.Bounding
		g.ClearProcessCapabilities()
		g.Config.Process.Capabilities.Bounding = bounding
	}

	return nil
}

func (b *Builder) configureEnvironment(g *generate.Generator, options RunOptions) {
	g.ClearProcessEnv()
	for _, envSpec := range append(b.Env(), options.Env...) {
		env := strings.SplitN(envSpec, "=", 2)
		if len(env) > 1 {
			g.AddProcessEnv(env[0], env[1])
		}
	}

	for src, dest := range b.Args {
		g.AddProcessEnv(src, dest)
	}
}

func (b *Builder) configureNamespaces(g *generate.Generator, options RunOptions) (bool, []string, error) {
	defaultNamespaceOptions, err := DefaultNamespaceOptions()
	if err != nil {
		return false, nil, err
	}

	namespaceOptions := defaultNamespaceOptions
	namespaceOptions.AddOrReplace(b.NamespaceOptions...)
	namespaceOptions.AddOrReplace(options.NamespaceOptions...)

	networkPolicy := options.ConfigureNetwork
	if networkPolicy == NetworkDefault {
		networkPolicy = b.ConfigureNetwork
	}

	configureNetwork, configureNetworks, configureUTS, err := setupNamespaces(g, namespaceOptions, b.IDMappingOptions, networkPolicy)
	if err != nil {
		return false, nil, err
	}

	if configureUTS {
		if options.Hostname != "" {
			g.SetHostname(options.Hostname)
		} else if b.Hostname() != "" {
			g.SetHostname(b.Hostname())
		}
	} else {
		g.SetHostname("")
	}
	return configureNetwork, configureNetworks, nil
}

// Run runs the specified command in the container's root filesystem.
func (b *Builder) Run(command []string, options RunOptions) error {
	p, err := ioutil.TempDir("", Package)
	if err != nil {
		return errors.Wrapf(err, "run: error creating temporary directory under %q", os.TempDir())
	}
	// On some hosts like AH, /tmp is a symlink and we need an
	// absolute path.
	path, err := filepath.EvalSymlinks(p)
	if err != nil {
		return errors.Wrapf(err, "run: error evaluating %q for symbolic links", p)
	}
	logrus.Debugf("using %q to hold bundle data", path)
	defer func() {
		if err2 := os.RemoveAll(path); err2 != nil {
			logrus.Errorf("error removing %q: %v", path, err2)
		}
	}()

	gp, err := generate.New("linux")
	if err != nil {
		return errors.Wrapf(err, "error generating new 'linux' runtime spec")
	}
	g := &gp

	isolation := options.Isolation
	if isolation == IsolationDefault {
		isolation = b.Isolation
		if isolation == IsolationDefault {
			isolation = IsolationOCI
		}
	}
	if err := checkAndOverrideIsolationOptions(isolation, &options); err != nil {
		return err
	}

	b.configureEnvironment(g, options)

	if b.CommonBuildOpts == nil {
		return errors.Errorf("Invalid format on container you must recreate the container")
	}

	if err := addCommonOptsToSpec(b.CommonBuildOpts, g); err != nil {
		return err
	}

	if options.WorkingDir != "" {
		g.SetProcessCwd(options.WorkingDir)
	} else if b.WorkDir() != "" {
		g.SetProcessCwd(b.WorkDir())
	}
	setupSelinux(g, b.ProcessLabel, b.MountLabel)
	mountPoint, err := b.Mount(b.MountLabel)
	if err != nil {
		return errors.Wrapf(err, "error mounting container %q", b.ContainerID)
	}
	defer func() {
		if err := b.Unmount(); err != nil {
			logrus.Errorf("error unmounting container: %v", err)
		}
	}()
	g.SetRootPath(mountPoint)
	if len(command) > 0 {
		command = runLookupPath(g, command)
		g.SetProcessArgs(command)
	} else {
		g.SetProcessArgs(nil)
	}

	setupMaskedPaths(g)
	setupReadOnlyPaths(g)

	setupTerminal(g, options.Terminal, options.TerminalSize)

	configureNetwork, configureNetworks, err := b.configureNamespaces(g, options)
	if err != nil {
		return err
	}

	if err := b.configureUIDGID(g, mountPoint, options); err != nil {
		return err
	}

	g.SetProcessApparmorProfile(b.CommonBuildOpts.ApparmorProfile)

	// Now grab the spec from the generator.  Set the generator to nil so that future contributors
	// will quickly be able to tell that they're supposed to be modifying the spec directly from here.
	spec := g.Spec()
	g = nil

	logrus.Debugf("ensuring working directory %q exists", filepath.Join(mountPoint, spec.Process.Cwd))
	if err = os.MkdirAll(filepath.Join(mountPoint, spec.Process.Cwd), 0755); err != nil {
		return errors.Wrapf(err, "error ensuring working directory %q exists", spec.Process.Cwd)
	}

	// Set the seccomp configuration using the specified profile name.  Some syscalls are
	// allowed if certain capabilities are to be granted (example: CAP_SYS_CHROOT and chroot),
	// so we sorted out the capabilities lists first.
	if err = setupSeccomp(spec, b.CommonBuildOpts.SeccompProfilePath); err != nil {
		return err
	}

	// Figure out who owns files that will appear to be owned by UID/GID 0 in the container.
	rootUID, rootGID, err := util.GetHostRootIDs(spec)
	if err != nil {
		return err
	}
	rootIDPair := &idtools.IDPair{UID: int(rootUID), GID: int(rootGID)}

	bindFiles := make(map[string]string)
	namespaceOptions := append(b.NamespaceOptions, options.NamespaceOptions...)
	networkNamespace := namespaceOptions.Find(string(specs.NetworkNamespace))
	if networkNamespace == nil || networkNamespace.Host || networkNamespace.Path != "" {
		hostFile, err := b.addNetworkConfig(path, "/etc/hosts", rootIDPair)
		if err != nil {
			return err
		}
		bindFiles["/etc/hosts"] = hostFile

		if err := addHostsToFile(b.CommonBuildOpts.AddHost, hostFile); err != nil {
			return err
		}

		resolvFile, err := b.addNetworkConfig(path, "/etc/resolv.conf", rootIDPair)
		if err != nil {
			return err
		}
		bindFiles["/etc/resolv.conf"] = resolvFile
	}

	err = b.setupMounts(mountPoint, spec, path, options.Mounts, bindFiles, b.Volumes(), b.CommonBuildOpts.Volumes, b.CommonBuildOpts.ShmSize, namespaceOptions)
	if err != nil {
		return errors.Wrapf(err, "error resolving mountpoints for container %q", b.ContainerID)
	}

	if options.CNIConfigDir == "" {
		options.CNIConfigDir = b.CNIConfigDir
		if b.CNIConfigDir == "" {
			options.CNIConfigDir = util.DefaultCNIConfigDir
		}
	}
	if options.CNIPluginPath == "" {
		options.CNIPluginPath = b.CNIPluginPath
		if b.CNIPluginPath == "" {
			options.CNIPluginPath = util.DefaultCNIPluginPath
		}
	}

	switch isolation {
	case IsolationOCI:
		// The default is --rootless=auto, which makes troubleshooting a bit harder.
		// rootlessFlag := []string{"--rootless=false"}
		// for _, arg := range options.Args {
		// 	if strings.HasPrefix(arg, "--rootless") {
		// 		rootlessFlag = nil
		// 	}
		// }
		// options.Args = append(options.Args, rootlessFlag...)
		var moreCreateArgs []string
		if options.NoPivot {
			moreCreateArgs = []string{"--no-pivot"}
		} else {
			moreCreateArgs = nil
		}
		err = b.runUsingRuntimeSubproc(options, configureNetwork, configureNetworks, moreCreateArgs, spec, mountPoint, path, Package+"-"+filepath.Base(path))
	case IsolationChroot:
		err = chroot.RunUsingChroot(spec, path, options.Stdin, options.Stdout, options.Stderr)
	case IsolationOCIRootless:
		if err := setupRootlessSpecChanges(spec, path, rootUID, rootGID); err != nil {
			return err
		}
		rootlessFlag := []string{"--rootless=true"}
		for _, arg := range options.Args {
			if strings.HasPrefix(arg, "--rootless") {
				rootlessFlag = nil
			}
		}
		options.Args = append(options.Args, rootlessFlag...)
		err = b.runUsingRuntimeSubproc(options, configureNetwork, configureNetworks, []string{"--no-new-keyring"}, spec, mountPoint, path, Package+"-"+filepath.Base(path))
	default:
		err = errors.Errorf("don't know how to run this command")
	}
	return err
}

func checkAndOverrideIsolationOptions(isolation Isolation, options *RunOptions) error {
	switch isolation {
	case IsolationOCIRootless:
		if ns := options.NamespaceOptions.Find(string(specs.IPCNamespace)); ns == nil || ns.Host {
			logrus.Debugf("Forcing use of an IPC namespace.")
		}
		options.NamespaceOptions.AddOrReplace(NamespaceOption{Name: string(specs.IPCNamespace)})
		hostNetworking := true
		networkNamespacePath := ""
		if ns := options.NamespaceOptions.Find(string(specs.NetworkNamespace)); ns != nil {
			hostNetworking = ns.Host
			networkNamespacePath = ns.Path
			if !hostNetworking && networkNamespacePath != "" && !filepath.IsAbs(networkNamespacePath) {
				logrus.Debugf("Disabling network namespace configuration.")
				networkNamespacePath = ""
			}
		}
		options.NamespaceOptions.AddOrReplace(NamespaceOption{
			Name: string(specs.NetworkNamespace),
			Host: hostNetworking,
			Path: networkNamespacePath,
		})
		if ns := options.NamespaceOptions.Find(string(specs.PIDNamespace)); ns == nil || ns.Host {
			logrus.Debugf("Forcing use of a PID namespace.")
		}
		options.NamespaceOptions.AddOrReplace(NamespaceOption{Name: string(specs.PIDNamespace), Host: false})
		if ns := options.NamespaceOptions.Find(string(specs.UserNamespace)); ns == nil || ns.Host {
			logrus.Debugf("Forcing use of a user namespace.")
		}
		options.NamespaceOptions.AddOrReplace(NamespaceOption{Name: string(specs.UserNamespace)})
		if ns := options.NamespaceOptions.Find(string(specs.UTSNamespace)); ns != nil && !ns.Host {
			logrus.Debugf("Disabling UTS namespace.")
		}
		options.NamespaceOptions.AddOrReplace(NamespaceOption{Name: string(specs.UTSNamespace), Host: true})
	case IsolationOCI:
		pidns := options.NamespaceOptions.Find(string(specs.PIDNamespace))
		userns := options.NamespaceOptions.Find(string(specs.UserNamespace))
		if (pidns == nil || pidns.Host) && (userns != nil && !userns.Host) {
			return fmt.Errorf("not allowed to mix host PID namespace with container user namespace")
		}
	}
	return nil
}

func setupRootlessSpecChanges(spec *specs.Spec, bundleDir string, rootUID, rootGID uint32) error {
	spec.Hostname = ""
	spec.Process.User.AdditionalGids = nil
	spec.Linux.Resources = nil

	emptyDir := filepath.Join(bundleDir, "empty")
	if err := os.Mkdir(emptyDir, 0); err != nil {
		return errors.Wrapf(err, "error creating %q", emptyDir)
	}

	// Replace /sys with a read-only bind mount.
	mounts := []specs.Mount{
		{
			Source:      "/dev",
			Destination: "/dev",
			Type:        "tmpfs",
			Options:     []string{"private", "strictatime", "noexec", "nosuid", "mode=755", "size=65536k"},
		},
		{
			Source:      "mqueue",
			Destination: "/dev/mqueue",
			Type:        "mqueue",
			Options:     []string{"private", "nodev", "noexec", "nosuid"},
		},
		{
			Source:      "pts",
			Destination: "/dev/pts",
			Type:        "devpts",
			Options:     []string{"private", "noexec", "nosuid", "newinstance", "ptmxmode=0666", "mode=0620"},
		},
		{
			Source:      "shm",
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Options:     []string{"private", "nodev", "noexec", "nosuid", "mode=1777", "size=65536k"},
		},
		{
			Source:      "/proc",
			Destination: "/proc",
			Type:        "proc",
			Options:     []string{"private", "nodev", "noexec", "nosuid"},
		},
		{
			Source:      "/sys",
			Destination: "/sys",
			Type:        "bind",
			Options:     []string{bind.NoBindOption, "rbind", "private", "nodev", "noexec", "nosuid", "ro"},
		},
	}
	// Cover up /sys/fs/cgroup and /sys/fs/selinux, if they exist in our source for /sys.
	if _, err := os.Stat("/sys/fs/cgroup"); err == nil {
		spec.Linux.MaskedPaths = append(spec.Linux.MaskedPaths, "/sys/fs/cgroup")
	}
	if _, err := os.Stat("/sys/fs/selinux"); err == nil {
		spec.Linux.MaskedPaths = append(spec.Linux.MaskedPaths, "/sys/fs/selinux")
	}
	// Keep anything that isn't under /dev, /proc, or /sys.
	for i := range spec.Mounts {
		if spec.Mounts[i].Destination == "/dev" || strings.HasPrefix(spec.Mounts[i].Destination, "/dev/") ||
			spec.Mounts[i].Destination == "/proc" || strings.HasPrefix(spec.Mounts[i].Destination, "/proc/") ||
			spec.Mounts[i].Destination == "/sys" || strings.HasPrefix(spec.Mounts[i].Destination, "/sys/") {
			continue
		}
		mounts = append(mounts, spec.Mounts[i])
	}
	spec.Mounts = mounts
	return nil
}

type runUsingRuntimeSubprocOptions struct {
	Options           RunOptions
	Spec              *specs.Spec
	RootPath          string
	BundlePath        string
	ConfigureNetwork  bool
	ConfigureNetworks []string
	MoreCreateArgs    []string
	ContainerName     string
}

func (b *Builder) runUsingRuntimeSubproc(options RunOptions, configureNetwork bool, configureNetworks, moreCreateArgs []string, spec *specs.Spec, rootPath, bundlePath, containerName string) (err error) {
	var confwg sync.WaitGroup
	config, conferr := json.Marshal(runUsingRuntimeSubprocOptions{
		Options:           options,
		Spec:              spec,
		RootPath:          rootPath,
		BundlePath:        bundlePath,
		ConfigureNetwork:  configureNetwork,
		ConfigureNetworks: configureNetworks,
		MoreCreateArgs:    moreCreateArgs,
		ContainerName:     containerName,
	})
	if conferr != nil {
		return errors.Wrapf(conferr, "error encoding configuration for %q", runUsingRuntimeCommand)
	}
	cmd := reexec.Command(runUsingRuntimeCommand)
	cmd.Dir = bundlePath
	cmd.Stdin = options.Stdin
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = options.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = options.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	cmd.Env = append(os.Environ(), fmt.Sprintf("LOGLEVEL=%d", logrus.GetLevel()))
	preader, pwriter, err := os.Pipe()
	if err != nil {
		return errors.Wrapf(err, "error creating configuration pipe")
	}
	confwg.Add(1)
	go func() {
		_, conferr = io.Copy(pwriter, bytes.NewReader(config))
		if conferr != nil {
			conferr = errors.Wrapf(conferr, "error while copying configuration down pipe to child process")
		}
		confwg.Done()
	}()
	cmd.ExtraFiles = append([]*os.File{preader}, cmd.ExtraFiles...)
	defer preader.Close()
	defer pwriter.Close()
	err = cmd.Run()
	if err != nil {
		err = errors.Wrapf(err, "error while running runtime")
	}
	confwg.Wait()
	if err == nil {
		return conferr
	}
	if conferr != nil {
		logrus.Debugf("%v", conferr)
	}
	return err
}

func init() {
	reexec.Register(runUsingRuntimeCommand, runUsingRuntimeMain)
}

func runUsingRuntimeMain() {
	var options runUsingRuntimeSubprocOptions
	// Set logging.
	if level := os.Getenv("LOGLEVEL"); level != "" {
		if ll, err := strconv.Atoi(level); err == nil {
			logrus.SetLevel(logrus.Level(ll))
		}
	}
	// Unpack our configuration.
	confPipe := os.NewFile(3, "confpipe")
	if confPipe == nil {
		fmt.Fprintf(os.Stderr, "error reading options pipe\n")
		os.Exit(1)
	}
	defer confPipe.Close()
	if err := json.NewDecoder(confPipe).Decode(&options); err != nil {
		fmt.Fprintf(os.Stderr, "error decoding options: %v\n", err)
		os.Exit(1)
	}
	// Set ourselves up to read the container's exit status.  We're doing this in a child process
	// so that we won't mess with the setting in a caller of the library. This stubs to OS specific
	// calls
	if err := setChildProcess(); err != nil {
		os.Exit(1)
	}
	// Run the container, start to finish.
	status, err := runUsingRuntime(options.Options, options.ConfigureNetwork, options.ConfigureNetworks, options.MoreCreateArgs, options.Spec, options.RootPath, options.BundlePath, options.ContainerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running container: %v\n", err)
		os.Exit(1)
	}
	// Pass the container's exit status back to the caller by exiting with the same status.
	if status.Exited() {
		os.Exit(status.ExitStatus())
	} else if status.Signaled() {
		fmt.Fprintf(os.Stderr, "container exited on %s\n", status.Signal())
		os.Exit(1)
	}
	os.Exit(1)
}

func runUsingRuntime(options RunOptions, configureNetwork bool, configureNetworks, moreCreateArgs []string, spec *specs.Spec, rootPath, bundlePath, containerName string) (wstatus unix.WaitStatus, err error) {
	// Lock the caller to a single OS-level thread.
	runtime.LockOSThread()

	// Set up bind mounts for things that a namespaced user might not be able to get to directly.
	unmountAll, err := bind.SetupIntermediateMountNamespace(spec, bundlePath)
	if unmountAll != nil {
		defer func() {
			if err := unmountAll(); err != nil {
				logrus.Error(err)
			}
		}()
	}
	if err != nil {
		return 1, err
	}

	// Write the runtime configuration.
	specbytes, err := json.Marshal(spec)
	if err != nil {
		return 1, errors.Wrapf(err, "error encoding configuration %#v as json", spec)
	}
	if err = ioutils.AtomicWriteFile(filepath.Join(bundlePath, "config.json"), specbytes, 0600); err != nil {
		return 1, errors.Wrapf(err, "error storing runtime configuration in %q", filepath.Join(bundlePath, "config.json"))
	}

	logrus.Debugf("config = %v", string(specbytes))

	// Decide which runtime to use.
	runtime := options.Runtime
	if runtime == "" {
		runtime = util.Runtime()
	}

	// Default to just passing down our stdio.
	getCreateStdio := func() (io.ReadCloser, io.WriteCloser, io.WriteCloser) {
		return os.Stdin, os.Stdout, os.Stderr
	}

	// Figure out how we're doing stdio handling, and create pipes and sockets.
	var stdio sync.WaitGroup
	var consoleListener *net.UnixListener
	var errorFds, closeBeforeReadingErrorFds []int
	stdioPipe := make([][]int, 3)
	copyConsole := false
	copyPipes := false
	finishCopy := make([]int, 2)
	if err = unix.Pipe(finishCopy); err != nil {
		return 1, errors.Wrapf(err, "error creating pipe for notifying to stop stdio")
	}
	finishedCopy := make(chan struct{})
	if spec.Process != nil {
		if spec.Process.Terminal {
			copyConsole = true
			// Create a listening socket for accepting the container's terminal's PTY master.
			socketPath := filepath.Join(bundlePath, "console.sock")
			consoleListener, err = net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
			if err != nil {
				return 1, errors.Wrapf(err, "error creating socket %q to receive terminal descriptor", consoleListener.Addr())
			}
			// Add console socket arguments.
			moreCreateArgs = append(moreCreateArgs, "--console-socket", socketPath)
		} else {
			copyPipes = true
			// Figure out who should own the pipes.
			uid, gid, err := util.GetHostRootIDs(spec)
			if err != nil {
				return 1, err
			}
			// Create stdio pipes.
			if stdioPipe, err = runMakeStdioPipe(int(uid), int(gid)); err != nil {
				return 1, err
			}
			errorFds = []int{stdioPipe[unix.Stdout][0], stdioPipe[unix.Stderr][0]}
			closeBeforeReadingErrorFds = []int{stdioPipe[unix.Stdout][1], stdioPipe[unix.Stderr][1]}
			// Set stdio to our pipes.
			getCreateStdio = func() (io.ReadCloser, io.WriteCloser, io.WriteCloser) {
				stdin := os.NewFile(uintptr(stdioPipe[unix.Stdin][0]), "/dev/stdin")
				stdout := os.NewFile(uintptr(stdioPipe[unix.Stdout][1]), "/dev/stdout")
				stderr := os.NewFile(uintptr(stdioPipe[unix.Stderr][1]), "/dev/stderr")
				return stdin, stdout, stderr
			}
		}
	} else {
		if options.Quiet {
			// Discard stdout.
			getCreateStdio = func() (io.ReadCloser, io.WriteCloser, io.WriteCloser) {
				return os.Stdin, nil, os.Stderr
			}
		}
	}

	// Build the commands that we'll execute.
	pidFile := filepath.Join(bundlePath, "pid")
	args := append(append(append(options.Args, "create", "--bundle", bundlePath, "--pid-file", pidFile), moreCreateArgs...), containerName)
	create := exec.Command(runtime, args...)
	create.Dir = bundlePath
	stdin, stdout, stderr := getCreateStdio()
	create.Stdin, create.Stdout, create.Stderr = stdin, stdout, stderr
	if create.SysProcAttr == nil {
		create.SysProcAttr = &syscall.SysProcAttr{}
	}

	args = append(options.Args, "start", containerName)
	start := exec.Command(runtime, args...)
	start.Dir = bundlePath
	start.Stderr = os.Stderr

	args = append(options.Args, "kill", containerName)
	kill := exec.Command(runtime, args...)
	kill.Dir = bundlePath
	kill.Stderr = os.Stderr

	args = append(options.Args, "delete", containerName)
	del := exec.Command(runtime, args...)
	del.Dir = bundlePath
	del.Stderr = os.Stderr

	// Actually create the container.
	logrus.Debugf("Running %q", create.Args)
	err = create.Run()
	if err != nil {
		return 1, errors.Wrapf(err, "error creating container for %v: %s", spec.Process.Args, runCollectOutput(errorFds, closeBeforeReadingErrorFds))
	}
	defer func() {
		err2 := del.Run()
		if err2 != nil {
			if err == nil {
				err = errors.Wrapf(err2, "error deleting container")
			} else {
				logrus.Infof("error deleting container: %v", err2)
			}
		}
	}()

	// Make sure we read the container's exit status when it exits.
	pidValue, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 1, errors.Wrapf(err, "error reading pid from %q", pidFile)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidValue)))
	if err != nil {
		return 1, errors.Wrapf(err, "error parsing pid %s as a number", string(pidValue))
	}
	var reaping sync.WaitGroup
	reaping.Add(1)
	go func() {
		defer reaping.Done()
		var err error
		_, err = unix.Wait4(pid, &wstatus, 0, nil)
		if err != nil {
			wstatus = 0
			logrus.Errorf("error waiting for container child process %d: %v\n", pid, err)
		}
	}()

	if configureNetwork {
		teardown, err := runConfigureNetwork(options, configureNetworks, pid, containerName, spec.Process.Args)
		if teardown != nil {
			defer teardown()
		}
		if err != nil {
			return 1, err
		}
	}

	if copyPipes {
		// We don't need the ends of the pipes that belong to the container.
		stdin.Close()
		if stdout != nil {
			stdout.Close()
		}
		stderr.Close()
	}

	// Handle stdio for the container in the background.
	stdio.Add(1)
	go runCopyStdio(&stdio, copyPipes, stdioPipe, copyConsole, consoleListener, finishCopy, finishedCopy, spec)

	// Start the container.
	logrus.Debugf("Running %q", start.Args)
	err = start.Run()
	if err != nil {
		return 1, errors.Wrapf(err, "error starting container")
	}
	stopped := false
	defer func() {
		if !stopped {
			err2 := kill.Run()
			if err2 != nil {
				if err == nil {
					err = errors.Wrapf(err2, "error stopping container")
				} else {
					logrus.Infof("error stopping container: %v", err2)
				}
			}
		}
	}()

	// Wait for the container to exit.
	for {
		now := time.Now()
		var state specs.State
		args = append(options.Args, "state", containerName)
		stat := exec.Command(runtime, args...)
		stat.Dir = bundlePath
		stat.Stderr = os.Stderr
		stateOutput, stateErr := stat.Output()
		if stateErr != nil {
			return 1, errors.Wrapf(stateErr, "error reading container state")
		}
		if err = json.Unmarshal(stateOutput, &state); err != nil {
			return 1, errors.Wrapf(stateErr, "error parsing container state %q", string(stateOutput))
		}
		switch state.Status {
		case "running":
		case "stopped":
			stopped = true
		default:
			return 1, errors.Errorf("container status unexpectedly changed to %q", state.Status)
		}
		if stopped {
			break
		}
		select {
		case <-finishedCopy:
			stopped = true
		case <-time.After(time.Until(now.Add(100 * time.Millisecond))):
			continue
		}
		if stopped {
			break
		}
	}

	// Close the writing end of the stop-handling-stdio notification pipe.
	unix.Close(finishCopy[1])
	// Wait for the stdio copy goroutine to flush.
	stdio.Wait()
	// Wait until we finish reading the exit status.
	reaping.Wait()

	return wstatus, nil
}

func runCollectOutput(fds, closeBeforeReadingFds []int) string {
	for _, fd := range closeBeforeReadingFds {
		unix.Close(fd)
	}
	var b bytes.Buffer
	buf := make([]byte, 8192)
	for _, fd := range fds {
		nread, err := unix.Read(fd, buf)
		if err != nil {
			if errno, isErrno := err.(syscall.Errno); isErrno {
				switch errno {
				default:
					logrus.Errorf("error reading from pipe %d: %v", fd, err)
				case syscall.EINTR, syscall.EAGAIN:
				}
			} else {
				logrus.Errorf("unable to wait for data from pipe %d: %v", fd, err)
			}
			continue
		}
		for nread > 0 {
			r := buf[:nread]
			if nwritten, err := b.Write(r); err != nil || nwritten != len(r) {
				if nwritten != len(r) {
					logrus.Errorf("error buffering data from pipe %d: %v", fd, err)
					break
				}
			}
			nread, err = unix.Read(fd, buf)
			if err != nil {
				if errno, isErrno := err.(syscall.Errno); isErrno {
					switch errno {
					default:
						logrus.Errorf("error reading from pipe %d: %v", fd, err)
					case syscall.EINTR, syscall.EAGAIN:
					}
				} else {
					logrus.Errorf("unable to wait for data from pipe %d: %v", fd, err)
				}
				break
			}
		}
	}
	return b.String()
}

func runConfigureNetwork(options RunOptions, configureNetworks []string, pid int, containerName string, command []string) (teardown func(), err error) {
	var netconf, undo []*libcni.NetworkConfigList
	// Scan for CNI configuration files.
	confdir := options.CNIConfigDir
	files, err := libcni.ConfFiles(confdir, []string{".conf"})
	if err != nil {
		return nil, errors.Wrapf(err, "error finding CNI networking configuration files named *.conf in directory %q", confdir)
	}
	lists, err := libcni.ConfFiles(confdir, []string{".conflist"})
	if err != nil {
		return nil, errors.Wrapf(err, "error finding CNI networking configuration list files named *.conflist in directory %q", confdir)
	}
	logrus.Debugf("CNI network configuration file list: %#v", append(files, lists...))
	// Read the CNI configuration files.
	for _, file := range files {
		nc, err := libcni.ConfFromFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "error loading networking configuration from file %q for %v", file, command)
		}
		if len(configureNetworks) > 0 && nc.Network != nil && (nc.Network.Name == "" || !util.StringInSlice(nc.Network.Name, configureNetworks)) {
			if nc.Network.Name == "" {
				logrus.Debugf("configuration in %q has no name, skipping it", file)
			} else {
				logrus.Debugf("configuration in %q has name %q, skipping it", file, nc.Network.Name)
			}
			continue
		}
		cl, err := libcni.ConfListFromConf(nc)
		if err != nil {
			return nil, errors.Wrapf(err, "error converting networking configuration from file %q for %v", file, command)
		}
		logrus.Debugf("using network configuration from %q", file)
		netconf = append(netconf, cl)
	}
	for _, list := range lists {
		cl, err := libcni.ConfListFromFile(list)
		if err != nil {
			return nil, errors.Wrapf(err, "error loading networking configuration list from file %q for %v", list, command)
		}
		if len(configureNetworks) > 0 && (cl.Name == "" || !util.StringInSlice(cl.Name, configureNetworks)) {
			if cl.Name == "" {
				logrus.Debugf("configuration list in %q has no name, skipping it", list)
			} else {
				logrus.Debugf("configuration list in %q has name %q, skipping it", list, cl.Name)
			}
			continue
		}
		logrus.Debugf("using network configuration list from %q", list)
		netconf = append(netconf, cl)
	}
	// Make sure we can access the container's network namespace,
	// even after it exits, to successfully tear down the
	// interfaces.  Ensure this by opening a handle to the network
	// namespace, and using our copy to both configure and
	// deconfigure it.
	netns := fmt.Sprintf("/proc/%d/ns/net", pid)
	netFD, err := unix.Open(netns, unix.O_RDONLY, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening network namespace for %v", command)
	}
	mynetns := fmt.Sprintf("/proc/%d/fd/%d", unix.Getpid(), netFD)
	// Build our search path for the plugins.
	pluginPaths := strings.Split(options.CNIPluginPath, string(os.PathListSeparator))
	cni := libcni.CNIConfig{Path: pluginPaths}
	// Configure the interfaces.
	rtconf := make(map[*libcni.NetworkConfigList]*libcni.RuntimeConf)
	teardown = func() {
		for _, nc := range undo {
			if err = cni.DelNetworkList(nc, rtconf[nc]); err != nil {
				logrus.Errorf("error cleaning up network %v for %v: %v", rtconf[nc].IfName, command, err)
			}
		}
		unix.Close(netFD)
	}
	for i, nc := range netconf {
		// Build the runtime config for use with this network configuration.
		rtconf[nc] = &libcni.RuntimeConf{
			ContainerID:    containerName,
			NetNS:          mynetns,
			IfName:         fmt.Sprintf("if%d", i),
			Args:           [][2]string{},
			CapabilityArgs: map[string]interface{}{},
		}
		// Bring it up.
		_, err := cni.AddNetworkList(nc, rtconf[nc])
		if err != nil {
			return teardown, errors.Wrapf(err, "error configuring network list %v for %v", rtconf[nc].IfName, command)
		}
		// Add it to the list of networks to take down when the container process exits.
		undo = append([]*libcni.NetworkConfigList{nc}, undo...)
	}
	return teardown, nil
}

func runCopyStdio(stdio *sync.WaitGroup, copyPipes bool, stdioPipe [][]int, copyConsole bool, consoleListener *net.UnixListener, finishCopy []int, finishedCopy chan struct{}, spec *specs.Spec) {
	defer func() {
		unix.Close(finishCopy[0])
		if copyPipes {
			unix.Close(stdioPipe[unix.Stdin][1])
			unix.Close(stdioPipe[unix.Stdout][0])
			unix.Close(stdioPipe[unix.Stderr][0])
		}
		stdio.Done()
		finishedCopy <- struct{}{}
	}()
	// Map describing where data on an incoming descriptor should go.
	relayMap := make(map[int]int)
	// Map describing incoming and outgoing descriptors.
	readDesc := make(map[int]string)
	writeDesc := make(map[int]string)
	// Buffers.
	relayBuffer := make(map[int]*bytes.Buffer)
	// Set up the terminal descriptor or pipes for polling.
	if copyConsole {
		// Accept a connection over our listening socket.
		fd, err := runAcceptTerminal(consoleListener, spec.Process.ConsoleSize)
		if err != nil {
			logrus.Errorf("%v", err)
			return
		}
		terminalFD := fd
		// Input from our stdin, output from the terminal descriptor.
		relayMap[unix.Stdin] = terminalFD
		readDesc[unix.Stdin] = "stdin"
		relayBuffer[terminalFD] = new(bytes.Buffer)
		writeDesc[terminalFD] = "container terminal input"
		relayMap[terminalFD] = unix.Stdout
		readDesc[terminalFD] = "container terminal output"
		relayBuffer[unix.Stdout] = new(bytes.Buffer)
		writeDesc[unix.Stdout] = "output"
		// Set our terminal's mode to raw, to pass handling of special
		// terminal input to the terminal in the container.
		if terminal.IsTerminal(unix.Stdin) {
			if state, err := terminal.MakeRaw(unix.Stdin); err != nil {
				logrus.Warnf("error setting terminal state: %v", err)
			} else {
				defer func() {
					if err = terminal.Restore(unix.Stdin, state); err != nil {
						logrus.Errorf("unable to restore terminal state: %v", err)
					}
				}()
			}
		}
	}
	if copyPipes {
		// Input from our stdin, output from the stdout and stderr pipes.
		relayMap[unix.Stdin] = stdioPipe[unix.Stdin][1]
		readDesc[unix.Stdin] = "stdin"
		relayBuffer[stdioPipe[unix.Stdin][1]] = new(bytes.Buffer)
		writeDesc[stdioPipe[unix.Stdin][1]] = "container stdin"
		relayMap[stdioPipe[unix.Stdout][0]] = unix.Stdout
		readDesc[stdioPipe[unix.Stdout][0]] = "container stdout"
		relayBuffer[unix.Stdout] = new(bytes.Buffer)
		writeDesc[unix.Stdout] = "stdout"
		relayMap[stdioPipe[unix.Stderr][0]] = unix.Stderr
		readDesc[stdioPipe[unix.Stderr][0]] = "container stderr"
		relayBuffer[unix.Stderr] = new(bytes.Buffer)
		writeDesc[unix.Stderr] = "stderr"
	}
	// Set our reading descriptors to non-blocking.
	for rfd, wfd := range relayMap {
		if err := unix.SetNonblock(rfd, true); err != nil {
			logrus.Errorf("error setting %s to nonblocking: %v", readDesc[rfd], err)
			return
		}
		if err := unix.SetNonblock(wfd, false); err != nil {
			logrus.Errorf("error setting descriptor %d (%s) blocking: %v", wfd, writeDesc[wfd], err)
		}
	}
	// A helper that returns false if err is an error that would cause us
	// to give up.
	logIfNotRetryable := func(err error, what string) (retry bool) {
		if err == nil {
			return true
		}
		if errno, isErrno := err.(syscall.Errno); isErrno {
			switch errno {
			case syscall.EINTR, syscall.EAGAIN:
				return true
			}
		}
		logrus.Errorf("%s: %v", what, err)
		return false
	}
	// Pass data back and forth.
	pollTimeout := -1
	for len(relayMap) > 0 {
		// Start building the list of descriptors to poll.
		pollFds := make([]unix.PollFd, 0, len(relayMap)+1)
		// Poll for a notification that we should stop handling stdio.
		pollFds = append(pollFds, unix.PollFd{Fd: int32(finishCopy[0]), Events: unix.POLLIN | unix.POLLHUP})
		// Poll on our reading descriptors.
		for rfd := range relayMap {
			pollFds = append(pollFds, unix.PollFd{Fd: int32(rfd), Events: unix.POLLIN | unix.POLLHUP})
		}
		buf := make([]byte, 8192)
		// Wait for new data from any input descriptor, or a notification that we're done.
		_, err := unix.Poll(pollFds, pollTimeout)
		if !logIfNotRetryable(err, fmt.Sprintf("error waiting for stdio/terminal data to relay: %v", err)) {
			return
		}
		removes := make(map[int]struct{})
		for _, pollFd := range pollFds {
			// If this descriptor's just been closed from the other end, mark it for
			// removal from the set that we're checking for.
			if pollFd.Revents&unix.POLLHUP == unix.POLLHUP {
				removes[int(pollFd.Fd)] = struct{}{}
			}
			// If the descriptor was closed elsewhere, remove it from our list.
			if pollFd.Revents&unix.POLLNVAL != 0 {
				logrus.Debugf("error polling descriptor %s: closed?", readDesc[int(pollFd.Fd)])
				removes[int(pollFd.Fd)] = struct{}{}
			}
			// If the POLLIN flag isn't set, then there's no data to be read from this descriptor.
			if pollFd.Revents&unix.POLLIN == 0 {
				// If we're using pipes and it's our stdin and it's closed, close the writing
				// end of the corresponding pipe.
				if copyPipes && int(pollFd.Fd) == unix.Stdin && pollFd.Revents&unix.POLLHUP != 0 {
					unix.Close(stdioPipe[unix.Stdin][1])
					stdioPipe[unix.Stdin][1] = -1
				}
				continue
			}
			// Read whatever there is to be read.
			readFD := int(pollFd.Fd)
			writeFD, needToRelay := relayMap[readFD]
			if needToRelay {
				n, err := unix.Read(readFD, buf)
				if !logIfNotRetryable(err, fmt.Sprintf("unable to read %s data: %v", readDesc[readFD], err)) {
					return
				}
				// If it's zero-length on our stdin and we're
				// using pipes, it's an EOF, so close the stdin
				// pipe's writing end.
				if n == 0 && copyPipes && int(pollFd.Fd) == unix.Stdin {
					unix.Close(stdioPipe[unix.Stdin][1])
					stdioPipe[unix.Stdin][1] = -1
				}
				if n > 0 {
					// Buffer the data in case we get blocked on where they need to go.
					nwritten, err := relayBuffer[writeFD].Write(buf[:n])
					if err != nil {
						logrus.Debugf("buffer: %v", err)
						continue
					}
					if nwritten != n {
						logrus.Debugf("buffer: expected to buffer %d bytes, wrote %d", n, nwritten)
						continue
					}
					// If this is the last of the data we'll be able to read from this
					// descriptor, read all that there is to read.
					for pollFd.Revents&unix.POLLHUP == unix.POLLHUP {
						nr, err := unix.Read(readFD, buf)
						logIfNotRetryable(err, fmt.Sprintf("read %s: %v", readDesc[readFD], err))
						if nr <= 0 {
							break
						}
						nwritten, err := relayBuffer[writeFD].Write(buf[:nr])
						if err != nil {
							logrus.Debugf("buffer: %v", err)
							break
						}
						if nwritten != nr {
							logrus.Debugf("buffer: expected to buffer %d bytes, wrote %d", nr, nwritten)
							break
						}
					}
				}
			}
		}
		// Try to drain the output buffers.  Set the default timeout
		// for the next poll() to 100ms if we still have data to write.
		pollTimeout = -1
		for writeFD := range relayBuffer {
			if relayBuffer[writeFD].Len() > 0 {
				n, err := unix.Write(writeFD, relayBuffer[writeFD].Bytes())
				if !logIfNotRetryable(err, fmt.Sprintf("unable to write %s data: %v", writeDesc[writeFD], err)) {
					return
				}
				if n > 0 {
					relayBuffer[writeFD].Next(n)
				}
			}
			if relayBuffer[writeFD].Len() > 0 {
				pollTimeout = 100
			}
		}
		// Remove any descriptors which we don't need to poll any more from the poll descriptor list.
		for remove := range removes {
			delete(relayMap, remove)
		}
		// If the we-can-return pipe had anything for us, we're done.
		for _, pollFd := range pollFds {
			if int(pollFd.Fd) == finishCopy[0] && pollFd.Revents != 0 {
				// The pipe is closed, indicating that we can stop now.
				return
			}
		}
	}
}

func runAcceptTerminal(consoleListener *net.UnixListener, terminalSize *specs.Box) (int, error) {
	defer consoleListener.Close()
	c, err := consoleListener.AcceptUnix()
	if err != nil {
		return -1, errors.Wrapf(err, "error accepting socket descriptor connection")
	}
	defer c.Close()
	// Expect a control message over our new connection.
	b := make([]byte, 8192)
	oob := make([]byte, 8192)
	n, oobn, _, _, err := c.ReadMsgUnix(b, oob)
	if err != nil {
		return -1, errors.Wrapf(err, "error reading socket descriptor")
	}
	if n > 0 {
		logrus.Debugf("socket descriptor is for %q", string(b[:n]))
	}
	if oobn > len(oob) {
		return -1, errors.Errorf("too much out-of-bounds data (%d bytes)", oobn)
	}
	// Parse the control message.
	scm, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return -1, errors.Wrapf(err, "error parsing out-of-bound data as a socket control message")
	}
	logrus.Debugf("control messages: %v", scm)
	// Expect to get a descriptor.
	terminalFD := -1
	for i := range scm {
		fds, err := unix.ParseUnixRights(&scm[i])
		if err != nil {
			return -1, errors.Wrapf(err, "error parsing unix rights control message: %v", &scm[i])
		}
		logrus.Debugf("fds: %v", fds)
		if len(fds) == 0 {
			continue
		}
		terminalFD = fds[0]
		break
	}
	if terminalFD == -1 {
		return -1, errors.Errorf("unable to read terminal descriptor")
	}
	// Set the pseudoterminal's size to the configured size, or our own.
	winsize := &unix.Winsize{}
	if terminalSize != nil {
		// Use configured sizes.
		winsize.Row = uint16(terminalSize.Height)
		winsize.Col = uint16(terminalSize.Width)
	} else {
		if terminal.IsTerminal(unix.Stdin) {
			// Use the size of our terminal.
			if winsize, err = unix.IoctlGetWinsize(unix.Stdin, unix.TIOCGWINSZ); err != nil {
				logrus.Warnf("error reading size of controlling terminal: %v", err)
				winsize.Row = 0
				winsize.Col = 0
			}
		}
	}
	if winsize.Row != 0 && winsize.Col != 0 {
		if err = unix.IoctlSetWinsize(terminalFD, unix.TIOCSWINSZ, winsize); err != nil {
			logrus.Warnf("error setting size of container pseudoterminal: %v", err)
		}
		// FIXME - if we're connected to a terminal, we should
		// be passing the updated terminal size down when we
		// receive a SIGWINCH.
	}
	return terminalFD, nil
}

// Create pipes to use for relaying stdio.
func runMakeStdioPipe(uid, gid int) ([][]int, error) {
	stdioPipe := make([][]int, 3)
	for i := range stdioPipe {
		stdioPipe[i] = make([]int, 2)
		if err := unix.Pipe(stdioPipe[i]); err != nil {
			return nil, errors.Wrapf(err, "error creating pipe for container FD %d", i)
		}
	}
	if err := unix.Fchown(stdioPipe[unix.Stdin][0], uid, gid); err != nil {
		return nil, errors.Wrapf(err, "error setting owner of stdin pipe descriptor")
	}
	if err := unix.Fchown(stdioPipe[unix.Stdout][1], uid, gid); err != nil {
		return nil, errors.Wrapf(err, "error setting owner of stdout pipe descriptor")
	}
	if err := unix.Fchown(stdioPipe[unix.Stderr][1], uid, gid); err != nil {
		return nil, errors.Wrapf(err, "error setting owner of stderr pipe descriptor")
	}
	return stdioPipe, nil
}
