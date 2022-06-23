//go:build linux
// +build linux

package buildah

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/containers/buildah/bind"
	"github.com/containers/buildah/chroot"
	"github.com/containers/buildah/copier"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/internal"
	internalParse "github.com/containers/buildah/internal/parse"
	internalUtil "github.com/containers/buildah/internal/util"
	"github.com/containers/buildah/pkg/overlay"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/pkg/sshagent"
	"github.com/containers/buildah/util"
	"github.com/containers/common/libnetwork/resolvconf"
	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/capabilities"
	"github.com/containers/common/pkg/chown"
	"github.com/containers/common/pkg/config"
	"github.com/containers/common/pkg/hooks"
	hooksExec "github.com/containers/common/pkg/hooks/exec"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/lockfile"
	"github.com/containers/storage/pkg/stringid"
	"github.com/containers/storage/pkg/unshare"
	storagetypes "github.com/containers/storage/types"
	"github.com/docker/go-units"
	"github.com/opencontainers/runtime-spec/specs-go"
	spec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// ContainerDevices is an alias for a slice of github.com/opencontainers/runc/libcontainer/configs.Device structures.
type ContainerDevices define.ContainerDevices

func setChildProcess() error {
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(1), 0, 0, 0); err != nil {
		fmt.Fprintf(os.Stderr, "prctl(PR_SET_CHILD_SUBREAPER, 1): %v\n", err)
		return err
	}
	return nil
}

