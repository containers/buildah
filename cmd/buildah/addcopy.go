package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	addDescription  = "Adds the contents of a file, URL, or directory to a container's working\n   directory.  If a local file appears to be an archive, its contents are\n   extracted and added instead of the archive file itself."
	copyDescription = "Copies the contents of a file, URL, or directory into a container's working\n   directory"
)

func addAndCopyCmd(c *cli.Context, extractLocalArchives bool) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container ID must be specified")
	}
	name := args[0]
	args = args.Tail()

	// If list is greater then one, the last item is the destination
	dest := ""
	size := len(args)
	if size > 1 {
		dest = args[size-1]
		args = args[:size-1]
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
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
