package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/util"
	"github.com/urfave/cli"
)

var (
	umountCommand = cli.Command{
		Name:           "umount",
		Aliases:        []string{"unmount"},
		Usage:          "Unmounts the root file system on the specified working containers",
		Description:    "Unmounts the root file system on the specified working containers",
		Action:         umountCmd,
		ArgsUsage:      "CONTAINER-NAME-OR-ID [...]",
		SkipArgReorder: true,
	}
)

func umountCmd(c *cli.Context) error {
	umountContainerErrStr := "error unmounting container"
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("at least one container ID must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	var lastError error
	for _, name := range args {

		builder, err := openBuilder(store, name)
		if err != nil {
			lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %s", umountContainerErrStr, name), lastError)
			continue
		}

		id := builder.ContainerID
		if err = builder.Unmount(); err != nil {
			lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container), lastError)
			continue
		}
		fmt.Printf("%s\n", id)
	}
	return lastError
}
