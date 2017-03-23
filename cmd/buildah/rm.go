package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	rmDescription = "Removes one or more working containers, unmounting them if necessary"
	rmCommand     = cli.Command{
		Name:        "rm",
		Aliases:     []string{"delete"},
		Usage:       "Remove one or more working containers",
		Description: rmDescription,
		Action:      rmCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [...]",
	}
)

func rmCmd(c *cli.Context) error {
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
			return fmt.Errorf("error removing container %q: %v", builder.Container, err)
		}
	}

	return nil
}
