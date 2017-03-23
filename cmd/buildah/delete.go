package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	deleteFlags = []cli.Flag{
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
	deleteDescription = "Deletes a working container, unmounting it if necessary"
)

func deleteCmd(c *cli.Context) error {
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

	err = builder.Delete()
	if err != nil {
		return fmt.Errorf("error deleting container %q: %v", builder.Container, err)
	}

	return nil
}