// Run runs the specified command in the container's root filesystem.
func (b *Builder) Run(command []string, options RunOptions) error {
	p, err := ioutil.TempDir("", define.Package)
	if err != nil {
		return err
	}
	// On some hosts like AH, /tmp is a symlink and we need an
	// absolute path.
	path, err := filepath.EvalSymlinks(p)
	if err != nil {
		return err
	}
	logrus.Debugf("using %q to hold bundle data", path)
	defer func() {
		if err2 := os.RemoveAll(path); err2 != nil {
			options.Logger.Error(err2)
		}
	}()

	gp, err := generate.New("linux")
	if err != nil {
		return fmt.Errorf("error generating new 'linux' runtime spec: %w", err)
	}
	g := &gp

	isolation := options.Isolation
	if isolation == define.IsolationDefault {
		isolation = b.Isolation
		if isolation == define.IsolationDefault {
			isolation = define.IsolationOCI
		}
	}
	if err := checkAndOverrideIsolationOptions(isolation, &options); err != nil {
		return err
	}

	// hardwire the environment to match docker build to avoid subtle and hard-to-debug differences due to containers.conf
	b.configureEnvironment(g, options, []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"})

	if b.CommonBuildOpts == nil {
		return fmt.Errorf("invalid format on container you must recreate the container")
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
		return fmt.Errorf("error mounting container %q: %w", b.ContainerID, err)
	}
	defer func() {
		if err := b.Unmount(); err != nil {
			options.Logger.Errorf("error unmounting container: %v", err)
		}
	}()
	g.SetRootPath(mountPoint)
	if len(command) > 0 {
		command = runLookupPath(g, command)
		g.SetProcessArgs(command)
	} else {
		g.SetProcessArgs(nil)
	}

	// Mount devices if any and if session is rootless attempt a bind-mount
	// just like podman.
	if unshare.IsRootless() {
		// We are going to create bind mounts for devices
		// but we need to make sure that we don't override
		// anything which is already in OCI spec.
		mounts := make(map[string]interface{})
		for _, m := range g.Mounts() {
			mounts[m.Destination] = true
		}
		newMounts := []spec.Mount{}
		for _, d := range b.Devices {
			// Default permission is read-only.
			perm := "ro"
			// Get permission configured for this device but only process `write`
			// permission in rootless since `mknod` is not supported anyways.
			if strings.Contains(string(d.Rule.Permissions), "w") {
				perm = "rw"
			}
			devMnt := spec.Mount{
				Destination: d.Destination,
				Type:        parse.TypeBind,
				Source:      d.Source,
				Options:     []string{"slave", "nosuid", "noexec", perm, "rbind"},
			}
			// Podman parity: podman skips these two devices hence we do the same.
			if d.Path == "/dev/ptmx" || strings.HasPrefix(d.Path, "/dev/tty") {
				continue
			}
			// Device is already in OCI spec do not re-mount.
			if _, found := mounts[d.Path]; found {
				continue
			}
			newMounts = append(newMounts, devMnt)
		}
		g.Config.Mounts = append(newMounts, g.Config.Mounts...)
	} else {
		for _, d := range b.Devices {
			sDev := spec.LinuxDevice{
				Type:     string(d.Type),
				Path:     d.Path,
				Major:    d.Major,
				Minor:    d.Minor,
				FileMode: &d.FileMode,
				UID:      &d.Uid,
				GID:      &d.Gid,
			}
			g.AddDevice(sDev)
			g.AddLinuxResourcesDevice(true, string(d.Type), &d.Major, &d.Minor, string(d.Permissions))
		}
	}

	setupMaskedPaths(g)
	setupReadOnlyPaths(g)

	setupTerminal(g, options.Terminal, options.TerminalSize)

	configureNetwork, configureNetworks, err := b.configureNamespaces(g, &options)
	if err != nil {
		return err
	}

	// rootless and networks are not supported
	if len(configureNetworks) > 0 && isolation == IsolationOCIRootless {
		return errors.New("cannot use networks as rootless")
	}

	homeDir, err := b.configureUIDGID(g, mountPoint, options)
	if err != nil {
		return err
	}

	g.SetProcessApparmorProfile(b.CommonBuildOpts.ApparmorProfile)

	// Now grab the spec from the generator.  Set the generator to nil so that future contributors
	// will quickly be able to tell that they're supposed to be modifying the spec directly from here.
	spec := g.Config
	g = nil

	// Set the seccomp configuration using the specified profile name.  Some syscalls are
	// allowed if certain capabilities are to be granted (example: CAP_SYS_CHROOT and chroot),
	// so we sorted out the capabilities lists first.
	if err = setupSeccomp(spec, b.CommonBuildOpts.SeccompProfilePath); err != nil {
		return err
	}

	uid, gid := spec.Process.User.UID, spec.Process.User.GID
	if spec.Linux != nil {
		uid, gid, err = util.GetHostIDs(spec.Linux.UIDMappings, spec.Linux.GIDMappings, uid, gid)
		if err != nil {
			return err
		}
	}

	idPair := &idtools.IDPair{UID: int(uid), GID: int(gid)}

	mode := os.FileMode(0755)
	coptions := copier.MkdirOptions{
		ChownNew: idPair,
		ChmodNew: &mode,
	}
	if err := copier.Mkdir(mountPoint, filepath.Join(mountPoint, spec.Process.Cwd), coptions); err != nil {
		return err
	}

	bindFiles := make(map[string]string)
	volumes := b.Volumes()

	// Figure out who owns files that will appear to be owned by UID/GID 0 in the container.
	rootUID, rootGID, err := util.GetHostRootIDs(spec)
	if err != nil {
		return err
	}
	rootIDPair := &idtools.IDPair{UID: int(rootUID), GID: int(rootGID)}

	hostFile := ""
	if !options.NoHosts && !contains(volumes, config.DefaultHostsFile) && options.ConfigureNetwork != define.NetworkDisabled {
		hostFile, err = b.generateHosts(path, rootIDPair, mountPoint)
		if err != nil {
			return err
		}
		bindFiles[config.DefaultHostsFile] = hostFile
	}

	// generate /etc/hostname if the user intentionally did not override
	if !(contains(volumes, "/etc/hostname")) {
		if _, ok := bindFiles["/etc/hostname"]; !ok {
			hostFile, err := b.generateHostname(path, spec.Hostname, rootIDPair)
			if err != nil {
				return err
			}
			// Bind /etc/hostname
			bindFiles["/etc/hostname"] = hostFile
		}
	}

	if !contains(volumes, resolvconf.DefaultResolvConf) && options.ConfigureNetwork != define.NetworkDisabled && !(len(b.CommonBuildOpts.DNSServers) == 1 && strings.ToLower(b.CommonBuildOpts.DNSServers[0]) == "none") {
		resolvFile, err := b.addResolvConf(path, rootIDPair, b.CommonBuildOpts.DNSServers, b.CommonBuildOpts.DNSSearch, b.CommonBuildOpts.DNSOptions, spec.Linux.Namespaces)
		if err != nil {
			return err
		}
		bindFiles[resolvconf.DefaultResolvConf] = resolvFile
	}
	// Empty file, so no need to recreate if it exists
	if _, ok := bindFiles["/run/.containerenv"]; !ok {
		containerenvPath := filepath.Join(path, "/run/.containerenv")
		if err = os.MkdirAll(filepath.Dir(containerenvPath), 0755); err != nil {
			return err
		}

		rootless := 0
		if unshare.IsRootless() {
			rootless = 1
		}
		// Populate the .containerenv with container information
		containerenv := fmt.Sprintf(`
engine="buildah-%s"
name=%q
id=%q
image=%q
imageid=%q
rootless=%d
`, define.Version, b.Container, b.ContainerID, b.FromImage, b.FromImageID, rootless)

		if err = ioutils.AtomicWriteFile(containerenvPath, []byte(containerenv), 0755); err != nil {
			return err
		}
		if err := label.Relabel(containerenvPath, b.MountLabel, false); err != nil {
			return err
		}

		bindFiles["/run/.containerenv"] = containerenvPath
	}

	// Setup OCI hooks
	_, err = b.setupOCIHooks(spec, (len(options.Mounts) > 0 || len(volumes) > 0))
	if err != nil {
		return fmt.Errorf("unable to setup OCI hooks: %w", err)
	}

	runMountInfo := runMountInfo{
		ContextDir:       options.ContextDir,
		Secrets:          options.Secrets,
		SSHSources:       options.SSHSources,
		StageMountPoints: options.StageMountPoints,
		SystemContext:    options.SystemContext,
	}

	runArtifacts, err := b.setupMounts(mountPoint, spec, path, options.Mounts, bindFiles, volumes, b.CommonBuildOpts.Volumes, options.RunMounts, runMountInfo)
	if err != nil {
		return fmt.Errorf("error resolving mountpoints for container %q: %w", b.ContainerID, err)
	}
	if runArtifacts.SSHAuthSock != "" {
		sshenv := "SSH_AUTH_SOCK=" + runArtifacts.SSHAuthSock
		spec.Process.Env = append(spec.Process.Env, sshenv)
	}

	// following run was called from `buildah run`
	// and some images were mounted for this run
	// add them to cleanup artifacts
	if len(options.ExternalImageMounts) > 0 {
		runArtifacts.MountedImages = append(runArtifacts.MountedImages, options.ExternalImageMounts...)
	}

	defer func() {
		if err := b.cleanupRunMounts(options.SystemContext, mountPoint, runArtifacts); err != nil {
			options.Logger.Errorf("unable to cleanup run mounts %v", err)
		}
	}()

	defer b.cleanupTempVolumes()

	switch isolation {
	case define.IsolationOCI:
		var moreCreateArgs []string
		if options.NoPivot {
			moreCreateArgs = append(moreCreateArgs, "--no-pivot")
		}
		err = b.runUsingRuntimeSubproc(isolation, options, configureNetwork, configureNetworks, moreCreateArgs, spec,
			mountPoint, path, define.Package+"-"+filepath.Base(path), b.Container, hostFile)
	case IsolationChroot:
		err = chroot.RunUsingChroot(spec, path, homeDir, options.Stdin, options.Stdout, options.Stderr)
	case IsolationOCIRootless:
		moreCreateArgs := []string{"--no-new-keyring"}
		if options.NoPivot {
			moreCreateArgs = append(moreCreateArgs, "--no-pivot")
		}
		err = b.runUsingRuntimeSubproc(isolation, options, configureNetwork, configureNetworks, moreCreateArgs, spec,
			mountPoint, path, define.Package+"-"+filepath.Base(path), b.Container, hostFile)
	default:
		err = errors.New("don't know how to run this command")
	}
	return err
}

func (b *Builder) setupOCIHooks(config *spec.Spec, hasVolumes bool) (map[string][]spec.Hook, error) {
	allHooks := make(map[string][]spec.Hook)
	if len(b.CommonBuildOpts.OCIHooksDir) == 0 {
		if unshare.IsRootless() {
			return nil, nil
		}
		for _, hDir := range []string{hooks.DefaultDir, hooks.OverrideDir} {
			manager, err := hooks.New(context.Background(), []string{hDir}, []string{})
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			ociHooks, err := manager.Hooks(config, b.ImageAnnotations, hasVolumes)
			if err != nil {
				return nil, err
			}
			if len(ociHooks) > 0 || config.Hooks != nil {
				logrus.Warnf("Implicit hook directories are deprecated; set --hooks-dir=%q explicitly to continue to load ociHooks from this directory", hDir)
			}
			for i, hook := range ociHooks {
				allHooks[i] = hook
			}
		}
	} else {
		manager, err := hooks.New(context.Background(), b.CommonBuildOpts.OCIHooksDir, []string{})
		if err != nil {
			return nil, err
		}

		allHooks, err = manager.Hooks(config, b.ImageAnnotations, hasVolumes)
		if err != nil {
			return nil, err
		}
	}

	hookErr, err := hooksExec.RuntimeConfigFilter(context.Background(), allHooks["precreate"], config, hooksExec.DefaultPostKillTimeout)
	if err != nil {
		logrus.Warnf("Container: precreate hook: %v", err)
		if hookErr != nil && hookErr != err {
			logrus.Debugf("container: precreate hook (hook error): %v", hookErr)
		}
		return nil, err
	}
	return allHooks, nil
}

func addCommonOptsToSpec(commonOpts *define.CommonBuildOptions, g *generate.Generator) error {
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

	defaultContainerConfig, err := config.Default()
	if err != nil {
		return fmt.Errorf("failed to get container config: %w", err)
	}
	// Other process resource limits
	if err := addRlimits(commonOpts.Ulimit, g, defaultContainerConfig.Containers.DefaultUlimits); err != nil {
		return err
	}

	logrus.Debugf("Resources: %#v", commonOpts)
	return nil
}

// Destinations which can be cleaned up after every RUN
func cleanableDestinationListFromMounts(mounts []spec.Mount) []string {
	mountDest := []string{}
	for _, mount := range mounts {
		// Add all destination to mountArtifacts so that they can be cleaned up later
		if mount.Destination != "" {
			// we dont want to remove destinations with  /etc, /dev, /sys, /proc as rootfs already contains these files
			// and unionfs will create a `whiteout` i.e `.wh` files on removal of overlapping files from these directories.
			// everything other than these will be cleanedup
			if !strings.HasPrefix(mount.Destination, "/etc") && !strings.HasPrefix(mount.Destination, "/dev") && !strings.HasPrefix(mount.Destination, "/sys") && !strings.HasPrefix(mount.Destination, "/proc") {
				mountDest = append(mountDest, mount.Destination)
			}
		}
	}
	return mountDest
}

func setupRootlessNetwork(pid int) (teardown func(), err error) {
	slirp4netns, err := exec.LookPath("slirp4netns")
	if err != nil {
		return nil, err
	}

	rootlessSlirpSyncR, rootlessSlirpSyncW, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("cannot create slirp4netns sync pipe: %w", err)
	}
	defer rootlessSlirpSyncR.Close()

	// Be sure there are no fds inherited to slirp4netns except the sync pipe
	files, err := ioutil.ReadDir("/proc/self/fd")
	if err != nil {
		return nil, fmt.Errorf("cannot list open fds: %w", err)
	}
	for _, f := range files {
		fd, err := strconv.Atoi(f.Name())
		if err != nil {
			return nil, fmt.Errorf("cannot parse fd: %w", err)
		}
		if fd == int(rootlessSlirpSyncW.Fd()) {
			continue
		}
		unix.CloseOnExec(fd)
	}

	cmd := exec.Command(slirp4netns, "--mtu", "65520", "-r", "3", "-c", strconv.Itoa(pid), "tap0")
	setPdeathsig(cmd)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	cmd.ExtraFiles = []*os.File{rootlessSlirpSyncW}

	err = cmd.Start()
	rootlessSlirpSyncW.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot start slirp4netns: %w", err)
	}

	b := make([]byte, 1)
	for {
		if err := rootlessSlirpSyncR.SetDeadline(time.Now().Add(1 * time.Second)); err != nil {
			return nil, fmt.Errorf("error setting slirp4netns pipe timeout: %w", err)
		}
		if _, err := rootlessSlirpSyncR.Read(b); err == nil {
			break
		} else {
			if os.IsTimeout(err) {
				// Check if the process is still running.
				var status syscall.WaitStatus
				_, err := syscall.Wait4(cmd.Process.Pid, &status, syscall.WNOHANG, nil)
				if err != nil {
					return nil, fmt.Errorf("failed to read slirp4netns process status: %w", err)
				}
				if status.Exited() || status.Signaled() {
					return nil, errors.New("slirp4netns failed")
				}

				continue
			}
			return nil, fmt.Errorf("failed to read from slirp4netns sync pipe: %w", err)
		}
	}

	return func() {
		cmd.Process.Kill() // nolint:errcheck
		cmd.Wait()         // nolint:errcheck
	}, nil
}

