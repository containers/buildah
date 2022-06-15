//go:build freebsd
// +build freebsd

package buildah

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/containers/buildah/bind"
	"github.com/containers/buildah/chroot"
	"github.com/containers/buildah/copier"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/internal"
	internalParse "github.com/containers/buildah/internal/parse"
	internalUtil "github.com/containers/buildah/internal/util"
	"github.com/containers/buildah/pkg/jail"
	"github.com/containers/buildah/pkg/overlay"
	"github.com/containers/buildah/pkg/sshagent"
	"github.com/containers/buildah/util"
	"github.com/containers/common/libnetwork/resolvconf"
	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/config"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/lockfile"
	"github.com/containers/storage/pkg/stringid"
	storagetypes "github.com/containers/storage/types"
	"github.com/docker/go-units"
	"github.com/opencontainers/runtime-spec/specs-go"
	spec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	P_PID             = 0
	P_PGID            = 2
	PROC_REAP_ACQUIRE = 2
	PROC_REAP_RELEASE = 3
)

func procctl(idtype int, id int, cmd int, arg *byte) error {
	_, _, e1 := unix.Syscall6(
		unix.SYS_PROCCTL, uintptr(idtype), uintptr(id),
		uintptr(cmd), uintptr(unsafe.Pointer(arg)), 0, 0)
	if e1 != 0 {
		return unix.Errno(e1)
	}
	return nil
}

