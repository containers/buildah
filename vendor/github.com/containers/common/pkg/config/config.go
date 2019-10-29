package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/containers/common/pkg/unshare"
	units "github.com/docker/go-units"
	selinux "github.com/opencontainers/selinux/go-selinux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DefaultCgroupManager   = "systemd"
	DefaultApparmorProfile = "container-default"
	// DefaultPidsLimit is the default value for maximum number of processes
	// allowed inside a container
	DefaultPidsLimit = 2048
	// DefaultLogSizeMax is the default value for the maximum log size
	// allowed for a container. Negative values mean that no limit is imposed.
	DefaultLogSizeMax = -1
	OCIBufSize        = 8192
	// DefaultSeccompPath defines the default seccomp path
	DefaultSeccompPath = "/usr/share/containers/seccomp.json"
	// DefaultShmSize default value
	DefaultShmSize = "65536k"
	// DefaultContainersConfig holds the default containers config path
	DefaultContainersConfig = "/usr/share/containers/containers.conf"
	// OverrideContainersConfig holds the default config paths overridden by the root user
	OverrideContainersConfig = "/etc/containers/containers.conf"
)

// UserOverrideContainersConfig holds the containers config path overridden by the rootless user
var UserOverrideContainersConfig = filepath.Join(os.Getenv("HOME"), ".config/containers/containers.conf")

// tomlConfig is another way of looking at a Config, which is
// TOML-friendly (it has all of the explicit tables). It's just used for
// conversions.
type tomlConfig struct {
	Containers struct{ ContainersConfig } `toml:"containers"`
	Network    struct{ NetworkConfig }    `toml:"network"`
}

type Config struct {
	ContainersConfig
	NetworkConfig
}

// ContainersConfig represents the "containers" TOML config table
type ContainersConfig struct {
	// DefaultUlimits specifies the default ulimits to apply to containers
	DefaultUlimits []string `toml:"default_ulimits"`
	// Env is the environment variable list for container process.
	Env []string `toml:"env"`
	// HTTPProxy is the proxy environment variable list to apply to container process
	HTTPProxy []string `toml:"http_proxy"`
	// SELinux determines whether or not SELinux is used for pod separation.
	SELinux bool `toml:"selinux"`
	// SeccompProfile is the seccomp.json profile path which is used as the
	// default for the runtime.
	SeccompProfile string `toml:"seccomp_profile"`
	// ApparmorProfile is the apparmor profile name which is used as the
	// default for the runtime.
	ApparmorProfile string `toml:"apparmor_profile"`
	// CgroupManager is the manager implementation name which is used to
	// handle cgroups for containers. Supports cgroupfs and systemd.
	CgroupManager string `toml:"cgroup_manager"`
	// Capabilities to add to all containers.
	DefaultCapabilities []string `toml:"default_capabilities"`
	// Sysctls to add to all containers.
	DefaultSysctls []string `toml:"default_sysctls"`
	// PidsLimit is the number of processes each container is restricted to
	// by the cgroup process number controller.
	PidsLimit int64 `toml:"pids_limit"`
	// Devices to add to containers
	AdditionalDevices []string `toml:"additional_devices"`
	// LogSizeMax is the maximum number of bytes after which the log file
	// will be truncated. It can be expressed as a human-friendly string
	// that is parsed to bytes.
	// Negative values indicate that the log file won't be truncated.
	LogSizeMax int64 `toml:"log_size_max"`
	// HooksDir holds paths to the directories containing hooks
	// configuration files.  When the same filename is present in in
	// multiple directories, the file in the directory listed last in
	// this slice takes precedence.
	HooksDir []string `toml:"hooks_dir"`
	// ShmSize holds the size of /dev/shm.
	ShmSize string `toml:"shm_size"`
	// Run an init inside the container that forwards signals and reaps processes.
	Init bool `toml:"init"`
}

// NetworkConfig represents the "network" TOML config table
type NetworkConfig struct {
	// NetworkDir is where CNI network configuration files are stored.
	NetworkDir string `toml:"network_dir"`

	// PluginDir is where CNI plugin binaries are stored.
	PluginDir string `toml:"plugin_dir,omitempty"`

	// PluginDirs is where CNI plugin binaries are stored.
	PluginDirs []string `toml:"plugin_dirs"`
}

