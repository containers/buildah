package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

const (
	defaultFormat = `Container: {{.Container}}
ID: {{.ContainerID}}
`
	inspectTypeContainer = "container"
	inspectTypeImage     = "image"
)

var (
	inspectFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "type, t",
			Usage: "look at the item of the specified `type` (container or image) and name",
		},
		cli.StringFlag{
			Name:  "format, f",
			Usage: "use `format` as a Go template to format the output",
		},
	}
	inspectDescription = "Inspects a build container's or built image's configuration."
	inspectCommand     = cli.Command{
		Name:        "inspect",
		Usage:       "Inspects the configuration of a container or image",
		Description: inspectDescription,
		Flags:       inspectFlags,
		Action:      inspectCmd,
		ArgsUsage:   "CONTAINER-OR-IMAGE",
	}
)

func inspectCmd(c *cli.Context) error {
	var builder *buildah.Builder

	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container or image name must be specified")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments specified")
	}

	itemType := inspectTypeContainer
	if c.IsSet("type") {
		itemType = c.String("type")
	}
	switch itemType {
	case inspectTypeContainer:
	case inspectTypeImage:
	default:
		return fmt.Errorf("the only recognized types are %q and %q", inspectTypeContainer, inspectTypeImage)
	}

	format := defaultFormat
	if c.IsSet("format") {
		if c.String("format") != "" {
			format = c.String("format")
		}
	}
	t := template.Must(template.New("format").Parse(format))

	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	switch itemType {
	case inspectTypeContainer:
		builder, err = openBuilder(store, name)
		if err != nil {
			return errors.Wrapf(err, "error reading build container %q", name)
		}
	case inspectTypeImage:
		builder, err = openImage(store, name)
		if err != nil {
			return errors.Wrapf(err, "error reading image %q", name)
		}
	}

	if c.IsSet("format") {
		return t.Execute(os.Stdout, builder)
	}

	b, err := json.MarshalIndent(builder, "", "    ")
	if err != nil {
		return errors.Wrapf(err, "error encoding build container as json")
	}
	_, err = fmt.Println(string(b))
	return err
}
