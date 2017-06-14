package main

import (
	"fmt"

	"github.com/pkg/errors"
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
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "also list non-buildah containers",
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
	all := false
	if c.IsSet("all") {
		all = c.Bool("all")
	}

	list := func(n int, containerID, imageID, image, container string, isBuilder bool) {
		if n == 0 && !noheading && !quiet {
			if truncate {
				fmt.Printf("%-12s  %-8s %-12s %-32s %s\n", "CONTAINER ID", "BUILDER", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
			} else {
				fmt.Printf("%-64s %-8s %-64s %-32s %s\n", "CONTAINER ID", "BUILDER", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
			}
		}
		if quiet {
			fmt.Printf("%s\n", containerID)
		} else {
			isBuilderValue := ""
			if isBuilder {
				isBuilderValue = "   *"
			}
			if truncate {
				fmt.Printf("%-12.12s  %-8s %-12.12s %-32s %s\n", containerID, isBuilderValue, imageID, image, container)
			} else {
				fmt.Printf("%-64s %-8s %-64s %-32s %s\n", containerID, isBuilderValue, imageID, image, container)
			}
		}
	}
	seenImages := make(map[string]string)
	imageNameForID := func(id string) string {
		if id == "" {
			return buildah.BaseImageFakeName
		}
		imageName, ok := seenImages[id]
		if ok {
			return imageName
		}
		img, err := store.Image(id)
		if err == nil && len(img.Names) > 0 {
			seenImages[id] = img.Names[0]
		}
		return seenImages[id]
	}

	builders, err := openBuilders(store)
	if err != nil {
		return errors.Wrapf(err, "error reading build containers")
	}
	if !all {
		for i, builder := range builders {
			image := imageNameForID(builder.FromImageID)
			list(i, builder.ContainerID, builder.FromImageID, image, builder.Container, true)
		}
	} else {
		builderMap := make(map[string]struct{})
		for _, builder := range builders {
			builderMap[builder.ContainerID] = struct{}{}
		}
		containers, err2 := store.Containers()
		if err2 != nil {
			return errors.Wrapf(err2, "error reading list of all containers")
		}
		for i, container := range containers {
			name := ""
			if len(container.Names) > 0 {
				name = container.Names[0]
			}
			_, ours := builderMap[container.ID]
			list(i, container.ID, container.ImageID, imageNameForID(container.ImageID), name, ours)
		}
	}

	return nil
}
