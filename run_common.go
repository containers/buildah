//go:build linux || freebsd
// +build linux freebsd

package buildah

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/containers/buildah/bind"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/util"
	"github.com/containers/common/libnetwork/etchosts"
	"github.com/containers/common/libnetwork/network"
	"github.com/containers/common/libnetwork/resolvconf"
	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// addResolvConf copies files from host and sets them up to bind mount into container
func (b *Builder) addResolvConf(rdir string, chownOpts *idtools.IDPair, dnsServers, dnsSearch, dnsOptions []string, namespaces []specs.LinuxNamespace) (string, error) {
	defaultConfig, err := config.Default()
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	nameservers := make([]string, 0, len(defaultConfig.Containers.DNSServers)+len(dnsServers))
	nameservers = append(nameservers, defaultConfig.Containers.DNSServers...)
	nameservers = append(nameservers, dnsServers...)

	keepHostServers := false
	// special check for slirp ip
	if len(nameservers) == 0 && b.Isolation == IsolationOCIRootless {
		for _, ns := range namespaces {
			if ns.Type == specs.NetworkNamespace && ns.Path == "" {
				keepHostServers = true
				// if we are using slirp4netns, also add the built-in DNS server.
				logrus.Debugf("adding slirp4netns 10.0.2.3 built-in DNS server")
				nameservers = append([]string{"10.0.2.3"}, nameservers...)
			}
		}
	}

	searches := make([]string, 0, len(defaultConfig.Containers.DNSSearches)+len(dnsSearch))
	searches = append(searches, defaultConfig.Containers.DNSSearches...)
	searches = append(searches, dnsSearch...)

	options := make([]string, 0, len(defaultConfig.Containers.DNSOptions)+len(dnsOptions))
	options = append(options, defaultConfig.Containers.DNSOptions...)
	options = append(options, dnsOptions...)

	cfile := filepath.Join(rdir, "resolv.conf")
	if err := resolvconf.New(&resolvconf.Params{
		Path:            cfile,
		Namespaces:      namespaces,
		IPv6Enabled:     true, // TODO we should check if we have ipv6
		KeepHostServers: keepHostServers,
		Nameservers:     nameservers,
		Searches:        searches,
		Options:         options,
	}); err != nil {
		return "", fmt.Errorf("error building resolv.conf for container %s: %w", b.ContainerID, err)
	}

	uid := 0
	gid := 0
	if chownOpts != nil {
		uid = chownOpts.UID
		gid = chownOpts.GID
	}
	if err = os.Chown(cfile, uid, gid); err != nil {
		return "", err
	}

	if err := label.Relabel(cfile, b.MountLabel, false); err != nil {
		return "", err
	}
	return cfile, nil
}

// generateHosts creates a containers hosts file
func (b *Builder) generateHosts(rdir string, chownOpts *idtools.IDPair, imageRoot string) (string, error) {
	conf, err := config.Default()
	if err != nil {
		return "", err
	}

	path, err := etchosts.GetBaseHostFile(conf.Containers.BaseHostsFile, imageRoot)
	if err != nil {
		return "", err
	}

	targetfile := filepath.Join(rdir, "hosts")
	if err := etchosts.New(&etchosts.Params{
		BaseFile:                 path,
		ExtraHosts:               b.CommonBuildOpts.AddHost,
		HostContainersInternalIP: etchosts.GetHostContainersInternalIP(conf, nil, nil),
		TargetFile:               targetfile,
	}); err != nil {
		return "", err
	}

	uid := 0
	gid := 0
	if chownOpts != nil {
		uid = chownOpts.UID
		gid = chownOpts.GID
	}
	if err = os.Chown(targetfile, uid, gid); err != nil {
		return "", err
	}
	if err := label.Relabel(targetfile, b.MountLabel, false); err != nil {
		return "", err
	}

	return targetfile, nil
}

