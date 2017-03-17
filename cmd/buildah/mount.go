package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	mountFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "name or `ID` of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "a previous root `directory` of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "`pathname` of a symlink to create to the root directory of the container",
		},
	}
)

func mountCmd(c *cli.Context) error {
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
		if link == "" {
			return fmt.Errorf("link location can not be empty")
		}
	}
	if name == "" && root == "" {
		return fmt.Errorf("either --name or --root, or both, must be specified")
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
		return fmt.Errorf("error mounting container: %v", err)
	}

	if link != "" {
		err = builder.Link(link)
		if err != nil {
			return fmt.Errorf("error creating symlink to %q: %v", mountPoint, err)
		}
	}

	fmt.Printf("%s\n", mountPoint)

	return nil
}
