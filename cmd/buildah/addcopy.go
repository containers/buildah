package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	addDescription  = "Adds the contents of a file, URL, or directory to a container's working\n   directory.  If a local file appears to be an archive, its contents are\n   extracted and added instead of the archive file itself."
	copyDescription = "Copies the contents of a file, URL, or directory into a container's working\n   directory"

	addCommand = cli.Command{
		Name:        "add",
		Usage:       "Add content to the container",
		Description: addDescription,
		Action:      addCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...] [DESTINATION]",
	}

	copyCommand = cli.Command{
		Name:        "copy",
		Usage:       "Copy content into the container",
		Description: copyDescription,
		Action:      copyCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...] [DESTINATION]",
	}
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
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	err = builder.Add(dest, extractLocalArchives, args...)
	if err != nil {
		return errors.Wrapf(err, "error adding content to container %q", builder.Container)
	}

	return nil
}

func addCmd(c *cli.Context) error {
	return addAndCopyCmd(c, true)
}

func copyCmd(c *cli.Context) error {
	return addAndCopyCmd(c, false)
}
