package main

import (
	"fmt"

	"github.com/nalind/buildah"
	"github.com/urfave/cli"
)

var (
	listFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet",
			Usage: "omit column headings",
		},
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
