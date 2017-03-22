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
		cli.StringFlag{
			Name:  "link",
			Usage: "`pathname` of a symbolic link to the root directory of the working container",
		},
	}
	umountDescription = "Unmounts a working container's root filesystem"
)

func umountCmd(c *cli.Context) error {
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
		if link == "" {
			return fmt.Errorf("link location can not be empty")
		}
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root, link)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	err = builder.Unmount()
	if err != nil {
		return fmt.Errorf("error unmounting container %q: %v", builder.Container, err)
	}

	return nil
}
