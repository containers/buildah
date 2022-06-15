//go:build linux || freebsd
// +build linux freebsd

package buildah

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
