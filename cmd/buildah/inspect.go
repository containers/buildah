package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	inspectTypeContainer = "container"
	inspectTypeImage     = "image"
	inspectTypeManifest  = "manifest"
)

type inspectResults struct {
	format      string
	inspectType string
}

func init() {
	var (
		opts               inspectResults
		inspectDescription = "\n  Inspects a build container's or built image's configuration."
	)

	inspectCommand := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect the configuration of a container or image",
		Long:  inspectDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return inspectCmd(cmd, args, opts)
		},
		Example: `buildah inspect containerID
  buildah inspect --type image imageID
  buildah inspect --format '{{.OCIv1.Config.Env}}' alpine`,
	}
	inspectCommand.SetUsageTemplate(UsageTemplate())

	flags := inspectCommand.Flags()
	flags.SetInterspersed(false)
	flags.StringVarP(&opts.format, "format", "f", "", "use `format` as a Go template to format the output")
	flags.StringVarP(&opts.inspectType, "type", "t", inspectTypeContainer, "look at the item of the specified `type` (container or image) and name")

	rootCmd.AddCommand(inspectCommand)
}

func inspectCmd(c *cobra.Command, args []string, iopts inspectResults) error {
	var builder *buildah.Builder

	if len(args) == 0 {
		return errors.New("container or image name must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.New("too many arguments specified")
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return fmt.Errorf("building system context: %w", err)
	}

	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	ctx := getContext()

	switch iopts.inspectType {
	case inspectTypeContainer:
		builder, err = openBuilder(ctx, store, name)
		if err != nil {
			if c.Flag("type").Changed {
				return fmt.Errorf("reading build container: %w", err)
			}
			builder, err = openImage(ctx, systemContext, store, name)
			if err != nil {
				if manifestErr := manifestInspect(ctx, store, systemContext, name); manifestErr == nil {
					return nil
				}
				return err
			}
		}
	case inspectTypeImage:
		builder, err = openImage(ctx, systemContext, store, name)
		if err != nil {
			return err
		}
	case inspectTypeManifest:
		return manifestInspect(ctx, store, systemContext, name)
	default:
		return fmt.Errorf("the only recognized types are %q and %q", inspectTypeContainer, inspectTypeImage)
	}
	out := buildah.GetBuildInfo(builder)
	if iopts.format != "" {
		format := iopts.format
		if matched, err := regexp.MatchString("{{.*}}", format); err != nil {
			return fmt.Errorf("validating format provided: %s: %w", format, err)
		} else if !matched {
			return fmt.Errorf("invalid format provided: %s", format)
		}
		t, err := template.New("format").Parse(format)
		if err != nil {
			return fmt.Errorf("Template parsing error: %w", err)
		}
		if err = t.Execute(os.Stdout, out); err != nil {
			return err
		}
		if term.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Println()
		}
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if term.IsTerminal(int(os.Stdout.Fd())) {
		enc.SetEscapeHTML(false)
	}
	return enc.Encode(out)
}
