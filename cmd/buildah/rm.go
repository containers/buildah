package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	rmDescription = "Removes one or more working containers, unmounting them if necessary"
	rmFlags       = []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "remove all containers",
		},
	}
	rmCommand = cli.Command{
		Name:        "rm",
		Aliases:     []string{"delete"},
		Usage:       "Remove one or more working containers",
		Description: rmDescription,
		Action:      rmCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID [...]",
		Flags:       rmFlags,
	}
)

func rmCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 && !c.Bool("all") {
		return errors.Errorf("container ID must be specified")
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	var e error
	if c.Bool("all") {
		builders, err := openBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build containers")
		}

		for _, builder := range builders {
			err = builder.Delete()
			if e == nil {
				e = err
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "error removing container %q: %v\n", builder.Container, err)
				continue
			}
		}
	} else {
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

	}
	return e
}
