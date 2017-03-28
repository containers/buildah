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
		cli.BoolFlag{
			Name:  "containers",
			Usage: "print containers only",
		},
		cli.BoolFlag{
			Name:  "images",
			Usage: "print images only",
		},
		cli.BoolFlag{
			Name:  "no-trunc",
			Usage: "don't truncate output",
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

	truncate := true
	if c.IsSet("no-trunc") {
		truncate = !c.Bool("no-trunc")
	}

	containers := true
	if c.IsSet("images") {
		containers = !c.Bool("images")
	}

	images := true
	if c.IsSet("containers") {
		images = !c.Bool("containers")
	}

	if images {
		images, err := store.Images()
		if err != nil {
			return err
		}

		if len(images) > 0 && !quiet {
			fmt.Printf("\nIMAGES\n\n")
			if truncate {
				fmt.Printf("%-12s %-64s\n", "IMAGE ID", "IMAGE NAME")
			} else {
				fmt.Printf("%-64s %-64s\n", "IMAGE ID", "IMAGE NAME")
			}
		}
		for _, image := range images {
			if quiet {
				fmt.Printf("%s\n", image.ID)
				continue
			}
			names := []string{""}
			if len(image.Names) > 0 {
				names = image.Names
			}
			for _, name := range names {
				if truncate {
					fmt.Printf("%-12.12s %s\n", image.ID, name)
				} else {
					fmt.Printf("%-64s %s\n", image.ID, name)
				}
			}
		}
	}
	if containers {
		builders, err := openBuilders(store)
		if err != nil {
			return fmt.Errorf("error reading build containers: %v", err)
		}
		if len(builders) > 0 && !quiet {
			fmt.Printf("\nCONTAINERS\n\n")
			if truncate {
				fmt.Printf("%.12s %.12s %-10s %s\n", "CONTAINER ID", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
			} else {
				fmt.Printf("%.64s %.64s %-10s %s\n", "CONTAINER ID", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
			}
		}
		for _, builder := range builders {
			if builder.FromImage == "" {
				builder.FromImage = buildah.BaseImageFakeName
			}
			if truncate {
				fmt.Printf("%-12.12s %-12.12s %-10s %s\n", builder.ContainerID, builder.FromImageID, builder.FromImage, builder.Container)
			} else {
				fmt.Printf("%.64s %.64s %-10s %s\n", builder.ContainerID, builder.FromImageID, builder.FromImage, builder.Container)
			}
		}
	}
	return nil
}
