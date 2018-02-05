package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	runFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "add-host",
			Usage: "add a custom host-to-IP mapping (host:ip) (default [])",
		},
		cli.StringFlag{
			Name:  "cgroup-parent",
			Usage: "optional parent cgroup for the container",
		},
		cli.Uint64Flag{
			Name:  "cpu-period",
			Usage: "limit the CPU CFS (Completely Fair Scheduler) period",
		},
		cli.Int64Flag{
			Name:  "cpu-quota",
			Usage: "limit the CPU CFS (Completely Fair Scheduler) quota",
		},
		cli.Uint64Flag{
			Name:  "cpu-shares, c",
			Usage: "CPU shares (relative weight)",
		},
		cli.StringFlag{
			Name:  "cpuset-cpus",
			Usage: "CPUs in which to allow execution (0-3, 0,1)",
		},
		cli.StringFlag{
			Name:  "cpuset-mems",
			Usage: "memory nodes (MEMs) in which to allow execution (0-3, 0,1). Only effective on NUMA systems.",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "set the hostname inside of the container",
		},
		cli.StringFlag{
			Name:  "memory, m",
			Usage: "memory limit (format: <number>[<unit>], where unit = b, k, m or g)",
		},
		cli.StringFlag{
			Name:  "memory-swap",
			Usage: "swap limit equal to memory plus swap: '-1' to enable unlimited swap",
		},
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
			Name:  "security-opt",
			Usage: "security Options (default [])",
		},
		cli.BoolFlag{
			Name:  "tty",
			Usage: "allocate a pseudo-TTY in the container",
		},
		cli.StringSliceFlag{
			Name:  "ulimit",
			Usage: "Ulimit options (default [])",
		},
		cli.StringSliceFlag{
			Name:  "volume, v",
			Usage: "bind mount a host location into the container while running the command",
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
	if err := validateFlags(c, runFlags); err != nil {
		return err
	}

	args = args.Tail()
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	runtimeFlags := []string{}
	for _, arg := range c.StringSlice("runtime-flag") {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	options := buildah.RunOptions{
		Hostname: c.String("hostname"),
		Runtime:  c.String("runtime"),
		Args:     runtimeFlags,
	}

	if c.IsSet("tty") {
		if c.Bool("tty") {
			options.Terminal = buildah.WithTerminal
		} else {
			options.Terminal = buildah.WithoutTerminal
		}
	}

	for _, volumeSpec := range c.StringSlice("volume") {
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
