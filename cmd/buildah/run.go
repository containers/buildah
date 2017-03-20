package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	runFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "`name or ID` of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "root `directory` of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "`pathname` of a symbolic link to the root directory of the working container",
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
	}
	runDescription = "Runs a specified command using the container's root filesystem as a root\n   filesystem, using configuration settings inherited from the container's\n   image or as specified using previous calls to the config command"
)

func runCmd(c *cli.Context) error {
	args := c.Args()
	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	root := ""
	if c.IsSet("root") {
		root = c.String("root")
	}
	link := ""
	if c.IsSet("link") {
		link = c.String("link")
	}
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
	if name == "" && root == "" && link == "" {
		if len(args) == 0 {
			return fmt.Errorf("either a container name or --root or --link, or some combination, must be specified")
		}
		name = args[0]
		args = args.Tail()
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root, link)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
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
