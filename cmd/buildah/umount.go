package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	umountFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "umount all of the currently mounted containers",
		},
	}
	umountCommand = cli.Command{
		Name:           "umount",
		Aliases:        []string{"unmount"},
		Usage:          "Unmounts the root file system on the specified working containers",
		Description:    "Unmounts the root file system on the specified working containers",
		Action:         umountCmd,
		ArgsUsage:      "CONTAINER-NAME-OR-ID [...]",
		Flags:          umountFlags,
		SkipArgReorder: true,
	}
)

func umountCmd(c *cli.Context) error {
	umountAll := c.Bool("all")
	umountContainerErrStr := "error unmounting container"
	args := c.Args()
	if len(args) == 0 && !umountAll {
		return errors.Errorf("at least one container ID must be specified")
	}
	if len(args) > 0 && umountAll {
		return errors.Errorf("when using the --all switch, you may not pass any container IDs")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	var lastError error
	if len(args) > 0 {
		for _, name := range args {
			builder, err := openBuilder(getContext(), store, name)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %s", umountContainerErrStr, name)
				continue
			}
			if builder.MountPoint == "" {
				continue
			}

			id := builder.ContainerID
			if err = builder.Unmount(); err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container)
				continue
			}
			fmt.Printf("%s\n", id)
		}
	} else {
		builders, err := openBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build Containers")
		}
		for _, builder := range builders {
			if builder.MountPoint == "" {
				continue
			}

			id := builder.ContainerID
			if err = builder.Unmount(); err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container)
				continue
			}
			fmt.Printf("%s\n", id)
		}
	}
	return lastError
}
