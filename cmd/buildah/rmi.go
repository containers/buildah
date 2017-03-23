package main

import (
	"fmt"

	"github.com/urfave/cli"
)

var (
	rmiDescription = "Removes one or more locally stored images."
	rmiCommand     = cli.Command{
		Name:        "rmi",
		Usage:       "Removes one or more images from local storage",
		Description: rmiDescription,
		Action:      rmiCmd,
		ArgsUsage:   "IMAGE-NAME-OR-ID [...]",
	}
)

func rmiCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container ID must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	for _, id := range args {
		_, err := store.DeleteImage(id, true)
		if err != nil {
			return fmt.Errorf("error removing image %q: %v", id, err)
		}
		fmt.Printf("%s\n", id)
	}

	return nil
}
