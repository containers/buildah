package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/Sirupsen/logrus"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	runFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "runtime",
			Usage: "`path` to an alternate runtime",
			Value: buildah.DefaultRuntime,
		},
		cli.StringSliceFlag{
			Name:  "runtime-flag",
			Usage: "add global flags for the container runtime",
		},
		cli.StringSliceFlag{
			Name:  "volume, v",
			Usage: "bind mount a host location into the container while running the command",
		},
		cli.BoolFlag{
			Name:  "tty",
			Usage: "allocate a pseudo-TTY in the container",
		},
	}
	runDescription = "Runs a specified command using the container's root filesystem as a root\n   filesystem, using configuration settings inherited from the container's\n   image or as specified using previous calls to the config command"
	runCommand     = cli.Command{
		Name:        "run",
		Usage:       "Run a command inside of the container",
		Description: runDescription,
		Flags:       runFlags,
		Action:      runCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID COMMAND [ARGS [...]]",
	}
)

func runCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := args[0]
	args = args.Tail()

	runtime := ""
	if c.IsSet("runtime") {
		runtime = c.String("runtime")
	}
	flags := []string{}
	if c.IsSet("runtime-flag") {
		flags = c.StringSlice("runtime-flag")
	}
	volumes := []string{}
	if c.IsSet("v") || c.IsSet("volume") {
		volumes = c.StringSlice("volume")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	hostname := ""
	if c.IsSet("hostname") {
		hostname = c.String("hostname")
	}
	options := buildah.RunOptions{
		Hostname: hostname,
		Runtime:  runtime,
		Args:     flags,
	}

	if c.IsSet("tty") {
		if c.Bool("tty") {
			options.Terminal = buildah.WithTerminal
		} else {
			options.Terminal = buildah.WithoutTerminal
		}
	}

	for _, volumeSpec := range volumes {
		volSpec := strings.Split(volumeSpec, ":")
		if len(volSpec) >= 2 {
			mountOptions := "bind"
			if len(volSpec) >= 3 {
				mountOptions = mountOptions + "," + volSpec[2]
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
