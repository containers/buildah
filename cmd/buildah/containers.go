package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/containers/buildah"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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

type containersResults struct {
	all        bool
	filter     string
	format     string
	json       bool
	noheading  bool
	notruncate bool
	quiet      bool
}

func init() {
	var (
		containersDescription = "\n  Lists containers which appear to be " + buildah.Package + " working containers, their\n  names and IDs, and the names and IDs of the images from which they were\n  initialized."
		opts                  containersResults
	)
	containersCommand := &cobra.Command{
		Use:     "containers",
		Aliases: []string{"list", "ls", "ps"},
		Short:   "List working containers and their base images",
		Long:    containersDescription,
		//Flags:                  sortFlags(containersFlags),
		RunE: func(cmd *cobra.Command, args []string) error {
			return containersCmd(cmd, args, opts)
		},
		Example: `buildah containers
  buildah containers --format "{{.ContainerID}} {{.ContainerName}}"
  buildah containers -q --noheading --notruncate`,
	}
	containersCommand.SetUsageTemplate(UsageTemplate())

	flags := containersCommand.Flags()
	flags.BoolVarP(&opts.all, "all", "a", false, "also list non-buildah containers")
	flags.StringVarP(&opts.filter, "filter", "f", "", "filter output based on conditions provided")
	flags.StringVar(&opts.format, "format", "", "pretty-print containers using a Go template")
	flags.BoolVar(&opts.json, "json", false, "output in JSON format")
	flags.BoolVarP(&opts.noheading, "noheading", "n", false, "do not print column headings")
	flags.BoolVar(&opts.notruncate, "notruncate", false, "do not truncate output")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "display only container IDs")

	rootCmd.AddCommand(containersCommand)
}

func containersCmd(c *cobra.Command, args []string, iopts containersResults) error {
	if len(args) > 0 {
		return errors.New("'buildah containers' does not accept arguments")
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	if c.Flag("quiet").Changed && c.Flag("format").Changed {
		return errors.Errorf("quiet and format are mutually exclusive")
	}

	opts := containerOptions{
		all:        iopts.all,
		format:     iopts.format,
		json:       iopts.json,
		noHeading:  iopts.noheading,
		noTruncate: iopts.notruncate,
		quiet:      iopts.quiet,
	}

	var params *containerFilterParams
	if c.Flag("filter").Changed {
		params, err = parseCtrFilter(iopts.filter)
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
