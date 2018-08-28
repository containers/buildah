package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/pkg/parse"
	"github.com/projectatomic/buildah/util"
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
		Name:                   "rm",
		Aliases:                []string{"delete"},
		Usage:                  "Remove one or more working containers",
		Description:            rmDescription,
		Action:                 rmCmd,
		ArgsUsage:              "CONTAINER-NAME-OR-ID [...]",
		Flags:                  rmFlags,
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func rmCmd(c *cli.Context) error {
	delContainerErrStr := "error removing container"
	args := c.Args()
	if len(args) == 0 && !c.Bool("all") {
		return errors.Errorf("container ID must be specified")
	}
	if len(args) > 0 && c.Bool("all") {
		return errors.Errorf("when using the --all switch, you may not pass any containers names or IDs")
	}
	if err := parse.ValidateFlags(c, rmFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	var lastError error
	if c.Bool("all") {
		builders, err := openBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build containers")
		}

		for _, builder := range builders {
			if err = builder.Delete(); err != nil {
				lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", delContainerErrStr, builder.Container), lastError)
				continue
			}
			fmt.Printf("%s\n", builder.ContainerID)
		}
	} else {
		for _, name := range args {
			builder, err := openBuilder(getContext(), store, name)
			if err != nil {
				lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", delContainerErrStr, name), lastError)
				continue
			}
			if err = builder.Delete(); err != nil {
				lastError = util.WriteError(os.Stderr, errors.Wrapf(err, "%s %q", delContainerErrStr, name), lastError)
				continue
			}
			fmt.Printf("%s\n", builder.ContainerID)
		}

	}
	return lastError
}
