package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	addFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "name or ID of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "root directory of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "a symlink to the root directory of the working container",
		},
		cli.StringFlag{
			Name:  "dest",
			Usage: "destination directory in the working container's filesystem",
		},
	}
	copyFlags = addFlags
)

func addAndCopyCmd(c *cli.Context, extractLocalArchives bool) error {
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
	dest := ""
	if c.IsSet("dest") {
		dest = c.String("dest")
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

	err = builder.Add(dest, extractLocalArchives, c.Args()...)
	if err != nil {
		return fmt.Errorf("error adding content to container: %v", err)
	}

	return nil
}

func addCmd(c *cli.Context) error {
	return addAndCopyCmd(c, true)
}

func copyCmd(c *cli.Context) error {
	return addAndCopyCmd(c, false)
}
