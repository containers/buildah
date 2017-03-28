package main

import (
	"fmt"

	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	containersFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "display only container IDs",
		},
		cli.BoolFlag{
			Name:  "noheading, n",
			Usage: "do not print column headings",
		},
		cli.BoolFlag{
			Name:  "notruncate",
			Usage: "do not truncate output",
		},
	}
	containersDescription = "Lists containers which appear to be " + buildah.Package + " working containers, their\n   names and IDs, and the names and IDs of the images from which they were\n   initialized"
	containersCommand     = cli.Command{
		Name:        "containers",
		Usage:       "List working containers and their base images",
		Description: containersDescription,
		Flags:       containersFlags,
		Action:      containersCmd,
		ArgsUsage:   " ",
	}
)

func containersCmd(c *cli.Context) error {
	store, err := getStore(c)
	if err != nil {
		return err
	}

	quiet := false
	if c.IsSet("quiet") {
		quiet = c.Bool("quiet")
	}
	noheading := false
	if c.IsSet("noheading") {
		noheading = c.Bool("noheading")
	}
	truncate := true
	if c.IsSet("notruncate") {
		truncate = !c.Bool("notruncate")
	}

	builders, err := openBuilders(store)
	if err != nil {
		return fmt.Errorf("error reading build containers: %v", err)
	}
	if len(builders) > 0 && !noheading && !quiet {
		if truncate {
			fmt.Printf("%-12s %-12s %-10s %s\n", "CONTAINER ID", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
		} else {
			fmt.Printf("%-64s %-64s %-10s %s\n", "CONTAINER ID", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
		}
	}
	for _, builder := range builders {
		if builder.FromImage == "" {
			builder.FromImage = buildah.BaseImageFakeName
		}
		if quiet {
			fmt.Printf("%s\n", builder.ContainerID)
		} else {
			if truncate {
				fmt.Printf("%-12.12s %-12.12s %-10s %s\n", builder.ContainerID, builder.FromImageID, builder.FromImage, builder.Container)
			} else {
				fmt.Printf("%-64s %-64s %-10s %s\n", builder.ContainerID, builder.FromImageID, builder.FromImage, builder.Container)
			}
		}
	}

	return nil
}
