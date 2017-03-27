package main

import (
	"fmt"

	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	// TODO implement
	listFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet",
			Usage: "omit column headings",
		},
	}
	listDescription = "Lists containers which appear to be " + buildah.Package + " working containers, their\n   names and IDs, and the names and IDs of the images from which they were\n   initialized"

	listCommand = cli.Command{
		Name:        "list",
		Usage:       "List working containers and their base images",
		Description: listDescription,
		Flags:       listFlags,
		Action:      listCmd,
		ArgsUsage:   " ",
	}
)

func listCmd(c *cli.Context) error {
	store, err := getStore(c)
	if err != nil {
		return err
	}

	quiet := false
	if c.IsSet("quiet") {
		quiet = c.Bool("quiet")
	}

	builders, err := openBuilders(store)
	if err != nil {
		return fmt.Errorf("error reading build containers: %v", err)
	}
	if len(builders) > 0 && !quiet {
		fmt.Printf("%-64s %-64s %-10s %s\n", "CONTAINER ID", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
	}
	for _, builder := range builders {
		if builder.FromImage == "" {
			builder.FromImage = buildah.BaseImageFakeName
		}
		fmt.Printf("%-64s %-64s %-10s %s\n", builder.ContainerID, builder.FromImageID, builder.FromImage, builder.Container)
	}

	return nil
}
