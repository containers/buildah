package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	deleteDescription = "Deletes working container(s), unmounting them if necessary"
	deleteCommand     = cli.Command{
		Name:        "delete",
		Usage:       "Deletes working container(s)",
		Description: deleteDescription,
		Action:      deleteCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [...]",
	}
)

func deleteCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container ID must be specified")
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	for _, name := range args {
		builder, err := openBuilder(store, name)
		if err != nil {
			return fmt.Errorf("error reading build container %q: %v", name, err)
		}

		err = builder.Delete()
		if err != nil {
			return fmt.Errorf("error deleting container %q: %v", builder.Container, err)
		}
	}

	return nil
}
