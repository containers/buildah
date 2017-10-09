package buildah

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containers/storage/pkg/ioutils"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
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

func (b *Builder) setupMounts(mountPoint string, spec *specs.Spec, optionMounts []specs.Mount, bindFiles, volumes []string) error {
	// The passed-in mounts matter the most to us.
	mounts := make([]specs.Mount, len(optionMounts))
	copy(mounts, optionMounts)
	haveMount := func(destination string) bool {
		for _, mount := range mounts {
			if mount.Destination == destination {
				// Already have something to mount there.
				return true
			}
		}
		return false
	}
	// Add mounts from the generated list, unless they conflict.
	for _, specMount := range spec.Mounts {
		if haveMount(specMount.Destination) {
			// Already have something to mount there, so skip this one.
			continue
		}
		mounts = append(mounts, specMount)
	}
	// Add bind mounts for important files, unless they conflict.
	for _, boundFile := range bindFiles {
		if haveMount(boundFile) {
			// Already have something to mount there, so skip this one.
			continue
		}
		mounts = append(mounts, specs.Mount{
			Source:      boundFile,
			Destination: boundFile,
			Type:        "bind",
			Options:     []string{"rbind", "ro"},
		})
	}
	// Add temporary copies of the contents of volume locations at the
	// volume locations, unless we already have something there.
	for _, volume := range volumes {
		if haveMount(volume) {
			// Already mounting something there, no need to bother.
			continue
		}
		cdir, err := b.store.ContainerDirectory(b.ContainerID)
		if err != nil {
			return errors.Wrapf(err, "error determining work directory for container %q", b.ContainerID)
		}
		subdir := digest.Canonical.FromString(volume).Hex()
		volumePath := filepath.Join(cdir, "buildah-volumes", subdir)
		logrus.Debugf("using %q for volume at %q", volumePath, volume)
		// If we need to, initialize the volume path's initial contents.
		if _, err = os.Stat(volumePath); os.IsNotExist(err) {
			if err = os.MkdirAll(volumePath, 0755); err != nil {
				return errors.Wrapf(err, "error creating directory %q for volume %q in container %q", volumePath, volume, b.ContainerID)
			}
			srcPath := filepath.Join(mountPoint, volume)
			if err = copyFileWithTar(srcPath, volumePath); err != nil && !os.IsNotExist(err) {
				return errors.Wrapf(err, "error populating directory %q for volume %q in container %q using contents of %q", volumePath, volume, b.ContainerID, srcPath)
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
	// Set the list in the spec.
	spec.Mounts = mounts
	return nil
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

	for _, envSpec := range append(b.Env(), options.Env...) {
		env := strings.SplitN(envSpec, "=", 2)
		if len(env) > 1 {
			g.AddProcessEnv(env[0], env[1])
		}
	}
	if len(command) > 0 {
		g.SetProcessArgs(command)
	} else {
		cmd := b.Cmd()
		if len(options.Cmd) > 0 {
			cmd = options.Cmd
		}
		ep := b.Entrypoint()
		if len(options.Entrypoint) > 0 {
			ep = options.Entrypoint
		}
		g.SetProcessArgs(append(ep, cmd...))
	}
	if options.WorkingDir != "" {
		g.SetProcessCwd(options.WorkingDir)
	} else if b.WorkDir() != "" {
		g.SetProcessCwd(b.WorkDir())
	}
	if options.Hostname != "" {
		g.SetHostname(options.Hostname)
	} else if b.Hostname() != "" {
		g.SetHostname(b.Hostname())
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
		g.SetProcessTerminal(terminal.IsTerminal(int(os.Stdout.Fd())))
	case WithTerminal:
		g.SetProcessTerminal(true)
	case WithoutTerminal:
		g.SetProcessTerminal(false)
	}
	if !options.NetworkDisabled {
		if err = g.RemoveLinuxNamespace("network"); err != nil {
			return errors.Wrapf(err, "error removing network namespace for run")
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
	if err = os.MkdirAll(filepath.Join(mountPoint, spec.Process.Cwd), 0755); err != nil {
		return errors.Wrapf(err, "error ensuring working directory %q exists", spec.Process.Cwd)
	}

	bindFiles := []string{"/etc/hosts", "/etc/resolv.conf"}
	err = b.setupMounts(mountPoint, spec, options.Mounts, bindFiles, b.Volumes())
	if err != nil {
		return errors.Wrapf(err, "error resolving mountpoints for container")
	}
	specbytes, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	err = ioutils.AtomicWriteFile(filepath.Join(path, "config.json"), specbytes, 0600)
	if err != nil {
		return errors.Wrapf(err, "error storing runtime configuration")
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
