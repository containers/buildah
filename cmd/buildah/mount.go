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
	mountDescription = "Mounts a working container's root filesystem for manipulation"
	mountFlags       = []cli.Flag{
		cli.BoolFlag{
			Name:  "notruncate",
			Usage: "do not truncate output",
		},
	}
	mountCommand = cli.Command{
		Name:                   "mount",
		Usage:                  "Mount a working container's root filesystem",
		Description:            mountDescription,
		Action:                 mountCmd,
		ArgsUsage:              "[CONTAINER-NAME-OR-ID [...]]",
		Flags:                  sortFlags(mountFlags),
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func mountCmd(c *cli.Context) error {
	args := c.Args()

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, mountFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}
	truncate := !c.Bool("notruncate")

	var lastError error
	if len(args) > 0 {
		for _, name := range args {
			builder, err := openBuilder(getContext(), store, name)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error reading build container %q", name)
				continue
			}
			mountPoint, err := builder.Mount(builder.MountLabel)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error mounting %q container %q", name, builder.Container)
				continue
			}
			if len(args) > 1 {
				fmt.Printf("%s %s\n", name, mountPoint)
			} else {
				fmt.Printf("%s\n", mountPoint)
			}
		}
	} else {
		builders, err := openBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build containers")
		}
		for _, builder := range builders {
			if builder.MountPoint == "" {
				continue
			}
			if truncate {
				fmt.Printf("%-12.12s %s\n", builder.ContainerID, builder.MountPoint)
			} else {
				fmt.Printf("%-64s %s\n", builder.ContainerID, builder.MountPoint)
			}
		}
	}
	return lastError
}
