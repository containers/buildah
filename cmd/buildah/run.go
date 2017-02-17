package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/nalind/buildah"
	"github.com/urfave/cli"
)

var (
	runFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "name of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "root directory of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "symlink to the root directory of the working container",
		},
		cli.StringFlag{
			Name:  "runtime",
			Usage: "use an alternate runtime",
			Value: buildah.DefaultRuntime,
		},
		cli.StringSliceFlag{
			Name:  "runtime-flag",
			Usage: "add global flags for the container runtime",
		},
	}
)

func runCmd(c *cli.Context) error {
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
		return fmt.Errorf("either --name or --root or --link, or some combination, must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root, link)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	updateConfig(builder, c)
	hostname := ""
	if c.IsSet("hostname") {
		hostname = c.String("hostname")
	}
	options := buildah.RunOptions{
		Hostname: hostname,
		Runtime:  runtime,
		Args:     flags,
	}
	runerr := builder.Run(c.Args(), options)
	if runerr != nil {
		logrus.Debugf("error running %v in container: %v", c.Args(), runerr)
	}
	if ee, ok := runerr.(*exec.ExitError); ok {
		if w, ok := ee.Sys().(syscall.WaitStatus); ok {
			os.Exit(w.ExitStatus())
		}
	}
	return runerr
}
