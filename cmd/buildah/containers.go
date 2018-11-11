package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type jsonContainer struct {
	ID            string `json:"id"`
	Builder       bool   `json:"builder"`
	ImageID       string `json:"imageid"`
	ImageName     string `json:"imagename"`
	ContainerName string `json:"containername"`
}

type containerOutputParams struct {
	ContainerID   string
	Builder       string
	ImageID       string
	ImageName     string
	ContainerName string
}

type containerOptions struct {
	all        bool
	format     string
	json       bool
	noHeading  bool
	noTruncate bool
	quiet      bool
}

type containerFilterParams struct {
	id       string
	name     string
	ancestor string
}

var (
	containersFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "also list non-buildah containers",
		},
		cli.StringFlag{
			Name:  "filter, f",
			Usage: "filter output based on conditions provided",
		},
		cli.StringFlag{
			Name:  "format",
			Usage: "pretty-print containers using a Go template",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "output in JSON format",
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
			Name:  "quiet, q",
			Usage: "display only container IDs",
		},
	}
	containersDescription = "Lists containers which appear to be " + buildah.Package + " working containers, their\n   names and IDs, and the names and IDs of the images from which they were\n   initialized"
	containersCommand     = cli.Command{
		Name:                   "containers",
		Usage:                  "List working containers and their base images",
		Description:            containersDescription,
		Flags:                  sortFlags(containersFlags),
		Action:                 containersCmd,
		ArgsUsage:              " ",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func containersCmd(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return errors.New("'buildah containers' does not accept arguments")
	}
	if err := parse.ValidateFlags(c, containersFlags); err != nil {
		return err
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	if c.IsSet("quiet") && c.IsSet("format") {
		return errors.Errorf("quiet and format are mutually exclusive")
	}

	opts := containerOptions{
		all:        c.Bool("all"),
		format:     c.String("format"),
		json:       c.Bool("json"),
		noHeading:  c.Bool("noheading"),
		noTruncate: c.Bool("notruncate"),
		quiet:      c.Bool("quiet"),
	}

	var params *containerFilterParams
	if c.IsSet("filter") {
		params, err = parseCtrFilter(c.String("filter"))
		if err != nil {
			return errors.Wrapf(err, "error parsing filter")
		}
	}

	if !opts.noHeading && !opts.quiet && opts.format == "" && !opts.json {
		containerOutputHeader(!opts.noTruncate)
	}

	return outputContainers(store, opts, params)
}

func outputContainers(store storage.Store, opts containerOptions, params *containerFilterParams) error {
	seenImages := make(map[string]string)
	imageNameForID := func(id string) string {
		if id == "" {
			return buildah.BaseImageFakeName
		}
		imageName, ok := seenImages[id]
		if ok {
			return imageName
		}
		img, err2 := store.Image(id)
		if err2 == nil && len(img.Names) > 0 {
			seenImages[id] = img.Names[0]
		}
		return seenImages[id]
	}

	builders, err := openBuilders(store)
	if err != nil {
		return errors.Wrapf(err, "error reading build containers")
	}
	var (
		containerOutput []containerOutputParams
		JSONContainers  []jsonContainer
	)
	if !opts.all {
		// only output containers created by buildah
		for _, builder := range builders {
			image := imageNameForID(builder.FromImageID)
			if !matchesCtrFilter(builder.ContainerID, builder.Container, builder.FromImageID, image, params) {
				continue
			}
			if opts.json {
				JSONContainers = append(JSONContainers, jsonContainer{ID: builder.ContainerID,
					Builder:       true,
					ImageID:       builder.FromImageID,
					ImageName:     image,
					ContainerName: builder.Container})
				continue
			}
			output := containerOutputParams{
				ContainerID:   builder.ContainerID,
				Builder:       "   *",
				ImageID:       builder.FromImageID,
				ImageName:     image,
				ContainerName: builder.Container,
			}
			containerOutput = append(containerOutput, output)
		}
	} else {
		// output all containers currently in storage
		builderMap := make(map[string]struct{})
		for _, builder := range builders {
			builderMap[builder.ContainerID] = struct{}{}
		}
		containers, err2 := store.Containers()
		if err2 != nil {
			return errors.Wrapf(err2, "error reading list of all containers")
		}
		for _, container := range containers {
			name := ""
			if len(container.Names) > 0 {
				name = container.Names[0]
			}
			_, ours := builderMap[container.ID]
			builder := ""
			if ours {
				builder = "   *"
			}
			if !matchesCtrFilter(container.ID, name, container.ImageID, imageNameForID(container.ImageID), params) {
				continue
			}
			if opts.json {
				JSONContainers = append(JSONContainers, jsonContainer{ID: container.ID,
					Builder:       ours,
					ImageID:       container.ImageID,
					ImageName:     imageNameForID(container.ImageID),
					ContainerName: name})
				continue
			}
			output := containerOutputParams{
				ContainerID:   container.ID,
				Builder:       builder,
				ImageID:       container.ImageID,
				ImageName:     imageNameForID(container.ImageID),
				ContainerName: name,
			}
			containerOutput = append(containerOutput, output)
		}
	}
	if opts.json {
		data, err := json.MarshalIndent(JSONContainers, "", "    ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", data)
		return nil
	}

	for _, ctr := range containerOutput {
		if opts.quiet {
			fmt.Printf("%-64s\n", ctr.ContainerID)
			continue
		}
		if opts.format != "" {
			if err := containerOutputUsingTemplate(opts.format, ctr); err != nil {
				return err
			}
			continue
		}
		containerOutputUsingFormatString(!opts.noTruncate, ctr)
	}
	return nil
}

func containerOutputUsingTemplate(format string, params containerOutputParams) error {
	if matched, err := regexp.MatchString("{{.*}}", format); err != nil {
		return errors.Wrapf(err, "error validating format provided: %s", format)
	} else if !matched {
		return errors.Errorf("error invalid format provided: %s", format)
	}

	tmpl, err := template.New("container").Parse(format)
	if err != nil {
		return errors.Wrapf(err, "Template parsing error")
	}

	err = tmpl.Execute(os.Stdout, params)
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

func containerOutputUsingFormatString(truncate bool, params containerOutputParams) {
	if truncate {
		fmt.Printf("%-12.12s  %-8s %-12.12s %-32s %s\n", params.ContainerID, params.Builder, params.ImageID, params.ImageName, params.ContainerName)
	} else {
		fmt.Printf("%-64s %-8s %-64s %-32s %s\n", params.ContainerID, params.Builder, params.ImageID, params.ImageName, params.ContainerName)
	}
}

func containerOutputHeader(truncate bool) {
	if truncate {
		fmt.Printf("%-12s  %-8s %-12s %-32s %s\n", "CONTAINER ID", "BUILDER", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
	} else {
		fmt.Printf("%-64s %-8s %-64s %-32s %s\n", "CONTAINER ID", "BUILDER", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
	}
}

func parseCtrFilter(filter string) (*containerFilterParams, error) {
	params := new(containerFilterParams)
	filters := strings.Split(filter, ",")
	for _, param := range filters {
		pair := strings.SplitN(param, "=", 2)
		if len(pair) != 2 {
			return nil, errors.Errorf("incorrect filter value %q, should be of form filter=value", param)
		}
		switch strings.TrimSpace(pair[0]) {
		case "id":
			params.id = pair[1]
		case "name":
			params.name = pair[1]
		case "ancestor":
			params.ancestor = pair[1]
		default:
			return nil, errors.Errorf("invalid filter %q", pair[0])
		}
	}
	return params, nil
}

func matchesCtrName(ctrName, argName string) bool {
	return strings.Contains(ctrName, argName)
}

func matchesAncestor(imgName, imgID, argName string) bool {
	if matchesID(imgID, argName) {
		return true
	}
	return matchesReference(imgName, argName)
}

func matchesCtrFilter(ctrID, ctrName, imgID, imgName string, params *containerFilterParams) bool {
	if params == nil {
		return true
	}
	if params.id != "" && !matchesID(ctrID, params.id) {
		return false
	}
	if params.name != "" && !matchesCtrName(ctrName, params.name) {
		return false
	}
	if params.ancestor != "" && !matchesAncestor(imgName, imgID, params.ancestor) {
		return false
	}
	return true
}
