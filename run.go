package buildah

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
)

const (
	// DefaultWorkingDir is used if none was specified.
	DefaultWorkingDir = "/"
	// DefaultRuntime is the default command to use to run the container.
	DefaultRuntime = "runc"
)

const (
	// DefaultTerminal indicates that this Run invocation should be
	// connected to a pseudoterminal if we're connected to a terminal.
	DefaultTerminal = iota
	// WithoutTerminal indicates that this Run invocation should NOT be
	// connected to a pseudoterminal.
	WithoutTerminal
	// WithTerminal indicates that this Run invocation should be connected
	// to a pseudoterminal.
	WithTerminal
)

// RunOptions can be used to alter how a command is run in the container.
type RunOptions struct {
	// Hostname is the hostname we set for the running container.
	Hostname string
	// Runtime is the name of the command to run.  It should accept the same arguments that runc does.
	Runtime string
	// Args adds global arguments for the runtime.
	Args []string
	// Mounts are additional mount points which we want to provide.
	Mounts []specs.Mount
	// Env is additional environment variables to set.
	Env []string
	// User is the user as whom to run the command.
	User string
	// WorkingDir is an override for the working directory.
	WorkingDir string
	// Cmd is an override for the configured default command.
	Cmd []string
	// Entrypoint is an override for the configured entry point.
	Entrypoint []string
	// NetworkDisabled puts the container into its own network namespace.
	NetworkDisabled bool
	// Terminal provides a way to specify whether or not the command should
	// be run with a pseudoterminal.  By default (DefaultTerminal), a
	// terminal is used if os.Stdout is connected to a terminal, but that
	// decision can be overridden by specifying either WithTerminal or
	// WithoutTerminal.
	Terminal int
}

// Run runs the specified command in the container's root filesystem.
func (b *Builder) Run(command []string, options RunOptions) error {
	var user specs.User
	path, err := ioutil.TempDir(os.TempDir(), Package)
	if err != nil {
		return err
	}
	logrus.Debugf("using %q to hold bundle data", path)
	defer func() {
		if err2 := os.RemoveAll(path); err2 != nil {
			logrus.Errorf("error removing %q: %v", path, err2)
		}
	}()
	g := generate.New()

	if b.OS() != "" {
		g.SetPlatformOS(b.OS())
	}
	if b.Architecture() != "" {
		g.SetPlatformArch(b.Architecture())
	}
	for _, envSpec := range append(b.Env(), options.Env...) {
		env := strings.SplitN(envSpec, "=", 2)
		if len(env) > 1 {
			g.AddProcessEnv(env[0], env[1])
		}
	}
	if len(command) > 0 {
		g.SetProcessArgs(command)
	} else if len(options.Cmd) != 0 {
		g.SetProcessArgs(options.Cmd)
	} else if len(b.Cmd()) != 0 {
		g.SetProcessArgs(b.Cmd())
	} else if len(options.Entrypoint) != 0 {
		g.SetProcessArgs(options.Entrypoint)
	} else if len(b.Entrypoint()) != 0 {
		g.SetProcessArgs(b.Entrypoint())
	}
	if options.WorkingDir != "" {
		g.SetProcessCwd(options.WorkingDir)
	} else if b.WorkDir() != "" {
		g.SetProcessCwd(b.WorkDir())
	}
	if options.Hostname != "" {
		g.SetHostname(options.Hostname)
	}
	for _, volume := range b.Volumes() {
		g.AddTmpfsMount(volume, nil)
	}
	mountPoint, err := b.Mount("")
	if err != nil {
		return err
	}
	defer func() {
		if err2 := b.Unmount(); err2 != nil {
			logrus.Errorf("error unmounting container: %v", err2)
		}
	}()
	g.SetRootPath(mountPoint)
	switch options.Terminal {
	case DefaultTerminal:
		g.SetProcessTerminal(logrus.IsTerminal(os.Stdout))
	case WithTerminal:
		g.SetProcessTerminal(true)
	case WithoutTerminal:
		g.SetProcessTerminal(false)
	}
	if !options.NetworkDisabled {
		if err = g.RemoveLinuxNamespace("network"); err != nil {
			return fmt.Errorf("error removing network namespace for run: %v)", err)
		}
	}
	if options.User != "" {
		user, err = getUser(mountPoint, options.User)
	} else {
		user, err = getUser(mountPoint, b.User())
	}
	if err != nil {
		return err
	}
	g.SetProcessUID(user.UID)
	g.SetProcessGID(user.GID)
	spec := g.Spec()
	if spec.Process.Cwd == "" {
		spec.Process.Cwd = DefaultWorkingDir
	}
	if err = os.MkdirAll(filepath.Join(mountPoint, b.WorkDir()), 0755); err != nil {
		return fmt.Errorf("error ensuring working directory %q exists: %v)", b.WorkDir(), err)
	}
	mounts := options.Mounts
	boundMounts := []specs.Mount{}
	for _, boundFile := range []string{"/etc/hosts", "/etc/resolv.conf"} {
		for _, mount := range mounts {
			if mount.Destination == boundFile {
				// Already have an override for it, so skip this one.
				continue
			}
		}
		boundMount := specs.Mount{
			Source:      boundFile,
			Destination: boundFile,
			Type:        "bind",
			Options:     []string{"rbind", "ro"},
		}
		boundMounts = append(boundMounts, boundMount)
	}
	for _, specMount := range spec.Mounts {
		override := false
		for _, mount := range mounts {
			if specMount.Destination == mount.Destination {
				// Already have an override for it, so skip this one.
				override = true
			}
		}
		if !override {
			mounts = append(mounts, specMount)
		}
	}
	spec.Mounts = append(mounts, boundMounts...)
	specbytes, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	err = ioutils.AtomicWriteFile(filepath.Join(path, "config.json"), specbytes, 0600)
	if err != nil {
		return fmt.Errorf("error storing runtime configuration: %v", err)
	}
	logrus.Debugf("config = %v", string(specbytes))
	runtime := options.Runtime
	if runtime == "" {
		runtime = DefaultRuntime
	}
	args := append(options.Args, "run", "-b", path, Package+"-"+b.ContainerID)
	cmd := exec.Command(runtime, args...)
	cmd.Dir = mountPoint
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		logrus.Debugf("error running runc %v: %v", spec.Process.Args, err)
	}
	return err
}