// DefaultCapabilities for the default_capabilities option in the containers.conf file
var DefaultCapabilities = []string{
	"CAP_AUDIT_WRITE",
	"CAP_CHOWN",
	"CAP_DAC_OVERRIDE",
	"CAP_FOWNER",
	"CAP_FSETID",
	"CAP_KILL",
	"CAP_MKNOD",
	"CAP_NET_BIND_SERVICE",
	"CAP_NET_RAW",
	"CAP_SETGID",
	"CAP_SETPCAP",
	"CAP_SETUID",
	"CAP_SYS_CHROOT",
}

// DefaultHooksDirs defines the default hooks directory
var DefaultHooksDirs = []string{"/usr/share/containers/oci/hooks.d"}

// New generates a Config from the containers.conf file path
func New(path string) (*Config, error) {
	defaultConfig := DefaultConfig()
	if path != "" {
		err := defaultConfig.UpdateFromFile(path)
		if err != nil {
			return nil, err
		}
	} else {
		err := defaultConfig.UpdateFromFile(DefaultContainersConfig)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		err = defaultConfig.UpdateFromFile(OverrideContainersConfig)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}

		if unshare.IsRootless() {
			err = defaultConfig.UpdateFromFile(UserOverrideContainersConfig)

			if err != nil && !os.IsNotExist(err) {
				return nil, err
			}
		}
	}
	if err := defaultConfig.Validate(true); err != nil {
		return nil, err
	}

	return defaultConfig, nil
}

// DefaultConfig defines the default values from containers.conf
func DefaultConfig() *Config {

	return &Config{
		ContainersConfig: ContainersConfig{
			AdditionalDevices: []string{},
			ApparmorProfile:   DefaultApparmorProfile,
			Env: []string{
				"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
			},
			HTTPProxy:           []string{},
			CgroupManager:       DefaultCgroupManager,
			DefaultCapabilities: DefaultCapabilities,
			DefaultSysctls:      []string{},
			DefaultUlimits:      []string{},
			LogSizeMax:          DefaultLogSizeMax,
			PidsLimit:           DefaultPidsLimit,
			SeccompProfile:      DefaultSeccompPath,
			SELinux:             selinuxEnabled(),
			ShmSize:             DefaultShmSize,
			HooksDir:            DefaultHooksDirs,
		},
		NetworkConfig: NetworkConfig{
			NetworkDir: cniConfigDir,
			PluginDirs: []string{cniBinDir},
		},
	}
}

func (t *tomlConfig) toConfig(c *Config) {
	c.ContainersConfig = t.Containers.ContainersConfig
	c.NetworkConfig = t.Network.NetworkConfig
}

func (t *tomlConfig) fromConfig(c *Config) {
	t.Containers.ContainersConfig = c.ContainersConfig
	t.Network.NetworkConfig = c.NetworkConfig
}

// UpdateFromFile populates the Config from the TOML-encoded file at the given path.
// Returns errors encountered when reading or parsing the files, or nil
// otherwise.
func (c *Config) UpdateFromFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	t := new(tomlConfig)
	t.fromConfig(c)

	_, err = toml.Decode(string(data), t)
	if err != nil {
		return fmt.Errorf("unable to decode configuration %v: %v", path, err)
	}

	t.toConfig(c)
	toCAPPrefixed := func(cap string) string {
		if !strings.HasPrefix(strings.ToLower(cap), "cap_") {
			return "CAP_" + strings.ToUpper(cap)
		}
		return cap
	}
	for i, cap := range c.ContainersConfig.DefaultCapabilities {
		c.ContainersConfig.DefaultCapabilities[i] = toCAPPrefixed(cap)
	}

	return nil
}

// Validate is the main entry point for library configuration validation.
// The parameter `onExecution` specifies if the validation should include
// execution checks. It returns an `error` on validation failure, otherwise
// `nil`.
func (c *Config) Validate(onExecution bool) error {

	if err := c.ContainersConfig.Validate(); err != nil {
		return errors.Wrapf(err, "containers config")
	}

	if !unshare.IsRootless() {
		if err := c.NetworkConfig.Validate(onExecution); err != nil {
			return errors.Wrapf(err, "network config")
		}
	}
	if !c.SELinux {
		selinux.SetDisabled()
	}

	return nil
}