func setChildProcess() error {
	if err := procctl(P_PID, unix.Getpid(), PROC_REAP_ACQUIRE, nil); err != nil {
		fmt.Fprintf(os.Stderr, "procctl(PROC_REAP_ACQUIRE): %v\n", err)
		return err
	}
	return nil
}

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

	gp, err := generate.New("freebsd")
	if err != nil {
		return errors.Wrapf(err, "error generating new 'freebsd' runtime spec")
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

	// hardwire the environment to match docker build to avoid subtle and hard-to-debug differences due to containers.conf
	b.configureEnvironment(g, options, []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"})

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

	setupTerminal(g, options.Terminal, options.TerminalSize)

	configureNetwork, configureNetworks, err := b.configureNamespaces(g, &options)
	if err != nil {
		return err
	}

	containerName := Package + "-" + filepath.Base(path)
	if configureNetwork {
		g.AddAnnotation("org.freebsd.parentJail", containerName+"-vnet")
	}

	homeDir, err := b.configureUIDGID(g, mountPoint, options)
	if err != nil {
		return err
	}

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

	if !contains(volumes, resolvconf.DefaultResolvConf) && options.ConfigureNetwork != define.NetworkDisabled && !(len(b.CommonBuildOpts.DNSServers) == 1 && strings.ToLower(b.CommonBuildOpts.DNSServers[0]) == "none") {
		resolvFile, err := b.addResolvConf(path, rootIDPair, b.CommonBuildOpts.DNSServers, b.CommonBuildOpts.DNSSearch, b.CommonBuildOpts.DNSOptions, nil)
		if err != nil {
			return err
		}
		bindFiles[resolvconf.DefaultResolvConf] = resolvFile
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
		return errors.Wrapf(err, "error resolving mountpoints for container %q", b.ContainerID)
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

	// If we are creating a network, make the vnet here so that we
	// can execute the OCI runtime inside it.
	if configureNetwork {
		mynetns := containerName + "-vnet"

		jconf := jail.NewConfig()
		jconf.Set("name", mynetns)
		jconf.Set("vnet", jail.NEW)
		jconf.Set("children.max", 1)
		jconf.Set("persist", true)
		jconf.Set("enforce_statfs", 0)
		jconf.Set("devfs_ruleset", 4)
		jconf.Set("allow.raw_sockets", true)
		jconf.Set("allow.mount", true)
		jconf.Set("allow.mount.devfs", true)
		jconf.Set("allow.mount.nullfs", true)
		jconf.Set("allow.mount.fdescfs", true)
		jconf.Set("securelevel", -1)
		netjail, err := jail.Create(jconf)
		if err != nil {
			return err
		}
		defer func() {
			jconf := jail.NewConfig()
			jconf.Set("persist", false)
			err2 := netjail.Set(jconf)
			if err2 != nil {
				logrus.Errorf("error releasing vnet jail %q: %v", mynetns, err2)
			}
		}()
	}

	switch isolation {
	case IsolationOCI:
		var moreCreateArgs []string
		if options.NoPivot {
			moreCreateArgs = []string{"--no-pivot"}
		} else {
			moreCreateArgs = nil
		}
		err = b.runUsingRuntimeSubproc(isolation, options, configureNetwork, configureNetworks, moreCreateArgs, spec, mountPoint, path, containerName, b.Container, hostFile)
	case IsolationChroot:
		err = chroot.RunUsingChroot(spec, path, homeDir, options.Stdin, options.Stdout, options.Stderr)
	default:
		err = errors.Errorf("don't know how to run this command")
	}
	return err
}

func addCommonOptsToSpec(commonOpts *define.CommonBuildOptions, g *generate.Generator) error {
	defaultContainerConfig, err := config.Default()
	if err != nil {
		return errors.Wrapf(err, "failed to get container config")
	}
	// Other process resource limits
	if err := addRlimits(commonOpts.Ulimit, g, defaultContainerConfig.Containers.DefaultUlimits); err != nil {
		return err
	}

	logrus.Debugf("Resources: %#v", commonOpts)
	return nil
}

func (b *Builder) setupMounts(mountPoint string, spec *specs.Spec, bundlePath string, optionMounts []specs.Mount, bindFiles map[string]string, builtinVolumes, volumeMounts []string, runFileMounts []string, runMountInfo runMountInfo) (*runMountArtifacts, error) {
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

	specMounts := spec.Mounts

	// Get the list of files we need to bind into the container.
	bindFileMounts := runSetupBoundFiles(bundlePath, bindFiles)

	// After this point we need to know the per-container persistent storage directory.
	_, err := b.store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return nil, errors.Wrapf(err, "error determining work directory for container %q", b.ContainerID)
	}

	// Figure out which UID and GID to tell the subscriptions package to use
	// for files that it creates.
	rootUID, rootGID, err := util.GetHostRootIDs(spec)
	if err != nil {
		return nil, err
	}

	// Get host UID and GID of the container process.
	processUID, processGID, err := util.GetHostIDs(nil, nil, spec.Process.User.UID, spec.Process.User.GID)
	if err != nil {
		return nil, err
	}

	idMaps := IDMaps{
		rootUID:    int(rootUID),
		rootGID:    int(rootGID),
		processUID: int(processUID),
		processGID: int(processGID),
	}
	// Get the list of mounts that are just for this Run() call.
	runMounts, mountArtifacts, err := b.runSetupRunMounts(runFileMounts, runMountInfo, idMaps)
	if err != nil {
		return nil, err
	}

	// Get the list of explicitly-specified volume mounts.
	volumes, err := b.runSetupVolumeMounts(volumeMounts, optionMounts)
	if err != nil {
		return nil, err
	}

	// prepare list of mount destinations which can be cleaned up safely.
	// we can clean bindFiles, subscriptionMounts and specMounts
	// everything other than these might have users content
	mountArtifacts.RunMountTargets = append(mountArtifacts.RunMountTargets, cleanableDestinationListFromMounts(bindFileMounts)...)

	allMounts := append(volumes, specMounts...)
	allMounts = append(allMounts, runMounts...)
	allMounts = append(allMounts, bindFileMounts...)
	allMounts = util.SortMounts(allMounts)

	// Add them all, in the preferred order, except where they conflict with something that was previously added.
	for _, mount := range allMounts {
		if haveMount(mount.Destination) {
			// Already mounting something there, no need to bother with this one.
			continue
		}
		// Add the mount.
		mounts = append(mounts, mount)
	}

	// Set the list in the spec.
	spec.Mounts = mounts
	return mountArtifacts, nil
}

// Destinations which can be cleaned up after every RUN
func cleanableDestinationListFromMounts(mounts []spec.Mount) []string {
	mountDest := []string{}
	for _, mount := range mounts {
		// Add all destination to mountArtifacts so that they can be cleaned up later
		if mount.Destination != "" {
			// we dont want to remove destinations with  /etc, /dev as rootfs already contains these files
			// and unionfs will create a `whiteout` i.e `.wh` files on removal of overlapping files from these directories.
			// everything other than these will be cleanedup
			if !strings.HasPrefix(mount.Destination, "/etc") && !strings.HasPrefix(mount.Destination, "/dev") {
				mountDest = append(mountDest, mount.Destination)
			}
		}
	}
	return mountDest
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
		// For now, we only support type secret.
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
			/*
				case "cache":
					mount, lockedPaths, err := b.getCacheMount(tokens, rootUID, rootGID, processUID, processGID, stageMountPoints)
					if err != nil {
						return nil, nil, err
					}
					finalMounts = append(finalMounts, *mount)
					mountTargets = append(mountTargets, mount.Destination)
					lockedTargets = lockedPaths
			*/
		default:
			return nil, nil, errors.Errorf("invalid mount type %q", kv[1])
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
			return nil, "", errors.Errorf("secret required but no secret with id %s found", id)
		}
		return nil, "", nil
	}
	var data []byte
	var envFile string
	var ctrFileOnHost string

	switch secr.SourceType {
	case "env":
		data = []byte(os.Getenv(secr.Source))
		tmpFile, err := ioutil.TempFile("/var/tmp", "buildah*")
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
	hostUID, hostGID, err := util.GetHostIDs(nil, nil, uid, gid)
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
		Type:        "nullfs",
		Source:      ctrFileOnHost,
		Options:     []string{"ro"},
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
			return nil, nil, errors.Errorf("ssh required but no ssh with id %s found", id)
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

	hostUID, hostGID, err := util.GetHostIDs(nil, nil, uid, gid)
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
		Type:        "nullfs",
		Source:      hostSock,
		Options:     []string{"ro"},
	}
	return &newMount, fwdAgent, nil
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
	volumes, err := b.runSetupVolumeMounts(nil, optionMounts)
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
	volumes, err := b.runSetupVolumeMounts(nil, optionMounts)
	if err != nil {
		return nil, err
	}
	return &volumes[0], nil
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

