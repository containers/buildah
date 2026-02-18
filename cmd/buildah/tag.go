package main

import (
	"fmt"

	"github.com/containers/buildah/pkg/parse"
	"github.com/spf13/cobra"
	"go.podman.io/common/libimage"
)

type tagOptions struct {
	tlsDetails string
}

func init() {
	var (
		tagDescription = "\n  Adds one or more additional names to locally-stored image."
		tagOpts        tagOptions
	)
	tagCommand := &cobra.Command{
		Use:   "tag",
		Short: "Add an additional name to a local image",
		Long:  tagDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tagCmd(cmd, args, tagOpts)
		},

		Example: `buildah tag imageName firstNewName
  buildah tag imageName firstNewName SecondNewName`,
		Args: cobra.MinimumNArgs(2),
	}
	tagCommand.SetUsageTemplate(UsageTemplate())

	flags := tagCommand.Flags()
	flags.StringVar(&tagOpts.tlsDetails, "tls-details", "", "path to a containers-tls-details.yaml file")

	rootCmd.AddCommand(tagCommand)
}

func tagCmd(c *cobra.Command, args []string, _ tagOptions) error {
	store, err := getStore(c)
	if err != nil {
		return err
	}
	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return fmt.Errorf("building system context: %w", err)
	}
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	// Allow tagging manifest list instead of resolving instances from manifest
	lookupOptions := &libimage.LookupImageOptions{ManifestList: true}
	image, _, err := runtime.LookupImage(args[0], lookupOptions)
	if err != nil {
		return err
	}

	for _, tag := range args[1:] {
		if err := image.Tag(tag); err != nil {
			return err
		}
	}
	return nil
}