// Validate is the main entry point for containers configuration validation
// It returns an `error` on validation failure, otherwise
// `nil`.
func (c *ContainersConfig) Validate() error {
	for _, u := range c.DefaultUlimits {
		ul, err := units.ParseUlimit(u)
		if err != nil {
			return fmt.Errorf("unrecognized ulimit %s: %v", u, err)
		}
		_, err = ul.GetRlimit()
		if err != nil {
			return err
		}
	}

	for _, d := range c.AdditionalDevices {
		_, _, _, err := Device(d)
		if err != nil {
			return err
		}
	}

	if c.LogSizeMax >= 0 && c.LogSizeMax < OCIBufSize {
		return fmt.Errorf("log size max should be negative or >= %d", OCIBufSize)
	}

	if _, err := units.FromHumanSize(c.ShmSize); err != nil {
		return fmt.Errorf("invalid --shm-size %s, %q", c.ShmSize, err)
	}

	return nil
}

// Validate is the main entry point for network configuration validation.
// The parameter `onExecution` specifies if the validation should include
// execution checks. It returns an `error` on validation failure, otherwise
// `nil`.
func (c *NetworkConfig) Validate(onExecution bool) error {
	if onExecution {
		err := IsDirectory(c.NetworkDir)
		if err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(c.NetworkDir, 0755); err != nil {
					return errors.Wrapf(err, "Cannot create network_dir: %s", c.NetworkDir)
				}
			} else {
				return errors.Wrapf(err, "invalid network_dir: %s", c.NetworkDir)
			}
		}

		for _, pluginDir := range c.PluginDirs {
			if err := os.MkdirAll(pluginDir, 0755); err != nil {
				return errors.Wrapf(err, "invalid plugin_dirs entry")
			}
		}
		// While the plugin_dir option is being deprecated, we need this check
		if c.PluginDir != "" {
			logrus.Warnf("The config field plugin_dir is being deprecated. Please use plugin_dirs instead")
			if err := os.MkdirAll(c.PluginDir, 0755); err != nil {
				return errors.Wrapf(err, "invalid plugin_dir entry")
			}
			// Append PluginDir to PluginDirs, so from now on we can operate in terms of PluginDirs and not worry
			// about missing cases.
			c.PluginDirs = append(c.PluginDirs, c.PluginDir)

			// Empty the pluginDir so on future config calls we don't print it out
			// thus seemlessly transitioning and depreciating the option
			c.PluginDir = ""
		}
	}

	return nil
}

// Device parses device mapping string to a src, dest & permissions string
// Valid values for device looklike:
//    '/dev/sdc"
//    '/dev/sdc:/dev/xvdc"
//    '/dev/sdc:/dev/xvdc:rwm"
//    '/dev/sdc:rm"
func Device(device string) (string, string, string, error) {
	src := ""
	dst := ""
	permissions := "rwm"
	split := strings.Split(device, ":")
	switch len(split) {
	case 3:
		if !IsValidDeviceMode(split[2]) {
			return "", "", "", fmt.Errorf("invalid device mode: %s", split[2])
		}
		permissions = split[2]
		fallthrough
	case 2:
		if IsValidDeviceMode(split[1]) {
			permissions = split[1]
		} else {
			if len(split[1]) == 0 || split[1][0] != '/' {
				return "", "", "", fmt.Errorf("invalid device mode: %s", split[1])
			}
			dst = split[1]
		}
		fallthrough
	case 1:
		if !strings.HasPrefix(split[0], "/dev/") {
			return "", "", "", fmt.Errorf("invalid device mode: %s", split[0])
		}
		src = split[0]
	default:
		return "", "", "", fmt.Errorf("invalid device specification: %s", device)
	}

	if dst == "" {
		dst = src
	}
	return src, dst, permissions, nil
}

// IsValidDeviceMode checks if the mode for device is valid or not.
// IsValid mode is a composition of r (read), w (write), and m (mknod).
func IsValidDeviceMode(mode string) bool {
	var legalDeviceMode = map[rune]bool{
		'r': true,
		'w': true,
		'm': true,
	}
	if mode == "" {
		return false
	}
	for _, c := range mode {
		if !legalDeviceMode[c] {
			return false
		}
		legalDeviceMode[c] = false
	}
	return true
}

// IsDirectory tests whether the given path exists and is a directory. It
// follows symlinks.
func IsDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.Mode().IsDir() {
		// Return a PathError to be consistent with os.Stat().
		return &os.PathError{
			Op:   "stat",
			Path: path,
			Err:  syscall.ENOTDIR,
		}
	}

	return nil
}
