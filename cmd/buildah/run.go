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
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	runFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "cap-add",
			Usage: "add the specified capability (default [])",
		},
		cli.StringSliceFlag{
			Name:  "cap-drop",
			Usage: "drop the specified capability (default [])",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "set the hostname inside of the container",
		},
		cli.StringFlag{
			Name:  "isolation",
			Usage: "which process isolation `type` to use",
		},
		cli.StringFlag{
			Name:  "runtime",
			Usage: "`path` to an alternate OCI runtime",
			Value: util.Runtime(),
		},
		cli.StringSliceFlag{
			Name:  "runtime-flag",
			Usage: "add global flags for the container runtime",
		},
		cli.BoolFlag{
			Name:  "no-pivot",
			Usage: "do not use pivot root to jail process inside rootfs",
		},
		cli.StringSliceFlag{
			Name:  "security-opt",
			Usage: "security options (default [])",
		},
		cli.BoolFlag{
			Name:  "t, tty, terminal",
			Usage: "allocate a pseudo-TTY in the container",
		},
		cli.StringSliceFlag{
			Name:  "volume, v",
			Usage: "bind mount a host location into the container while running the command",
		},
	}
	runDescription = "Runs a specified command using the container's root filesystem as a root\n   filesystem, using configuration settings inherited from the container's\n   image or as specified using previous calls to the config command"
	runCommand     = cli.Command{
		Name:                   "run",
		Usage:                  "Run a command inside of the container",
		Description:            runDescription,
		Flags:                  sortFlags(append(append(runFlags, userFlags...), buildahcli.NamespaceFlags...)),
		Action:                 runCmd,
		ArgsUsage:              "CONTAINER-NAME-OR-ID COMMAND [ARGS [...]]",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func runCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := args[0]
	if err := parse.ValidateFlags(c, append(append(runFlags, userFlags...), buildahcli.NamespaceFlags...)); err != nil {
		return err
	}

	args = args.Tail()
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

	isolation := builder.Isolation
	if c.IsSet("isolation") {
		isolation, err = parse.IsolationOption(c)
		if err != nil {
			return err
		}
	}

	runtimeFlags := []string{}
	for _, arg := range c.StringSlice("runtime-flag") {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	noPivot := c.Bool("no-pivot") || (os.Getenv("BUILDAH_NOPIVOT") != "")

	namespaceOptions, networkPolicy, err := parse.NamespaceOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing namespace-related options")
	}

	options := buildah.RunOptions{
		Hostname:         c.String("hostname"),
		Runtime:          c.String("runtime"),
		Args:             runtimeFlags,
		NoPivot:          noPivot,
		User:             c.String("user"),
		Isolation:        isolation,
		NamespaceOptions: namespaceOptions,
		ConfigureNetwork: networkPolicy,
		CNIPluginPath:    c.String("cni-plugin-path"),
		CNIConfigDir:     c.String("cni-config-dir"),
		AddCapabilities:  c.StringSlice("cap-add"),
		DropCapabilities: c.StringSlice("cap-drop"),
	}

	if c.IsSet("tty") {
		if c.Bool("tty") {
			options.Terminal = buildah.WithTerminal
		} else {
			options.Terminal = buildah.WithoutTerminal
		}
	}

	// validate volume paths
	if err := parse.ParseVolumes(c.StringSlice("volume")); err != nil {
		return err
	}

	for _, volumeSpec := range c.StringSlice("volume") {
		volSpec := strings.Split(volumeSpec, ":")
		if len(volSpec) >= 2 {
			var mountOptions string
			if len(volSpec) >= 3 {
				mountOptions = volSpec[2]
			}
			mountOpts := strings.Split(mountOptions, ",")
			mount := specs.Mount{
				Source:      volSpec[0],
				Destination: volSpec[1],
				Type:        "bind",
				Options:     mountOpts,
			}
			options.Mounts = append(options.Mounts, mount)
		}
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
	return runerr
}
