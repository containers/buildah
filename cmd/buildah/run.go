package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/Sirupsen/logrus"
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
	if c.IsSet("runtime") {
		runtime = c.String("runtime")
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