func (b *Builder) runConfigureNetwork(pid int, isolation define.Isolation, options RunOptions, configureNetworks []string, containerName string) (teardown func(), netStatus map[string]nettypes.StatusBlock, err error) {
	if isolation == IsolationOCIRootless {
		teardown, err = setupRootlessNetwork(pid)
		return teardown, nil, err
	}

	if len(configureNetworks) == 0 {
		configureNetworks = []string{b.NetworkInterface.DefaultNetworkName()}
	}

	// Make sure we can access the container's network namespace,
	// even after it exits, to successfully tear down the
	// interfaces.  Ensure this by opening a handle to the network
	// namespace, and using our copy to both configure and
	// deconfigure it.
	netns := fmt.Sprintf("/proc/%d/ns/net", pid)
	netFD, err := unix.Open(netns, unix.O_RDONLY, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening network namespace: %w", err)
	}
	mynetns := fmt.Sprintf("/proc/%d/fd/%d", unix.Getpid(), netFD)

	networks := make(map[string]nettypes.PerNetworkOptions, len(configureNetworks))
	for i, network := range configureNetworks {
		networks[network] = nettypes.PerNetworkOptions{
			InterfaceName: fmt.Sprintf("eth%d", i),
		}
	}

	opts := nettypes.NetworkOptions{
		ContainerID:   containerName,
		ContainerName: containerName,
		Networks:      networks,
	}
	netStatus, err = b.NetworkInterface.Setup(mynetns, nettypes.SetupOptions{NetworkOptions: opts})
	if err != nil {
		return nil, nil, err
	}

	teardown = func() {
		err := b.NetworkInterface.Teardown(mynetns, nettypes.TeardownOptions{NetworkOptions: opts})
		if err != nil {
			options.Logger.Errorf("failed to cleanup network: %v", err)
		}
	}

	return teardown, netStatus, nil
}

