package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	umountFlags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Usage:  "root `directory` of the working container",
			EnvVar: "BUILDAHROOT",
		},
	}
	umountDescription = "Unmounts a working container's root filesystem"
)

func umountCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("either a container name or --root, or some combination, must be specified")
	}
	name := args[0]
	args = args.Tail()

	root := c.String("root")

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	err = builder.Unmount()
	if err != nil {
		return fmt.Errorf("error unmounting container %q: %v", builder.Container, err)
	}

	return nil
}