// generateHostname creates a containers /etc/hostname file
func (b *Builder) generateHostname(rdir, hostname string, chownOpts *idtools.IDPair) (string, error) {
	var err error
	hostnamePath := "/etc/hostname"

	var hostnameBuffer bytes.Buffer
	hostnameBuffer.Write([]byte(fmt.Sprintf("%s\n", hostname)))

	cfile := filepath.Join(rdir, filepath.Base(hostnamePath))
	if err = ioutils.AtomicWriteFile(cfile, hostnameBuffer.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("error writing /etc/hostname into the container: %w", err)
	}

	uid := 0
	gid := 0
	if chownOpts != nil {
		uid = chownOpts.UID
		gid = chownOpts.GID
	}
	if err = os.Chown(cfile, uid, gid); err != nil {
		return "", err
	}
	if err := label.Relabel(cfile, b.MountLabel, false); err != nil {
		return "", err
	}

	return cfile, nil
}

func setupTerminal(g *generate.Generator, terminalPolicy TerminalPolicy, terminalSize *specs.Box) {
	switch terminalPolicy {
	case DefaultTerminal:
		onTerminal := term.IsTerminal(unix.Stdin) && term.IsTerminal(unix.Stdout) && term.IsTerminal(unix.Stderr)
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

func (b *Builder) configureUIDGID(g *generate.Generator, mountPoint string, options RunOptions) (string, error) {
	// Set the user UID/GID/supplemental group list/capabilities lists.
	user, homeDir, err := b.userForRun(mountPoint, options.User)
	if err != nil {
		return "", err
	}
	if err := setupCapabilities(g, b.Capabilities, options.AddCapabilities, options.DropCapabilities); err != nil {
		return "", err
	}
	g.SetProcessUID(user.UID)
	g.SetProcessGID(user.GID)
	for _, gid := range user.AdditionalGids {
		g.AddProcessAdditionalGid(gid)
	}

	// Remove capabilities if not running as root except Bounding set
	if user.UID != 0 && g.Config.Process.Capabilities != nil {
		bounding := g.Config.Process.Capabilities.Bounding
		g.ClearProcessCapabilities()
		g.Config.Process.Capabilities.Bounding = bounding
	}

	return homeDir, nil
}

func (b *Builder) configureEnvironment(g *generate.Generator, options RunOptions, defaultEnv []string) {
	g.ClearProcessEnv()

	if b.CommonBuildOpts.HTTPProxy {
		for _, envSpec := range config.ProxyEnv {
			if envVal, ok := os.LookupEnv(envSpec); ok {
				g.AddProcessEnv(envSpec, envVal)
			}
		}
	}

	for _, envSpec := range util.MergeEnv(util.MergeEnv(defaultEnv, b.Env()), options.Env) {
		env := strings.SplitN(envSpec, "=", 2)
		if len(env) > 1 {
			g.AddProcessEnv(env[0], env[1])
		}
	}
}

// getNetworkInterface creates the network interface
func getNetworkInterface(store storage.Store, cniConfDir, cniPluginPath string) (nettypes.ContainerNetwork, error) {
	conf, err := config.Default()
	if err != nil {
		return nil, err
	}
	// copy the config to not modify the default by accident
	newconf := *conf
	if len(cniConfDir) > 0 {
		newconf.Network.NetworkConfigDir = cniConfDir
	}
	if len(cniPluginPath) > 0 {
		plugins := strings.Split(cniPluginPath, string(os.PathListSeparator))
		newconf.Network.CNIPluginDirs = plugins
	}

	_, netInt, err := network.NetworkBackend(store, &newconf, false)
	if err != nil {
		return nil, err
	}
	return netInt, nil
}

// DefaultNamespaceOptions returns the default namespace settings from the
// runtime-tools generator library.
func DefaultNamespaceOptions() (define.NamespaceOptions, error) {
	cfg, err := config.Default()
	if err != nil {
		return nil, fmt.Errorf("failed to get container config: %w", err)
	}
	options := define.NamespaceOptions{
		{Name: string(specs.CgroupNamespace), Host: cfg.CgroupNS() == "host"},
		{Name: string(specs.IPCNamespace), Host: cfg.IPCNS() == "host"},
		{Name: string(specs.MountNamespace), Host: false},
		{Name: string(specs.NetworkNamespace), Host: cfg.NetNS() == "host"},
		{Name: string(specs.PIDNamespace), Host: cfg.PidNS() == "host"},
		{Name: string(specs.UserNamespace), Host: cfg.Containers.UserNS == "host"},
		{Name: string(specs.UTSNamespace), Host: cfg.UTSNS() == "host"},
	}
	return options, nil
}

func checkAndOverrideIsolationOptions(isolation define.Isolation, options *RunOptions) error {
	switch isolation {
	case IsolationOCIRootless:
		// only change the netns if the caller did not set it
		if ns := options.NamespaceOptions.Find(string(specs.NetworkNamespace)); ns == nil {
			if _, err := exec.LookPath("slirp4netns"); err != nil {
				// if slirp4netns is not installed we have to use the hosts net namespace
				options.NamespaceOptions.AddOrReplace(define.NamespaceOption{Name: string(specs.NetworkNamespace), Host: true})
			}
		}
		fallthrough
	case IsolationOCI:
		pidns := options.NamespaceOptions.Find(string(specs.PIDNamespace))
		userns := options.NamespaceOptions.Find(string(specs.UserNamespace))
		if (pidns != nil && pidns.Host) && (userns != nil && !userns.Host) {
			return fmt.Errorf("not allowed to mix host PID namespace with container user namespace")
		}
	}
	return nil
}

// fileCloser is a helper struct to prevent closing the file twice in the code
// users must call (fileCloser).Close() and not fileCloser.File.Close()
type fileCloser struct {
	file   *os.File
	closed bool
}

func (f *fileCloser) Close() {
	if !f.closed {
		if err := f.file.Close(); err != nil {
			logrus.Errorf("failed to close file: %v", err)
		}
		f.closed = true
	}
}

// waitForSync waits for a maximum of 4 minutes to read something from the file
func waitForSync(pipeR *os.File) error {
	if err := pipeR.SetDeadline(time.Now().Add(4 * time.Minute)); err != nil {
		return err
	}
	b := make([]byte, 16)
	_, err := pipeR.Read(b)
	return err
}

func contains(volumes []string, v string) bool {
	for _, i := range volumes {
		if i == v {
			return true
		}
	}
	return false
}

func runUsingRuntime(options RunOptions, configureNetwork bool, moreCreateArgs []string, spec *specs.Spec, bundlePath, containerName string,
	containerCreateW io.WriteCloser, containerStartR io.ReadCloser) (wstatus unix.WaitStatus, err error) {
	if options.Logger == nil {
		options.Logger = logrus.StandardLogger()
	}

	// Lock the caller to a single OS-level thread.
	runtime.LockOSThread()

	// Set up bind mounts for things that a namespaced user might not be able to get to directly.
	unmountAll, err := bind.SetupIntermediateMountNamespace(spec, bundlePath)
	if unmountAll != nil {
		defer func() {
			if err := unmountAll(); err != nil {
				options.Logger.Error(err)
			}
		}()
	}
	if err != nil {
		return 1, err
	}

	// Write the runtime configuration.
	specbytes, err := json.Marshal(spec)
	if err != nil {
		return 1, fmt.Errorf("error encoding configuration %#v as json: %w", spec, err)
	}
	if err = ioutils.AtomicWriteFile(filepath.Join(bundlePath, "config.json"), specbytes, 0600); err != nil {
		return 1, fmt.Errorf("error storing runtime configuration: %w", err)
	}

	logrus.Debugf("config = %v", string(specbytes))

	// Decide which runtime to use.
	runtime := options.Runtime
	if runtime == "" {
		runtime = util.Runtime()
	}
	localRuntime := util.FindLocalRuntime(runtime)
	if localRuntime != "" {
		runtime = localRuntime
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
		return 1, fmt.Errorf("error creating pipe for notifying to stop stdio: %w", err)
	}
	finishedCopy := make(chan struct{}, 1)
	var pargs []string
	if spec.Process != nil {
		pargs = spec.Process.Args
		if spec.Process.Terminal {
			copyConsole = true
			// Create a listening socket for accepting the container's terminal's PTY master.
			socketPath := filepath.Join(bundlePath, "console.sock")
			consoleListener, err = net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
			if err != nil {
				return 1, fmt.Errorf("error creating socket %q to receive terminal descriptor: %w", consoleListener.Addr(), err)
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
			if err = runLabelStdioPipes(stdioPipe, spec.Process.SelinuxLabel, spec.Linux.MountLabel); err != nil {
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

	runtimeArgs := options.Args[:]
	if options.CgroupManager == config.SystemdCgroupsManager {
		runtimeArgs = append(runtimeArgs, "--systemd-cgroup")
	}

	// Build the commands that we'll execute.
	pidFile := filepath.Join(bundlePath, "pid")
	args := append(append(append(runtimeArgs, "create", "--bundle", bundlePath, "--pid-file", pidFile), moreCreateArgs...), containerName)
	create := exec.Command(runtime, args...)
	setPdeathsig(create)
	create.Dir = bundlePath
	stdin, stdout, stderr := getCreateStdio()
	create.Stdin, create.Stdout, create.Stderr = stdin, stdout, stderr

	args = append(options.Args, "start", containerName)
	start := exec.Command(runtime, args...)
	setPdeathsig(start)
	start.Dir = bundlePath
	start.Stderr = os.Stderr

	kill := func(signal string) *exec.Cmd {
		args := append(options.Args, "kill", containerName)
		if signal != "" {
			args = append(args, signal)
		}
		kill := exec.Command(runtime, args...)
		kill.Dir = bundlePath
		kill.Stderr = os.Stderr
		return kill
	}

	args = append(options.Args, "delete", containerName)
	del := exec.Command(runtime, args...)
	del.Dir = bundlePath
	del.Stderr = os.Stderr

	// Actually create the container.
	logrus.Debugf("Running %q", create.Args)
	err = create.Run()
	if err != nil {
		return 1, fmt.Errorf("error from %s creating container for %v: %s: %w", runtime, pargs, runCollectOutput(options.Logger, errorFds, closeBeforeReadingErrorFds), err)
	}
	defer func() {
		err2 := del.Run()
		if err2 != nil {
			if err == nil {
				err = fmt.Errorf("error deleting container: %w", err2)
			} else {
				options.Logger.Infof("error from %s deleting container: %v", runtime, err2)
			}
		}
	}()

	// Make sure we read the container's exit status when it exits.
	pidValue, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 1, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidValue)))
	if err != nil {
		return 1, fmt.Errorf("error parsing pid %s as a number: %w", string(pidValue), err)
	}
	var stopped uint32
	var reaping sync.WaitGroup
	reaping.Add(1)
	go func() {
		defer reaping.Done()
		var err error
		_, err = unix.Wait4(pid, &wstatus, 0, nil)
		if err != nil {
			wstatus = 0
			options.Logger.Errorf("error waiting for container child process %d: %v\n", pid, err)
		}
		atomic.StoreUint32(&stopped, 1)
	}()

	if configureNetwork {
		if _, err := containerCreateW.Write([]byte{1}); err != nil {
			return 1, err
		}
		containerCreateW.Close()
		logrus.Debug("waiting for parent start message")
		b := make([]byte, 1)
		if _, err := containerStartR.Read(b); err != nil {
			return 1, fmt.Errorf("did not get container start message from parent: %w", err)
		}
		containerStartR.Close()
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
	go runCopyStdio(options.Logger, &stdio, copyPipes, stdioPipe, copyConsole, consoleListener, finishCopy, finishedCopy, spec)

	// Start the container.
	logrus.Debugf("Running %q", start.Args)
	err = start.Run()
	if err != nil {
		return 1, fmt.Errorf("error from %s starting container: %w", runtime, err)
	}
	defer func() {
		if atomic.LoadUint32(&stopped) == 0 {
			if err := kill("").Run(); err != nil {
				options.Logger.Infof("error from %s stopping container: %v", runtime, err)
			}
			atomic.StoreUint32(&stopped, 1)
		}
	}()

	// Wait for the container to exit.
	interrupted := make(chan os.Signal, 100)
	go func() {
		for range interrupted {
			if err := kill("SIGKILL").Run(); err != nil {
				logrus.Errorf("%v sending SIGKILL", err)
			}
		}
	}()
	signal.Notify(interrupted, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	for {
		now := time.Now()
		var state specs.State
		args = append(options.Args, "state", containerName)
		stat := exec.Command(runtime, args...)
		stat.Dir = bundlePath
		stat.Stderr = os.Stderr
		stateOutput, err := stat.Output()
		if err != nil {
			if atomic.LoadUint32(&stopped) != 0 {
				// container exited
				break
			}
			return 1, fmt.Errorf("error reading container state from %s (got output: %q): %w", runtime, string(stateOutput), err)
		}
		if err = json.Unmarshal(stateOutput, &state); err != nil {
			return 1, fmt.Errorf("error parsing container state %q from %s: %w", string(stateOutput), runtime, err)
		}
		switch state.Status {
		case "running":
		case "stopped":
			atomic.StoreUint32(&stopped, 1)
		default:
			return 1, fmt.Errorf("container status unexpectedly changed to %q", state.Status)
		}
		if atomic.LoadUint32(&stopped) != 0 {
			break
		}
		select {
		case <-finishedCopy:
			atomic.StoreUint32(&stopped, 1)
		case <-time.After(time.Until(now.Add(100 * time.Millisecond))):
			continue
		}
		if atomic.LoadUint32(&stopped) != 0 {
			break
		}
	}
	signal.Stop(interrupted)
	close(interrupted)

	// Close the writing end of the stop-handling-stdio notification pipe.
	unix.Close(finishCopy[1])
	// Wait for the stdio copy goroutine to flush.
	stdio.Wait()
	// Wait until we finish reading the exit status.
	reaping.Wait()

	return wstatus, nil
}

func runCollectOutput(logger *logrus.Logger, fds, closeBeforeReadingFds []int) string { //nolint:interfacer
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
					logger.Errorf("error reading from pipe %d: %v", fd, err)
				case syscall.EINTR, syscall.EAGAIN:
				}
			} else {
				logger.Errorf("unable to wait for data from pipe %d: %v", fd, err)
			}
			continue
		}
		for nread > 0 {
			r := buf[:nread]
			if nwritten, err := b.Write(r); err != nil || nwritten != len(r) {
				if nwritten != len(r) {
					logger.Errorf("error buffering data from pipe %d: %v", fd, err)
					break
				}
			}
			nread, err = unix.Read(fd, buf)
			if err != nil {
				if errno, isErrno := err.(syscall.Errno); isErrno {
					switch errno {
					default:
						logger.Errorf("error reading from pipe %d: %v", fd, err)
					case syscall.EINTR, syscall.EAGAIN:
					}
				} else {
					logger.Errorf("unable to wait for data from pipe %d: %v", fd, err)
				}
				break
			}
		}
	}
	return b.String()
}

func setNonblock(logger *logrus.Logger, fd int, description string, nonblocking bool) (bool, error) { //nolint:interfacer
	mask, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return false, err
	}
	blocked := mask&unix.O_NONBLOCK == 0

	if err := unix.SetNonblock(fd, nonblocking); err != nil {
		if nonblocking {
			logger.Errorf("error setting %s to nonblocking: %v", description, err)
		} else {
			logger.Errorf("error setting descriptor %s blocking: %v", description, err)
		}
	}
	return blocked, err
}