// Create pipes to use for relaying stdio.
func runMakeStdioPipe(uid, gid int) ([][]int, error) {
	stdioPipe := make([][]int, 3)
	for i := range stdioPipe {
		stdioPipe[i] = make([]int, 2)
		if err := unix.Pipe(stdioPipe[i]); err != nil {
			return nil, fmt.Errorf("error creating pipe for container FD %d: %w", i, err)
		}
	}
	if err := unix.Fchown(stdioPipe[unix.Stdin][0], uid, gid); err != nil {
		return nil, fmt.Errorf("error setting owner of stdin pipe descriptor: %w", err)
	}
	if err := unix.Fchown(stdioPipe[unix.Stdout][1], uid, gid); err != nil {
		return nil, fmt.Errorf("error setting owner of stdout pipe descriptor: %w", err)
	}
	if err := unix.Fchown(stdioPipe[unix.Stderr][1], uid, gid); err != nil {
		return nil, fmt.Errorf("error setting owner of stderr pipe descriptor: %w", err)
	}
	return stdioPipe, nil
}

func setupNamespaces(logger *logrus.Logger, g *generate.Generator, namespaceOptions define.NamespaceOptions, idmapOptions define.IDMappingOptions, policy define.NetworkConfigurationPolicy) (configureNetwork bool, configureNetworks []string, configureUTS bool, err error) {
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
				configureNetwork = (policy != define.NetworkDisabled)
			}
		case string(specs.UTSNamespace):
			configureUTS = false
			if !namespaceOption.Host && namespaceOption.Path == "" {
				configureUTS = true
			}
		}
		if namespaceOption.Host {
			if err := g.RemoveLinuxNamespace(namespaceOption.Name); err != nil {
				return false, nil, false, fmt.Errorf("error removing %q namespace for run: %w", namespaceOption.Name, err)
			}
		} else if err := g.AddOrReplaceLinuxNamespace(namespaceOption.Name, namespaceOption.Path); err != nil {
			if namespaceOption.Path == "" {
				return false, nil, false, fmt.Errorf("error adding new %q namespace for run: %w", namespaceOption.Name, err)
			}
			return false, nil, false, fmt.Errorf("error adding %q namespace %q for run: %w", namespaceOption.Name, namespaceOption.Path, err)
		}
	}

	// If we've got mappings, we're going to have to create a user namespace.
	if len(idmapOptions.UIDMap) > 0 || len(idmapOptions.GIDMap) > 0 || configureUserns {
		if err := g.AddOrReplaceLinuxNamespace(string(specs.UserNamespace), ""); err != nil {
			return false, nil, false, fmt.Errorf("error adding new %q namespace for run: %w", string(specs.UserNamespace), err)
		}
		hostUidmap, hostGidmap, err := unshare.GetHostIDMappings("")
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
			if err := g.AddOrReplaceLinuxNamespace(string(specs.NetworkNamespace), ""); err != nil {
				return false, nil, false, fmt.Errorf("error adding new %q namespace for run: %w", string(specs.NetworkNamespace), err)
			}
			configureNetwork = (policy != define.NetworkDisabled)
		}
	} else {
		if err := g.RemoveLinuxNamespace(string(specs.UserNamespace)); err != nil {
			return false, nil, false, fmt.Errorf("error removing %q namespace for run: %w", string(specs.UserNamespace), err)
		}
		if !specifiedNetwork {
			if err := g.RemoveLinuxNamespace(string(specs.NetworkNamespace)); err != nil {
				return false, nil, false, fmt.Errorf("error removing %q namespace for run: %w", string(specs.NetworkNamespace), err)
			}
		}
	}
	if configureNetwork && !unshare.IsRootless() {
		for name, val := range define.DefaultNetworkSysctl {
			// Check that the sysctl we are adding is actually supported
			// by the kernel
			p := filepath.Join("/proc/sys", strings.Replace(name, ".", "/", -1))
			_, err := os.Stat(p)
			if err != nil && !os.IsNotExist(err) {
				return false, nil, false, err
			}
			if err == nil {
				g.AddLinuxSysctl(name, val)
			} else {
				logger.Warnf("ignoring sysctl %s since %s doesn't exist", name, p)
			}
		}
	}
	return configureNetwork, configureNetworks, configureUTS, nil
}

func (b *Builder) configureNamespaces(g *generate.Generator, options *RunOptions) (bool, []string, error) {
	defaultNamespaceOptions, err := DefaultNamespaceOptions()
	if err != nil {
		return false, nil, err
	}

	namespaceOptions := defaultNamespaceOptions
	namespaceOptions.AddOrReplace(b.NamespaceOptions...)
	namespaceOptions.AddOrReplace(options.NamespaceOptions...)

	networkPolicy := options.ConfigureNetwork
	//Nothing was specified explicitly so network policy should be inherited from builder
	if networkPolicy == NetworkDefault {
		networkPolicy = b.ConfigureNetwork

		// If builder policy was NetworkDisabled and
		// we want to disable network for this run.
		// reset options.ConfigureNetwork to NetworkDisabled
		// since it will be treated as source of truth later.
		if networkPolicy == NetworkDisabled {
			options.ConfigureNetwork = networkPolicy
		}
	}

	configureNetwork, configureNetworks, configureUTS, err := setupNamespaces(options.Logger, g, namespaceOptions, b.IDMappingOptions, networkPolicy)
	if err != nil {
		return false, nil, err
	}

	if configureUTS {
		if options.Hostname != "" {
			g.SetHostname(options.Hostname)
		} else if b.Hostname() != "" {
			g.SetHostname(b.Hostname())
		} else {
			g.SetHostname(stringid.TruncateID(b.ContainerID))
		}
	} else {
		g.SetHostname("")
	}

	found := false
	spec := g.Config
	for i := range spec.Process.Env {
		if strings.HasPrefix(spec.Process.Env[i], "HOSTNAME=") {
			found = true
			break
		}
	}
	if !found {
		spec.Process.Env = append(spec.Process.Env, fmt.Sprintf("HOSTNAME=%s", spec.Hostname))
	}

	return configureNetwork, configureNetworks, nil
}

func runSetupBoundFiles(bundlePath string, bindFiles map[string]string) (mounts []specs.Mount) {
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
	return mounts
}

func addRlimits(ulimit []string, g *generate.Generator, defaultUlimits []string) error {
	var (
		ul  *units.Ulimit
		err error
	)

	ulimit = append(defaultUlimits, ulimit...)
	for _, u := range ulimit {
		if ul, err = units.ParseUlimit(u); err != nil {
			return fmt.Errorf("ulimit option %q requires name=SOFT:HARD, failed to be parsed: %w", u, err)
		}

		g.AddProcessRlimits("RLIMIT_"+strings.ToUpper(ul.Name), uint64(ul.Hard), uint64(ul.Soft))
	}
	return nil
}

