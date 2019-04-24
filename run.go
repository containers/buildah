package buildah

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/containers/buildah/bind"
	"github.com/containers/buildah/pkg/secrets"
	"github.com/containers/buildah/pkg/unshare"
	"github.com/containers/buildah/util"
	"github.com/containers/storage/pkg/reexec"
	"github.com/docker/go-units"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// runUsingRuntimeCommand is a command we use as a key for reexec
	runUsingRuntimeCommand = Package + "-oci-runtime"
)

var ErrNotSupported = errors.New("function not supported on non-linux systems")

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
	secretMounts := secrets.SecretMountsWithUIDGID(b.MountLabel, cdir, b.DefaultMountsFilePath, cdir, int(rootUID), int(rootGID), unshare.IsRootless())

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

func getDNSIP(dnsServers []string) (dns []net.IP, err error) {
	for _, i := range dnsServers {
		result := net.ParseIP(i)
		if result == nil {
			return dns, errors.Errorf("invalid IP address %s", i)
		}
		dns = append(dns, result)
	}
	return dns, nil
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

// Search for a command that isn't given as an absolute path using the $PATH
// under the rootfs.  We can't resolve absolute symbolic links without
// chroot()ing, which we may not be able to do, so just accept a link as a
// valid resolution.
func runLookupPath(g *generate.Generator, command []string) []string {
	// Look for the configured $PATH.
	spec := g.Config
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

func contains(volumes []string, v string) bool {
	for _, i := range volumes {
		if i == v {
			return true
		}
	}
	return false
}

func checkAndOverrideIsolationOptions(isolation Isolation, options *RunOptions) error {
	switch isolation {
	case IsolationOCIRootless:
		if ns := options.NamespaceOptions.Find(string(specs.IPCNamespace)); ns == nil || ns.Host {
			logrus.Debugf("Forcing use of an IPC namespace.")
		}
		options.NamespaceOptions.AddOrReplace(NamespaceOption{Name: string(specs.IPCNamespace)})
		_, err := exec.LookPath("slirp4netns")
		hostNetworking := err != nil
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
	Isolation         Isolation
}

func (b *Builder) runUsingRuntimeSubproc(isolation Isolation, options RunOptions, configureNetwork bool, configureNetworks, moreCreateArgs []string, spec *specs.Spec, rootPath, bundlePath, containerName string) (err error) {
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
		Isolation:         isolation,
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
