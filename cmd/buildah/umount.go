package main

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	umountCommand = cli.Command{
		Name:        "umount",
		Aliases:     []string{"unmount"},
		Usage:       "Unmount a working container's root filesystem",
		Description: "Unmounts a working container's root filesystem",
		Action:      umountCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID",
	}
)

func umountCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	err = builder.Unmount()
	if err != nil {
		return errors.Wrapf(err, "error unmounting container %q", builder.Container)
	}

	return nil
}