func (b *Builder) cleanupTempVolumes() {
	for tempVolume, val := range b.TempVolumes {
		if val {
			if err := overlay.RemoveTemp(tempVolume); err != nil {
				b.Logger.Errorf(err.Error())
			}
			b.TempVolumes[tempVolume] = false
		}
	}
}

func (b *Builder) runSetupVolumeMounts(mountLabel string, volumeMounts []string, optionMounts []specs.Mount, idMaps IDMaps) (mounts []specs.Mount, Err error) {
	// Make sure the overlay directory is clean before running
	containerDir, err := b.store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return nil, fmt.Errorf("error looking up container directory for %s: %w", b.ContainerID, err)
	}
	if err := overlay.CleanupContent(containerDir); err != nil {
		return nil, fmt.Errorf("error cleaning up overlay content for %s: %w", b.ContainerID, err)
	}

	parseMount := func(mountType, host, container string, options []string) (specs.Mount, error) {
		var foundrw, foundro, foundz, foundZ, foundO, foundU bool
		var rootProp, upperDir, workDir string
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
			case "O":
				foundO = true
			case "U":
				foundU = true
			case "private", "rprivate", "slave", "rslave", "shared", "rshared":
				rootProp = opt
			}

			if strings.HasPrefix(opt, "upperdir") {
				splitOpt := strings.SplitN(opt, "=", 2)
				if len(splitOpt) > 1 {
					upperDir = splitOpt[1]
				}
			}
			if strings.HasPrefix(opt, "workdir") {
				splitOpt := strings.SplitN(opt, "=", 2)
				if len(splitOpt) > 1 {
					workDir = splitOpt[1]
				}
			}
		}
		if !foundrw && !foundro {
			options = append(options, "rw")
		}
		if foundz {
			if err := label.Relabel(host, mountLabel, true); err != nil {
				return specs.Mount{}, err
			}
		}
		if foundZ {
			if err := label.Relabel(host, mountLabel, false); err != nil {
				return specs.Mount{}, err
			}
		}
		if foundU {
			if err := chown.ChangeHostPathOwnership(host, true, idMaps.processUID, idMaps.processGID); err != nil {
				return specs.Mount{}, err
			}
		}
		if foundO {
			if (upperDir != "" && workDir == "") || (workDir != "" && upperDir == "") {
				return specs.Mount{}, errors.New("if specifying upperdir then workdir must be specified or vice versa")
			}

			containerDir, err := b.store.ContainerDirectory(b.ContainerID)
			if err != nil {
				return specs.Mount{}, err
			}

			contentDir, err := overlay.TempDir(containerDir, idMaps.rootUID, idMaps.rootGID)
			if err != nil {
				return specs.Mount{}, fmt.Errorf("failed to create TempDir in the %s directory: %w", containerDir, err)
			}

			overlayOpts := overlay.Options{
				RootUID:                idMaps.rootUID,
				RootGID:                idMaps.rootGID,
				UpperDirOptionFragment: upperDir,
				WorkDirOptionFragment:  workDir,
				GraphOpts:              b.store.GraphOptions(),
			}

			overlayMount, err := overlay.MountWithOptions(contentDir, host, container, &overlayOpts)
			if err == nil {
				b.TempVolumes[contentDir] = true
			}

			// If chown true, add correct ownership to the overlay temp directories.
			if foundU {
				if err := chown.ChangeHostPathOwnership(contentDir, true, idMaps.processUID, idMaps.processGID); err != nil {
					return specs.Mount{}, err
				}
			}

			return overlayMount, err
		}
		if rootProp == "" {
			options = append(options, "private")
		}
		if mountType != "tmpfs" {
			mountType = "bind"
			options = append(options, "rbind")
		}
		return specs.Mount{
			Destination: container,
			Type:        mountType,
			Source:      host,
			Options:     options,
		}, nil
	}

	// Bind mount volumes specified for this particular Run() invocation
	for _, i := range optionMounts {
		logrus.Debugf("setting up mounted volume at %q", i.Destination)
		mount, err := parseMount(i.Type, i.Source, i.Destination, i.Options)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}
	// Bind mount volumes given by the user when the container was created
	for _, i := range volumeMounts {
		var options []string
		spliti := parse.SplitStringWithColonEscape(i)
		if len(spliti) > 2 {
			options = strings.Split(spliti[2], ",")
		}
		options = append(options, "rbind")
		mount, err := parseMount("bind", spliti[0], spliti[1], options)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
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
		"/sys/fs/selinux",
		"/sys/dev",
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
			return fmt.Errorf("error adding %q to the bounding capability set: %w", cap, err)
		}
		if err := g.AddProcessCapabilityEffective(cap); err != nil {
			return fmt.Errorf("error adding %q to the effective capability set: %w", cap, err)
		}
		if err := g.AddProcessCapabilityPermitted(cap); err != nil {
			return fmt.Errorf("error adding %q to the permitted capability set: %w", cap, err)
		}
		if err := g.AddProcessCapabilityAmbient(cap); err != nil {
			return fmt.Errorf("error adding %q to the ambient capability set: %w", cap, err)
		}
	}
	return nil
}

func setupCapDrop(g *generate.Generator, caps ...string) error {
	for _, cap := range caps {
		if err := g.DropProcessCapabilityBounding(cap); err != nil {
			return fmt.Errorf("error removing %q from the bounding capability set: %w", cap, err)
		}
		if err := g.DropProcessCapabilityEffective(cap); err != nil {
			return fmt.Errorf("error removing %q from the effective capability set: %w", cap, err)
		}
		if err := g.DropProcessCapabilityPermitted(cap); err != nil {
			return fmt.Errorf("error removing %q from the permitted capability set: %w", cap, err)
		}
		if err := g.DropProcessCapabilityAmbient(cap); err != nil {
			return fmt.Errorf("error removing %q from the ambient capability set: %w", cap, err)
		}
	}
	return nil
}

func setupCapabilities(g *generate.Generator, defaultCapabilities, adds, drops []string) error {
	g.ClearProcessCapabilities()
	if err := setupCapAdd(g, defaultCapabilities...); err != nil {
		return err
	}
	for _, c := range adds {
		if strings.ToLower(c) == "all" {
			adds = capabilities.AllCapabilities()
			break
		}
	}
	for _, c := range drops {
		if strings.ToLower(c) == "all" {
			g.ClearProcessCapabilities()
			return nil
		}
	}
	if err := setupCapAdd(g, adds...); err != nil {
		return err
	}
	return setupCapDrop(g, drops...)
}

func addOrReplaceMount(mounts []specs.Mount, mount specs.Mount) []spec.Mount {
	for i := range mounts {
		if mounts[i].Destination == mount.Destination {
			mounts[i] = mount
			return mounts
		}
	}
	return append(mounts, mount)
}