func runCopyStdio(logger *logrus.Logger, stdio *sync.WaitGroup, copyPipes bool, stdioPipe [][]int, copyConsole bool, consoleListener *net.UnixListener, finishCopy []int, finishedCopy chan struct{}, spec *specs.Spec) {
	defer func() {
		unix.Close(finishCopy[0])
		if copyPipes {
			unix.Close(stdioPipe[unix.Stdin][1])
			unix.Close(stdioPipe[unix.Stdout][0])
			unix.Close(stdioPipe[unix.Stderr][0])
		}
		stdio.Done()
		finishedCopy <- struct{}{}
		close(finishedCopy)
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
		fd, err := runAcceptTerminal(logger, consoleListener, spec.Process.ConsoleSize)
		if err != nil {
			logger.Errorf("%v", err)
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
		if term.IsTerminal(unix.Stdin) {
			if state, err := term.MakeRaw(unix.Stdin); err != nil {
				logger.Warnf("error setting terminal state: %v", err)
			} else {
				defer func() {
					if err = term.Restore(unix.Stdin, state); err != nil {
						logger.Errorf("unable to restore terminal state: %v", err)
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
		blocked, err := setNonblock(logger, rfd, readDesc[rfd], true)
		if err != nil {
			return
		}
		if blocked {
			defer setNonblock(logger, rfd, readDesc[rfd], false) // nolint:errcheck
		}
		setNonblock(logger, wfd, writeDesc[wfd], false) // nolint:errcheck
	}

	if copyPipes {
		setNonblock(logger, stdioPipe[unix.Stdin][1], writeDesc[stdioPipe[unix.Stdin][1]], true) // nolint:errcheck
	}

	runCopyStdioPassData(copyPipes, stdioPipe, finishCopy, relayMap, relayBuffer, readDesc, writeDesc)
}

func canRetry(err error) bool {
	if errno, isErrno := err.(syscall.Errno); isErrno {
		return errno == syscall.EINTR || errno == syscall.EAGAIN
	}
	return false
}

func runCopyStdioPassData(copyPipes bool, stdioPipe [][]int, finishCopy []int, relayMap map[int]int, relayBuffer map[int]*bytes.Buffer, readDesc map[int]string, writeDesc map[int]string) {
	closeStdin := false

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
		if !util.LogIfNotRetryable(err, fmt.Sprintf("error waiting for stdio/terminal data to relay: %v", err)) {
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
				continue
			}
			// Read whatever there is to be read.
			readFD := int(pollFd.Fd)
			writeFD, needToRelay := relayMap[readFD]
			if needToRelay {
				n, err := unix.Read(readFD, buf)
				if !util.LogIfNotRetryable(err, fmt.Sprintf("unable to read %s data: %v", readDesc[readFD], err)) {
					return
				}
				// If it's zero-length on our stdin and we're
				// using pipes, it's an EOF, so close the stdin
				// pipe's writing end.
				if n == 0 && !canRetry(err) && int(pollFd.Fd) == unix.Stdin {
					removes[int(pollFd.Fd)] = struct{}{}
				} else if n > 0 {
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
						util.LogIfUnexpectedWhileDraining(err, fmt.Sprintf("read %s: %v", readDesc[readFD], err))
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
				if !util.LogIfNotRetryable(err, fmt.Sprintf("unable to write %s data: %v", writeDesc[writeFD], err)) {
					return
				}
				if n > 0 {
					relayBuffer[writeFD].Next(n)
				}
				if closeStdin && writeFD == stdioPipe[unix.Stdin][1] && stdioPipe[unix.Stdin][1] >= 0 && relayBuffer[stdioPipe[unix.Stdin][1]].Len() == 0 {
					logrus.Debugf("closing stdin")
					unix.Close(stdioPipe[unix.Stdin][1])
					stdioPipe[unix.Stdin][1] = -1
				}
			}
			if relayBuffer[writeFD].Len() > 0 {
				pollTimeout = 100
			}
		}
		// Remove any descriptors which we don't need to poll any more from the poll descriptor list.
		for remove := range removes {
			if copyPipes && remove == unix.Stdin {
				closeStdin = true
				if relayBuffer[stdioPipe[unix.Stdin][1]].Len() == 0 {
					logrus.Debugf("closing stdin")
					unix.Close(stdioPipe[unix.Stdin][1])
					stdioPipe[unix.Stdin][1] = -1
				}
			}
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
