package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	deleteFlags = []cli.Flag{
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
	}
)

func deleteCmd(c *cli.Context) error {
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

	err = builder.Delete()
	if err != nil {
		return fmt.Errorf("error deleting container: %v", err)
	}

	return nil
}