// setupSpecialMountSpecChanges creates special mounts for depending on the namespaces
// logic taken from podman and adapted for buildah
// https://github.com/containers/podman/blob/4ba71f955a944790edda6e007e6d074009d437a7/pkg/specgen/generate/oci.go#L178
func setupSpecialMountSpecChanges(spec *spec.Spec, shmSize string) ([]specs.Mount, error) {
	mounts := spec.Mounts
	isRootless := unshare.IsRootless()
	isNewUserns := false
	isNetns := false
	isPidns := false
	isIpcns := false

	for _, namespace := range spec.Linux.Namespaces {
		switch namespace.Type {
		case specs.NetworkNamespace:
			isNetns = true
		case specs.UserNamespace:
			isNewUserns = true
		case specs.PIDNamespace:
			isPidns = true
		case specs.IPCNamespace:
			isIpcns = true
		}
	}

	addCgroup := true
	// mount sys when root and no userns or when both netns and userns are private
	canMountSys := (!isRootless && !isNewUserns) || (isNetns && isNewUserns)
	if !canMountSys {
		addCgroup = false
		sys := "/sys"
		sysMnt := specs.Mount{
			Destination: sys,
			Type:        "bind",
			Source:      sys,
			Options:     []string{bind.NoBindOption, "rprivate", "nosuid", "noexec", "nodev", "ro", "rbind"},
		}
		mounts = addOrReplaceMount(mounts, sysMnt)
	}

	gid5Available := true
	if isRootless {
		_, gids, err := unshare.GetHostIDMappings("")
		if err != nil {
			return nil, err
		}
		gid5Available = checkIdsGreaterThan5(gids)
	}
	if gid5Available && len(spec.Linux.GIDMappings) > 0 {
		gid5Available = checkIdsGreaterThan5(spec.Linux.GIDMappings)
	}
	if !gid5Available {
		// If we have no GID mappings, the gid=5 default option would fail, so drop it.
		devPts := specs.Mount{
			Destination: "/dev/pts",
			Type:        "devpts",
			Source:      "devpts",
			Options:     []string{"rprivate", "nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"},
		}
		mounts = addOrReplaceMount(mounts, devPts)
	}

	isUserns := isNewUserns || isRootless

	if isUserns && !isIpcns {
		devMqueue := "/dev/mqueue"
		devMqueueMnt := specs.Mount{
			Destination: devMqueue,
			Type:        "bind",
			Source:      devMqueue,
			Options:     []string{bind.NoBindOption, "bind", "nosuid", "noexec", "nodev"},
		}
		mounts = addOrReplaceMount(mounts, devMqueueMnt)
	}
	if isUserns && !isPidns {
		proc := "/proc"
		procMount := specs.Mount{
			Destination: proc,
			Type:        "bind",
			Source:      proc,
			Options:     []string{bind.NoBindOption, "rbind", "nosuid", "noexec", "nodev"},
		}
		mounts = addOrReplaceMount(mounts, procMount)
	}

	if addCgroup {
		cgroupMnt := specs.Mount{
			Destination: "/sys/fs/cgroup",
			Type:        "cgroup",
			Source:      "cgroup",
			Options:     []string{"rprivate", "nosuid", "noexec", "nodev", "relatime", "rw"},
		}
		mounts = addOrReplaceMount(mounts, cgroupMnt)
	}

	// if userns and host ipc bind mount shm
	if isUserns && !isIpcns {
		// bind mount /dev/shm when it exists
		if _, err := os.Stat("/dev/shm"); err == nil {
			shmMount := specs.Mount{
				Source:      "/dev/shm",
				Type:        "bind",
				Destination: "/dev/shm",
				Options:     []string{bind.NoBindOption, "rbind", "nosuid", "noexec", "nodev"},
			}
			mounts = addOrReplaceMount(mounts, shmMount)
		}
	} else if shmSize != "" {
		shmMount := specs.Mount{
			Source:      "shm",
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Options:     []string{"private", "nodev", "noexec", "nosuid", "mode=1777", "size=" + shmSize},
		}
		mounts = addOrReplaceMount(mounts, shmMount)
	}

	return mounts, nil
}

func checkIdsGreaterThan5(ids []spec.LinuxIDMapping) bool {
	for _, r := range ids {
		if r.ContainerID <= 5 && 5 < r.ContainerID+r.Size {
			return true
		}
	}
	return false
}

// runSetupRunMounts sets up mounts that exist only in this RUN, not in subsequent runs
func (b *Builder) runSetupRunMounts(mounts []string, sources runMountInfo, idMaps IDMaps) ([]spec.Mount, *runMountArtifacts, error) {
	mountTargets := make([]string, 0, 10)
	tmpFiles := make([]string, 0, len(mounts))
	mountImages := make([]string, 0, 10)
	finalMounts := make([]specs.Mount, 0, len(mounts))
	agents := make([]*sshagent.AgentServer, 0, len(mounts))
	sshCount := 0
	defaultSSHSock := ""
	tokens := []string{}
	lockedTargets := []string{}
	for _, mount := range mounts {
		arr := strings.SplitN(mount, ",", 2)

		kv := strings.Split(arr[0], "=")
		if len(kv) != 2 || kv[0] != "type" {
			return nil, nil, errors.New("invalid mount type")
		}
		if len(arr) == 2 {
			tokens = strings.Split(arr[1], ",")
		}

		switch kv[1] {
		case "secret":
			mount, envFile, err := b.getSecretMount(tokens, sources.Secrets, idMaps)
			if err != nil {
				return nil, nil, err
			}
			if mount != nil {
				finalMounts = append(finalMounts, *mount)
				mountTargets = append(mountTargets, mount.Destination)
				if envFile != "" {
					tmpFiles = append(tmpFiles, envFile)
				}
			}
		case "ssh":
			mount, agent, err := b.getSSHMount(tokens, sshCount, sources.SSHSources, idMaps)
			if err != nil {
				return nil, nil, err
			}
			if mount != nil {
				finalMounts = append(finalMounts, *mount)
				mountTargets = append(mountTargets, mount.Destination)
				agents = append(agents, agent)
				if sshCount == 0 {
					defaultSSHSock = mount.Destination
				}
				// Count is needed as the default destination of the ssh sock inside the container is  /run/buildkit/ssh_agent.{i}
				sshCount++
			}
		case "bind":
			mount, image, err := b.getBindMount(tokens, sources.SystemContext, sources.ContextDir, sources.StageMountPoints, idMaps)
			if err != nil {
				return nil, nil, err
			}
			finalMounts = append(finalMounts, *mount)
			mountTargets = append(mountTargets, mount.Destination)
			// only perform cleanup if image was mounted ignore everything else
			if image != "" {
				mountImages = append(mountImages, image)
			}
		case "tmpfs":
			mount, err := b.getTmpfsMount(tokens, idMaps)
			if err != nil {
				return nil, nil, err
			}
			finalMounts = append(finalMounts, *mount)
			mountTargets = append(mountTargets, mount.Destination)
		case "cache":
			mount, lockedPaths, err := b.getCacheMount(tokens, sources.StageMountPoints, idMaps)
			if err != nil {
				return nil, nil, err
			}
			finalMounts = append(finalMounts, *mount)
			mountTargets = append(mountTargets, mount.Destination)
			lockedTargets = lockedPaths
		default:
			return nil, nil, fmt.Errorf("invalid mount type %q", kv[1])
		}
	}
	artifacts := &runMountArtifacts{
		RunMountTargets: mountTargets,
		TmpFiles:        tmpFiles,
		Agents:          agents,
		MountedImages:   mountImages,
		SSHAuthSock:     defaultSSHSock,
		LockedTargets:   lockedTargets,
	}
	return finalMounts, artifacts, nil
}

