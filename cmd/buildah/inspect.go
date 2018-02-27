package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
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
			Name:  "format, f",
			Usage: "use `format` as a Go template to format the output",
		},
		cli.StringFlag{
			Name:  "type, t",
			Usage: "look at the item of the specified `type` (container or image) and name",
			Value: inspectTypeContainer,
		},
	}
	inspectDescription = "Inspects a build container's or built image's configuration."
	inspectCommand     = cli.Command{
		Name:           "inspect",
		Usage:          "Inspects the configuration of a container or image",
		Description:    inspectDescription,
		Flags:          inspectFlags,
		Action:         inspectCmd,
		ArgsUsage:      "CONTAINER-OR-IMAGE",
		SkipArgReorder: true,
	}
)

func inspectCmd(c *cli.Context) error {
	var builder *buildah.Builder

	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container or image name must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := validateFlags(c, inspectFlags); err != nil {
		return err
	}

	systemContext, err := systemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	format := defaultFormat
	if c.String("format") != "" {
		format = c.String("format")
	}
	t := template.Must(template.New("format").Parse(format))

	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	switch c.String("type") {
	case inspectTypeContainer:
		builder, err = openBuilder(store, name)
		if err != nil {
			if c.IsSet("type") {
				return errors.Wrapf(err, "error reading build container %q", name)
			}
			builder, err = openImage(systemContext, store, name)
			if err != nil {
				return errors.Wrapf(err, "error reading build object %q", name)
			}
		}
	case inspectTypeImage:
		builder, err = openImage(systemContext, store, name)
		if err != nil {
			return errors.Wrapf(err, "error reading image %q", name)
		}
	default:
		return errors.Errorf("the only recognized types are %q and %q", inspectTypeContainer, inspectTypeImage)
	}

	if c.IsSet("format") {
		if err := t.Execute(os.Stdout, buildah.GetBuildInfo(builder)); err != nil {
			return err
		}
		if terminal.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Println()
		}
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		enc.SetEscapeHTML(false)
	}
	return enc.Encode(builder)
}
