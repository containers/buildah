package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
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
		return errors.Errorf("container ID must be specified")
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	var e error
	for _, name := range args {
		builder, err := openBuilder(store, name)
		if e == nil {
			e = err
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading build container %q: %v\n", name, err)
			continue
		}

		id := builder.ContainerID
		err = builder.Delete()
		if e == nil {
			e = err
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error removing container %q: %v\n", builder.Container, err)
			continue
		}
		fmt.Printf("%s\n", id)
	}

	return e
}
