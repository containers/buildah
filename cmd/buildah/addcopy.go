package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	addFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "`name or ID` of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "root `directory` of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "`pathname` of a symbolic link to the root directory of the working container",
		},
		cli.StringFlag{
			Name:  "dest",
			Usage: "destination `directory` (if absolute) or subdirectory of the working directory (if relative) in the working container's filesystem",
		},
	}
	copyFlags       = addFlags
	addDescription  = "Adds the contents of a file, URL, or directory to a container's working\n   directory.  If a local file appears to be an archive, its contents are\n   extracted and added instead of the archive file itself."
	copyDescription = "Copies the contents of a file, URL, or directory into a container's working\n   directory"
)

func addAndCopyCmd(c *cli.Context, extractLocalArchives bool) error {
	args := c.Args()
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
		if len(args) == 0 {
			return fmt.Errorf("either a container name or --root or --link, or some combination, must be specified")
		}
		name = args[0]
		args = args.Tail()
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root, link)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	err = builder.Add(dest, extractLocalArchives, args...)
	if err != nil {
		return fmt.Errorf("error adding content to container %q: %v", builder.Container, err)
	}

	return nil
}

func addCmd(c *cli.Context) error {
	return addAndCopyCmd(c, true)
}

func copyCmd(c *cli.Context) error {
	return addAndCopyCmd(c, false)
}
