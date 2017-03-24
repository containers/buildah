package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	umountDescription = "Unmounts a working container's root filesystem"
)

func umountCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container ID must be specified")
	}
	name := args[0]
	args = args.Tail()

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
