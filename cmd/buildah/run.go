package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runInputOptions struct {
	addHistory     bool
	capAdd         []string
	capDrop        []string
	hostname       string
	isolation      string
	runtime        string
	runtimeFlag    []string
	noPivot        bool
	securityOption []string
	terminal       bool
	volumes        []string
	*buildahcli.NameSpaceResults
}

func init() {
	var (
		runDescription = "\n  Runs a specified command using the container's root filesystem as a root\n  filesystem, using configuration settings inherited from the container's\n  image or as specified using previous calls to the config command."
		opts           runInputOptions
	)

	namespaceResults := buildahcli.NameSpaceResults{}

	runCommand := &cobra.Command{
		Use:   "run",
		Short: "Run a command inside of the container",
		Long:  runDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.NameSpaceResults = &namespaceResults
			return runCmd(cmd, args, opts)

		},
		Example: `buildah run containerID -- ps -auxw
  buildah run --tty containerID /bin/bash
  buildah run --volume /path/on/host:/path/in/container:ro,z containerID /bin/sh`,
	}
	runCommand.SetUsageTemplate(UsageTemplate())

	flags := runCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolVar(&opts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	flags.StringSliceVar(&opts.capAdd, "cap-add", []string{}, "add the specified capability (default []")
	flags.StringSliceVar(&opts.capDrop, "cap-drop", []string{}, "drop the specified capability (default [])")
	flags.StringVar(&opts.hostname, "hostname", "", "set the hostname inside of the container")
	flags.StringVar(&opts.isolation, "isolation", buildahcli.DefaultIsolation(), "`type` of process isolation to use. Use BUILDAH_ISOLATION environment variable to override.")
	flags.StringVar(&opts.runtime, "runtime", util.Runtime(), "`path` to an alternate OCI runtime")
	flags.StringSliceVar(&opts.runtimeFlag, "runtime-flag", []string{}, "add global flags for the container runtime")
	flags.BoolVar(&opts.noPivot, "no-pivot", false, "do not use pivot root to jail process inside rootfs")
	flags.StringArrayVar(&opts.securityOption, "security-opt", []string{}, "security options (default [])")
	// TODO add-third alias for tty
	flags.BoolVarP(&opts.terminal, "terminal", "t", false, "allocate a pseudo-TTY in the container")
	flags.BoolVar(&opts.terminal, "tty", false, "allocate a pseudo-TTY in the container")
	flags.MarkHidden("tty")
	flags.StringSliceVarP(&opts.volumes, "volume", "v", []string{}, "bind mount a host location into the container while running the command")

	userFlags := getUserFlags()
	namespaceFlags := buildahcli.GetNameSpaceFlags(&namespaceResults)

	flags.AddFlagSet(&userFlags)
	flags.AddFlagSet(&namespaceFlags)

	rootCmd.AddCommand(runCommand)
}

func runCmd(c *cobra.Command, args []string, iopts runInputOptions) error {
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := args[0]
	args = Tail(args)
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	if len(args) == 0 {
		return errors.Errorf("command must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	isolation, err := parse.IsolationOption(c)
	if err != nil {
		return err
	}

	runtimeFlags := []string{}
	for _, arg := range iopts.runtimeFlag {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	noPivot := iopts.noPivot || (os.Getenv("BUILDAH_NOPIVOT") != "")

	namespaceOptions, networkPolicy, err := parse.NamespaceOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing namespace-related options")
	}

	options := buildah.RunOptions{
		Hostname:         iopts.hostname,
		Runtime:          iopts.runtime,
		Args:             runtimeFlags,
		NoPivot:          noPivot,
		User:             c.Flag("user").Value.String(),
		Isolation:        isolation,
		NamespaceOptions: namespaceOptions,
		ConfigureNetwork: networkPolicy,
		CNIPluginPath:    iopts.CNIPlugInPath,
		CNIConfigDir:     iopts.CNIConfigDir,
		AddCapabilities:  iopts.capAdd,
		DropCapabilities: iopts.capDrop,
	}

	if c.Flag("terminal").Changed {
		if iopts.terminal {
			options.Terminal = buildah.WithTerminal
		} else {
			options.Terminal = buildah.WithoutTerminal
		}
	}

	// validate volume paths
	if err := parse.ParseVolumes(iopts.volumes); err != nil {
		return err
	}

	for _, volumeSpec := range iopts.volumes {
		mount, err := parse.ParseVolume(volumeSpec)
		if err != nil {
			return err
		}
		options.Mounts = append(options.Mounts, mount)
	}
	runerr := builder.Run(args, options)
	if runerr != nil {
		logrus.Debugf("error running %v in container %q: %v", args, builder.Container, runerr)
	}
	if ee, ok := runerr.(*exec.ExitError); ok {
		if w, ok := ee.Sys().(syscall.WaitStatus); ok {
			os.Exit(w.ExitStatus())
		}
	}
	if runerr == nil {
		shell := "/bin/sh -c"
		if len(builder.Shell()) > 0 {
			shell = strings.Join(builder.Shell(), " ")
		}
		conditionallyAddHistory(builder, c, "%s %s", shell, strings.Join(args, " "))
		return builder.Save()
	}
	return runerr
}
