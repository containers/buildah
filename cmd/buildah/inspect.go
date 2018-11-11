package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

const (
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
		Name:                   "inspect",
		Usage:                  "Inspects the configuration of a container or image",
		Description:            inspectDescription,
		Flags:                  sortFlags(inspectFlags),
		Action:                 inspectCmd,
		ArgsUsage:              "CONTAINER-OR-IMAGE",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func inspectCmd(c *cli.Context) error {
	var builder *buildah.Builder

	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container or image name must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := parse.ValidateFlags(c, inspectFlags); err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	ctx := getContext()

	switch c.String("type") {
	case inspectTypeContainer:
		builder, err = openBuilder(ctx, store, name)
		if err != nil {
			if c.IsSet("type") {
				return errors.Wrapf(err, "error reading build container %q", name)
			}
			builder, err = openImage(ctx, systemContext, store, name)
			if err != nil {
				return errors.Wrapf(err, "error reading build object %q", name)
			}
		}
	case inspectTypeImage:
		builder, err = openImage(ctx, systemContext, store, name)
		if err != nil {
			return errors.Wrapf(err, "error reading image %q", name)
		}
	default:
		return errors.Errorf("the only recognized types are %q and %q", inspectTypeContainer, inspectTypeImage)
	}
	out := buildah.GetBuildInfo(builder)
	if c.IsSet("format") {
		format := c.String("format")
		if matched, err := regexp.MatchString("{{.*}}", format); err != nil {
			return errors.Wrapf(err, "error validating format provided: %s", format)
		} else if !matched {
			return errors.Errorf("error invalid format provided: %s", format)
		}
		t, err := template.New("format").Parse(format)
		if err != nil {
			return errors.Wrapf(err, "Template parsing error")
		}
		if err = t.Execute(os.Stdout, out); err != nil {
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
	return enc.Encode(out)
}
