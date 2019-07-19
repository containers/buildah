// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/containers/buildah/pkg/unshare"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	unshareDescription = "\n  Runs a command in a modified user namespace."
	unshareCommand     = &cobra.Command{
		Use:   "unshare",
		Short: "Run a command in a modified user namespace",
		Long:  unshareDescription,
		RunE:  unshareCmd,
		Example: `buildah unshare id
  buildah unshare cat /proc/self/uid_map,
  buildah unshare buildah-script.sh`,
	}
	unshareMounts []string
)

func init() {
	unshareCommand.SetUsageTemplate(UsageTemplate())
	flags := unshareCommand.Flags()
	flags.SetInterspersed(false)
	flags.StringSliceVar(&unshareMounts, "mount", []string{}, "mount the specified containers (default [])")
	rootCmd.AddCommand(unshareCommand)
}

func unshareMount(c *cobra.Command, mounts []string) ([]string, func(), error) {
	var store storage.Store
	var mountedContainers, env []string
	if len(mounts) == 0 {
		return nil, nil, nil
	}
	unmount := func() {
		for _, mounted := range mountedContainers {
			builder, err := openBuilder(getContext(), store, mounted)
			if err != nil {
				fmt.Fprintln(os.Stderr, errors.Wrapf(err, "error loading information about build container %q", mounted))
				continue
			}
			err = builder.Unmount()
			if err != nil {
				fmt.Fprintln(os.Stderr, errors.Wrapf(err, "error unmounting build container %q", mounted))
				continue
			}
		}
	}
	store, err := getStore(c)
	if err != nil {
		return nil, nil, err
	}
	for _, mountSpec := range mounts {
		mount := strings.SplitN(mountSpec, "=", 2)
		container := mountSpec
		envVar := container
		if len(mount) == 2 {
			envVar = mount[0]
			container = mount[1]
		}
		builder, err := openBuilder(getContext(), store, container)
		if err != nil {
			unmount()
			return nil, nil, errors.Wrapf(err, "error loading information about build container %q", container)
		}
		mountPoint, err := builder.Mount(builder.MountLabel)
		if err != nil {
			unmount()
			return nil, nil, errors.Wrapf(err, "error mounting build container %q", container)
		}
		logrus.Debugf("mounted container %q at %q", container, mountPoint)
		mountedContainers = append(mountedContainers, container)
		if envVar != "" {
			envSpec := fmt.Sprintf("%s=%s", envVar, mountPoint)
			logrus.Debugf("adding %q to environment", envSpec)
			env = append(env, envSpec)
		}
	}
	return env, unmount, nil
}

// unshareCmd execs whatever using the ID mappings that we want to use for ourselves
func unshareCmd(c *cobra.Command, args []string) error {
	// Set the default isolation type to use the "rootless" method.
	if _, present := os.LookupEnv("BUILDAH_ISOLATION"); !present {
		if err := os.Setenv("BUILDAH_ISOLATION", "rootless"); err != nil {
			logrus.Errorf("error setting BUILDAH_ISOLATION=rootless in environment: %v", err)
			os.Exit(1)
		}
	}

	// force reexec using the configured ID mappings
	unshare.MaybeReexecUsingUserNamespace(true)
	// exec the specified command, if there is one
	if len(args) < 1 {
		// try to exec the shell, if one's set
		shell, shellSet := os.LookupEnv("SHELL")
		if !shellSet {
			logrus.Errorf("no command specified")
			os.Exit(1)
		}
		args = []string{shell}
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = unshare.RootlessEnv()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	mountEnvs, unmountMounts, err := unshareMount(c, unshareMounts)
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Env, mountEnvs...)
	unshare.ExecRunnable(cmd, unmountMounts)
	os.Exit(1)
	return nil
}
