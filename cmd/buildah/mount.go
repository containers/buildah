package main

import (
	"fmt"

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
		Name:           "mount",
		Usage:          "Mount a working container's root filesystem",
		Description:    mountDescription,
		Action:         mountCmd,
		ArgsUsage:      "CONTAINER-NAME-OR-ID",
		Flags:          mountFlags,
		SkipArgReorder: true,
	}
)

func mountCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := validateFlags(c, mountFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}
	truncate := !c.Bool("notruncate")

	if len(args) == 1 {
		name := args[0]
		builder, err := openBuilder(getContext(), store, name)
		if err != nil {
			return errors.Wrapf(err, "error reading build container %q", name)
		}
		mountPoint, err := builder.Mount(builder.MountLabel)
		if err != nil {
			return errors.Wrapf(err, "error mounting %q container %q", name, builder.Container)
		}
		fmt.Printf("%s\n", mountPoint)
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
	return nil
}
