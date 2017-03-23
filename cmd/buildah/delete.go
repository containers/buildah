package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	deleteFlags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Usage:  "root `directory` of the working container",
			EnvVar: "BUILDAHROOT",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "`pathname` of a symbolic link to the root directory of the working container",
		},
	}
	deleteDescription = "Deletes a working container, unmounting it if necessary"
)

func deleteCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("either a container name or --root or --link, or some combination, must be specified")
	}
	name := args[0]
	args = args.Tail()

	root := c.String("root")
	link := ""
	if c.IsSet("link") {
		link = c.String("link")
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root, link)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	err = builder.Delete()
	if err != nil {
		return fmt.Errorf("error deleting container %q: %v", builder.Container, err)
	}

	return nil
}
