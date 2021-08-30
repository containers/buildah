package main

import (
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/libimage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	tagDescription = "\n  Adds one or more additional names to locally-stored image."
	tagCommand     = &cobra.Command{
		Use:   "tag",
		Short: "Add an additional name to a local image",
		Long:  tagDescription,
		RunE:  tagCmd,

		Example: `buildah tag imageName firstNewName
  buildah tag imageName firstNewName SecondNewName`,
		Args: cobra.MinimumNArgs(2),
	}
)

func tagCmd(c *cobra.Command, args []string) error {
	store, err := getStore(c)
	if err != nil {
		return err
	}
	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
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

func init() {
	tagCommand.SetUsageTemplate(UsageTemplate())
	rootCmd.AddCommand(tagCommand)
}
