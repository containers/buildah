// +build linux

package main

import (
	"os"
	"os/exec"

	"github.com/containers/buildah/pkg/unshare"
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
)

func init() {
	unshareCommand.SetUsageTemplate(UsageTemplate())
	flags := unshareCommand.Flags()
	flags.SetInterspersed(false)
	rootCmd.AddCommand(unshareCommand)
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
	unshare.ExecRunnable(cmd)
	os.Exit(1)
	return nil
}
