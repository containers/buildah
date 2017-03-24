package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	mountDescription = "Mounts a working container's root filesystem for manipulation"
)

func mountCmd(c *cli.Context) error {
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

	mountPoint, err := builder.Mount("")
	if err != nil {
		return fmt.Errorf("error mounting container %q: %v", builder.Container, err)
	}

	fmt.Printf("%s\n", mountPoint)

	return nil
}