func (b *Builder) runSetupVolumeMounts(volumeMounts []string, optionMounts []specs.Mount) (mounts []specs.Mount, Err error) {

	// Make sure the overlay directory is clean before running
	_, err := b.store.ContainerDirectory(b.ContainerID)
	if err != nil {
		return nil, errors.Wrapf(err, "error looking up container directory for %s", b.ContainerID)
	}

	parseMount := func(mountType, host, container string, options []string) (specs.Mount, error) {
		var foundrw, foundro bool
		for _, opt := range options {
			switch opt {
			case "rw":
				foundrw = true
			case "ro":
				foundro = true
			}
		}
		if !foundrw && !foundro {
			options = append(options, "rw")
		}
		if mountType == "bind" || mountType == "rbind" {
			mountType = "nullfs"
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
		spliti := strings.Split(i, ":")
		if len(spliti) > 2 {
			options = strings.Split(spliti[2], ",")
		}
		options = append(options, "bind")
		mount, err := parseMount("bind", spliti[0], spliti[1], options)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mount)
	}
	return mounts, nil
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
			if errors.Cause(err) == storagetypes.ErrImageUnknown {
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

func setupCapabilities(g *generate.Generator, defaultCapabilities, adds, drops []string) error {
	return nil
}

func (b *Builder) runConfigureNetwork(pid int, isolation define.Isolation, options RunOptions, configureNetworks []string, containerName string) (teardown func(), netStatus map[string]nettypes.StatusBlock, err error) {
	//if isolation == IsolationOCIRootless {
	//return setupRootlessNetwork(pid)
	//}

	if len(configureNetworks) == 0 {
		configureNetworks = []string{b.NetworkInterface.DefaultNetworkName()}
	}
	logrus.Debugf("configureNetworks: %v", configureNetworks)

	mynetns := containerName + "-vnet"

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
	_, err = b.NetworkInterface.Setup(mynetns, nettypes.SetupOptions{NetworkOptions: opts})
	if err != nil {
		return nil, nil, err
	}

	teardown = func() {
		err := b.NetworkInterface.Teardown(mynetns, nettypes.TeardownOptions{NetworkOptions: opts})
		if err != nil {
			logrus.Errorf("failed to cleanup network: %v", err)
		}
	}

	return teardown, nil, nil
}

func setupNamespaces(logger *logrus.Logger, g *generate.Generator, namespaceOptions define.NamespaceOptions, idmapOptions define.IDMappingOptions, policy define.NetworkConfigurationPolicy) (configureNetwork bool, configureNetworks []string, configureUTS bool, err error) {
	// Set namespace options in the container configuration.
	for _, namespaceOption := range namespaceOptions {
		switch namespaceOption.Name {
		case string(specs.NetworkNamespace):
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
		// TODO: re-visit this when there is consensus on a
		// FreeBSD runtime-spec. FreeBSD jails have rough
		// equivalents for UTS and and network namespaces.
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
		options := []string{}
		if strings.HasPrefix(src, bundlePath) {
			options = append(options, bind.NoBindOption)
		}
		mounts = append(mounts, specs.Mount{
			Source:      src,
			Destination: dest,
			Type:        "nullfs",
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
			return errors.Wrapf(err, "ulimit option %q requires name=SOFT:HARD, failed to be parsed", u)
		}

		g.AddProcessRlimits("RLIMIT_"+strings.ToUpper(ul.Name), uint64(ul.Hard), uint64(ul.Soft))
	}
	return nil
}

// setPdeathsig sets a parent-death signal for the process
func setPdeathsig(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
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
	return stdioPipe, nil
}
