package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	deleteDescription = "Deletes a working container, unmounting it if necessary"
)

func deleteCmd(c *cli.Context) error {
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

	err = builder.Delete()
	if err != nil {
		return fmt.Errorf("error deleting container %q: %v", builder.Container, err)
	}

	return nil
}
