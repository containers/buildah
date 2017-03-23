package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	mountFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "`name or ID` of the working container",
		},
		cli.StringFlag{
			Name:   "root",
			Usage:  "root `directory` of the working container",
			EnvVar: "BUILDAHROOT",
		},
	}
	mountDescription = "Mounts a working container's root filesystem for manipulation"
)

func mountCmd(c *cli.Context) error {
	args := c.Args()
	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	root := c.String("root")
	if name == "" && root == "" {
		if len(args) == 0 {
			return fmt.Errorf("either a container name or --root, or some combination, must be specified")
		}
		name = args[0]
		args = args.Tail()
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	mountPoint, err := builder.Mount("")
	if err != nil {
		return fmt.Errorf("error mounting container %q: %v", builder.Container, err)
	}

	fmt.Printf("%s\n", mountPoint)

	return nil
}
