package main

import (
	"fmt"
	"os"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
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
		Name:                   "umount",
		Aliases:                []string{"unmount"},
		Usage:                  "Unmounts the root file system on the specified working containers",
		Description:            "Unmounts the root file system on the specified working containers",
		Action:                 umountCmd,
		ArgsUsage:              "[CONTAINER-NAME-OR-ID [...]]",
		Flags:                  sortFlags(umountFlags),
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
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
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, umountFlags); err != nil {
		return err
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

			if err = builder.Unmount(); err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container)
				continue
			}
			fmt.Printf("%s\n", builder.ContainerID)
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

			if err = builder.Unmount(); err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container)
				continue
			}
			fmt.Printf("%s\n", builder.ContainerID)
		}
	}
	return lastError
}
