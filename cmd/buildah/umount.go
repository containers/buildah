package main

import (
	"fmt"

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
		return fmt.Errorf("container ID must be specified")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments specified")
	}
	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	err = builder.Unmount()
	if err != nil {
		return fmt.Errorf("error unmounting container %q: %v", builder.Container, err)
	}

	return nil
}
