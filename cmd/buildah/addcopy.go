package main

import (
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	addAndCopyFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "chown",
			Usage: "Set the user and group ownership of the destination content",
		},
	}
	addDescription  = "Adds the contents of a file, URL, or directory to a container's working\n   directory.  If a local file appears to be an archive, its contents are\n   extracted and added instead of the archive file itself."
	copyDescription = "Copies the contents of a file, URL, or directory into a container's working\n   directory"

	addCommand = cli.Command{
		Name:        "add",
		Usage:       "Add content to the container",
		Description: addDescription,
		Flags:       addAndCopyFlags,
		Action:      addCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...] [DESTINATION]",
	}

	copyCommand = cli.Command{
		Name:        "copy",
		Usage:       "Copy content into the container",
		Description: copyDescription,
		Flags:       addAndCopyFlags,
		Action:      copyCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...] [DESTINATION]",
	}
)

func addAndCopyCmd(c *cli.Context, extractLocalArchives bool) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := args[0]
	args = args.Tail()

	if err := validateFlags(c, addAndCopyFlags); err != nil {
		return err
	}

	// If list is greater than one, the last item is the destination
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

	options := buildah.AddAndCopyOptions{
		Chown: c.String("chown"),
	}

	if err := builder.Add(dest, extractLocalArchives, options, args...); err != nil {
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
