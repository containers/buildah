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
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/runtime-tools/generate"
)

const (
	// DefaultWorkingDir is used if none was specified.
	DefaultWorkingDir = "/"
)

// RunOptions can be used to alter how a command is run in the container.
type RunOptions struct {
	// Hostname is the hostname we set for the running container.
	Hostname string
}

func getExportOptions() generate.ExportOptions {
	return generate.ExportOptions{}
}

// Run runs the specified command in the container's root filesystem.
func (b *Builder) Run(command []string, options RunOptions) error {
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
	config := b.updatedConfig()
	image := v1.Image{}
	err = json.Unmarshal(config, &image)
	if err != nil {
		return err
	}
	user, err := getUser(image.Config.User)
	if err != nil {
		return err
	}
	g := generate.New()

	if image.OS != "" {
		g.SetPlatformOS(image.OS)
	}
	if image.Architecture != "" {
		g.SetPlatformArch(image.Architecture)
	}
	g.SetProcessUID(user.UID)
	g.SetProcessGID(user.GID)
	for _, envSpec := range image.Config.Env {
		env := strings.SplitN(envSpec, "=", 2)
		if len(env) > 1 {
			g.AddProcessEnv(env[0], env[1])
		}
	}
	if len(command) > 0 {
		g.SetProcessArgs(command)
	} else if len(image.Config.Cmd) != 0 {
		g.SetProcessArgs(image.Config.Cmd)
	} else if len(image.Config.Entrypoint) != 0 {
		g.SetProcessArgs(image.Config.Entrypoint)
	}
	if image.Config.WorkingDir != "" {
		g.SetProcessCwd(image.Config.WorkingDir)
	}
	if options.Hostname != "" {
		g.SetHostname(options.Hostname)
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
	g.SetProcessTerminal(true)
	spec := g.Spec()
	if spec.Process.Cwd == "" {
		spec.Process.Cwd = DefaultWorkingDir
	}
	specbytes, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	err = ioutils.AtomicWriteFile(filepath.Join(path, "config.json"), specbytes, 0600)
	if err != nil {
		return fmt.Errorf("error storing runtime configuration: %v", err)
	}
	logrus.Debugf("config = %v", string(specbytes))
	cmd := exec.Command("runc", "--debug", "run", "-b", path, Package+"-"+b.ContainerID)
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
