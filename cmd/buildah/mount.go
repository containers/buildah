package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	mountFlags = []cli.Flag{
		cli.StringFlag{
			Name:   "root",
			Usage:  "root `directory` of the working container",
			EnvVar: "BUILDAHROOT",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "`pathname` of a symbolic link to create to the root directory of the container",
		},
	}
	mountDescription = "Mounts a working container's root filesystem for manipulation"
)

func mountCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("either a container name/ID or --root or --link, or some combination, must be specified")
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

	mountPoint, err := builder.Mount("")
	if err != nil {
		return fmt.Errorf("error mounting container %q: %v", builder.Container, err)
	}

	if link != "" {
		err = builder.Link(link)
		if err != nil {
			return fmt.Errorf("error creating symbolic link to %q: %v", mountPoint, err)
		}
	}

	fmt.Printf("%s\n", mountPoint)

	return nil
}