func (b *Builder) getBindMount(tokens []string, context *imagetypes.SystemContext, contextDir string, stageMountPoints map[string]internal.StageMountDetails, idMaps IDMaps) (*spec.Mount, string, error) {
	if contextDir == "" {
		return nil, "", errors.New("Context Directory for current run invocation is not configured")
	}
	var optionMounts []specs.Mount
	mount, image, err := internalParse.GetBindMount(context, tokens, contextDir, b.store, b.MountLabel, stageMountPoints)
	if err != nil {
		return nil, image, err
	}
	optionMounts = append(optionMounts, mount)
	volumes, err := b.runSetupVolumeMounts(b.MountLabel, nil, optionMounts, idMaps)
	if err != nil {
		return nil, image, err
	}
	return &volumes[0], image, nil
}

func (b *Builder) getTmpfsMount(tokens []string, idMaps IDMaps) (*spec.Mount, error) {
	var optionMounts []specs.Mount
	mount, err := internalParse.GetTmpfsMount(tokens)
	if err != nil {
		return nil, err
	}
	optionMounts = append(optionMounts, mount)
	volumes, err := b.runSetupVolumeMounts(b.MountLabel, nil, optionMounts, idMaps)
	if err != nil {
		return nil, err
	}
	return &volumes[0], nil
}

func (b *Builder) getCacheMount(tokens []string, stageMountPoints map[string]internal.StageMountDetails, idMaps IDMaps) (*spec.Mount, []string, error) {
	var optionMounts []specs.Mount
	mount, lockedTargets, err := internalParse.GetCacheMount(tokens, b.store, b.MountLabel, stageMountPoints)
	if err != nil {
		return nil, lockedTargets, err
	}
	optionMounts = append(optionMounts, mount)
	volumes, err := b.runSetupVolumeMounts(b.MountLabel, nil, optionMounts, idMaps)
	if err != nil {
		return nil, lockedTargets, err
	}
	return &volumes[0], lockedTargets, nil
}

func (b *Builder) getSecretMount(tokens []string, secrets map[string]define.Secret, idMaps IDMaps) (*spec.Mount, string, error) {
	errInvalidSyntax := errors.New("secret should have syntax id=id[,target=path,required=bool,mode=uint,uid=uint,gid=uint")
	if len(tokens) == 0 {
		return nil, "", errInvalidSyntax
	}
	var err error
	var id, target string
	var required bool
	var uid, gid uint32
	var mode uint32 = 0400
	for _, val := range tokens {
		kv := strings.SplitN(val, "=", 2)
		switch kv[0] {
		case "id":
			id = kv[1]
		case "target", "dst", "destination":
			target = kv[1]
		case "required":
			required, err = strconv.ParseBool(kv[1])
			if err != nil {
				return nil, "", errInvalidSyntax
			}
		case "mode":
			mode64, err := strconv.ParseUint(kv[1], 8, 32)
			if err != nil {
				return nil, "", errInvalidSyntax
			}
			mode = uint32(mode64)
		case "uid":
			uid64, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return nil, "", errInvalidSyntax
			}
			uid = uint32(uid64)
		case "gid":
			gid64, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return nil, "", errInvalidSyntax
			}
			gid = uint32(gid64)
		default:
			return nil, "", errInvalidSyntax
		}
	}

	if id == "" {
		return nil, "", errInvalidSyntax
	}
	// Default location for secretis is /run/secrets/id
	if target == "" {
		target = "/run/secrets/" + id
	}

	secr, ok := secrets[id]
	if !ok {
		if required {
			return nil, "", fmt.Errorf("secret required but no secret with id %s found", id)
		}
		return nil, "", nil
	}
	var data []byte
	var envFile string
	var ctrFileOnHost string

	switch secr.SourceType {
	case "env":
		data = []byte(os.Getenv(secr.Source))
		tmpFile, err := ioutil.TempFile("/dev/shm", "buildah*")
		if err != nil {
			return nil, "", err
		}
		envFile = tmpFile.Name()
		ctrFileOnHost = tmpFile.Name()
	case "file":
		containerWorkingDir, err := b.store.ContainerDirectory(b.ContainerID)
		if err != nil {
			return nil, "", err
		}
		data, err = ioutil.ReadFile(secr.Source)
		if err != nil {
			return nil, "", err
		}
		ctrFileOnHost = filepath.Join(containerWorkingDir, "secrets", id)
	default:
		return nil, "", errors.New("invalid source secret type")
	}

	// Copy secrets to container working dir (or tmp dir if it's an env), since we need to chmod,
	// chown and relabel it for the container user and we don't want to mess with the original file
	if err := os.MkdirAll(filepath.Dir(ctrFileOnHost), 0755); err != nil {
		return nil, "", err
	}
	if err := ioutil.WriteFile(ctrFileOnHost, data, 0644); err != nil {
		return nil, "", err
	}

	if err := label.Relabel(ctrFileOnHost, b.MountLabel, false); err != nil {
		return nil, "", err
	}
	hostUID, hostGID, err := util.GetHostIDs(idMaps.uidmap, idMaps.gidmap, uid, gid)
	if err != nil {
		return nil, "", err
	}
	if err := os.Lchown(ctrFileOnHost, int(hostUID), int(hostGID)); err != nil {
		return nil, "", err
	}
	if err := os.Chmod(ctrFileOnHost, os.FileMode(mode)); err != nil {
		return nil, "", err
	}
	newMount := specs.Mount{
		Destination: target,
		Type:        "bind",
		Source:      ctrFileOnHost,
		Options:     []string{"bind", "rprivate", "ro"},
	}
	return &newMount, envFile, nil
}

// getSSHMount parses the --mount type=ssh flag in the Containerfile, checks if there's an ssh source provided, and creates and starts an ssh-agent to be forwarded into the container
func (b *Builder) getSSHMount(tokens []string, count int, sshsources map[string]*sshagent.Source, idMaps IDMaps) (*spec.Mount, *sshagent.AgentServer, error) {
	errInvalidSyntax := errors.New("ssh should have syntax id=id[,target=path,required=bool,mode=uint,uid=uint,gid=uint")

	var err error
	var id, target string
	var required bool
	var uid, gid uint32
	var mode uint32 = 400
	for _, val := range tokens {
		kv := strings.SplitN(val, "=", 2)
		if len(kv) < 2 {
			return nil, nil, errInvalidSyntax
		}
		switch kv[0] {
		case "id":
			id = kv[1]
		case "target", "dst", "destination":
			target = kv[1]
		case "required":
			required, err = strconv.ParseBool(kv[1])
			if err != nil {
				return nil, nil, errInvalidSyntax
			}
		case "mode":
			mode64, err := strconv.ParseUint(kv[1], 8, 32)
			if err != nil {
				return nil, nil, errInvalidSyntax
			}
			mode = uint32(mode64)
		case "uid":
			uid64, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return nil, nil, errInvalidSyntax
			}
			uid = uint32(uid64)
		case "gid":
			gid64, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				return nil, nil, errInvalidSyntax
			}
			gid = uint32(gid64)
		default:
			return nil, nil, errInvalidSyntax
		}
	}

	if id == "" {
		id = "default"
	}
	// Default location for secretis is /run/buildkit/ssh_agent.{i}
	if target == "" {
		target = fmt.Sprintf("/run/buildkit/ssh_agent.%d", count)
	}

	sshsource, ok := sshsources[id]
	if !ok {
		if required {
			return nil, nil, fmt.Errorf("ssh required but no ssh with id %s found", id)
		}
		return nil, nil, nil
	}
	// Create new agent from keys or socket
	fwdAgent, err := sshagent.NewAgentServer(sshsource)
	if err != nil {
		return nil, nil, err
	}
	// Start ssh server, and get the host sock we're mounting in the container
	hostSock, err := fwdAgent.Serve(b.ProcessLabel)
	if err != nil {
		return nil, nil, err
	}

	if err := label.Relabel(filepath.Dir(hostSock), b.MountLabel, false); err != nil {
		if shutdownErr := fwdAgent.Shutdown(); shutdownErr != nil {
			b.Logger.Errorf("error shutting down agent: %v", shutdownErr)
		}
		return nil, nil, err
	}
	if err := label.Relabel(hostSock, b.MountLabel, false); err != nil {
		if shutdownErr := fwdAgent.Shutdown(); shutdownErr != nil {
			b.Logger.Errorf("error shutting down agent: %v", shutdownErr)
		}
		return nil, nil, err
	}
	hostUID, hostGID, err := util.GetHostIDs(idMaps.uidmap, idMaps.gidmap, uid, gid)
	if err != nil {
		if shutdownErr := fwdAgent.Shutdown(); shutdownErr != nil {
			b.Logger.Errorf("error shutting down agent: %v", shutdownErr)
		}
		return nil, nil, err
	}
	if err := os.Lchown(hostSock, int(hostUID), int(hostGID)); err != nil {
		if shutdownErr := fwdAgent.Shutdown(); shutdownErr != nil {
			b.Logger.Errorf("error shutting down agent: %v", shutdownErr)
		}
		return nil, nil, err
	}
	if err := os.Chmod(hostSock, os.FileMode(mode)); err != nil {
		if shutdownErr := fwdAgent.Shutdown(); shutdownErr != nil {
			b.Logger.Errorf("error shutting down agent: %v", shutdownErr)
		}
		return nil, nil, err
	}
	newMount := specs.Mount{
		Destination: target,
		Type:        "bind",
		Source:      hostSock,
		Options:     []string{"bind", "rprivate", "ro"},
	}
	return &newMount, fwdAgent, nil
}

// cleanupRunMounts cleans up run mounts so they only appear in this run.
func (b *Builder) cleanupRunMounts(context *imagetypes.SystemContext, mountpoint string, artifacts *runMountArtifacts) error {
	for _, agent := range artifacts.Agents {
		err := agent.Shutdown()
		if err != nil {
			return err
		}
	}

	//cleanup any mounted images for this run
	for _, image := range artifacts.MountedImages {
		if image != "" {
			// if flow hits here some image was mounted for this run
			i, err := internalUtil.LookupImage(context, b.store, image)
			if err == nil {
				// silently try to unmount and do nothing
				// if image is being used by something else
				_ = i.Unmount(false)
			}
			if errors.Is(err, storagetypes.ErrImageUnknown) {
				// Ignore only if ErrImageUnknown
				// Reason: Image is already unmounted do nothing
				continue
			}
			return err
		}
	}

	opts := copier.RemoveOptions{
		All: true,
	}
	for _, path := range artifacts.RunMountTargets {
		err := copier.Remove(mountpoint, path, opts)
		if err != nil {
			return err
		}
	}
	var prevErr error
	for _, path := range artifacts.TmpFiles {
		err := os.Remove(path)
		if !os.IsNotExist(err) {
			if prevErr != nil {
				logrus.Error(prevErr)
			}
			prevErr = err
		}
	}
	// unlock if any locked files from this RUN statement
	for _, path := range artifacts.LockedTargets {
		_, err := os.Stat(path)
		if err != nil {
			// Lockfile not found this might be a problem,
			// since LockedTargets must contain list of all locked files
			// don't break here since we need to unlock other files but
			// log so user can take a look
			logrus.Warnf("Lockfile %q was expected here, stat failed with %v", path, err)
			continue
		}
		lockfile, err := lockfile.GetLockfile(path)
		if err != nil {
			// unable to get lockfile
			// lets log error and continue
			// unlocking other files
			logrus.Warn(err)
			continue
		}
		if lockfile.Locked() {
			lockfile.Unlock()
		} else {
			logrus.Warnf("Lockfile %q was expected to be locked, this is unexpected", path)
			continue
		}
	}
	return prevErr
}

// setPdeathsig sets a parent-death signal for the process
func setPdeathsig(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
