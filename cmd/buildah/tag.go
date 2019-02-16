package main

import (
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
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
	_, img, err := util.FindImage(store, "", systemContext, args[0])
	if err != nil {
		return errors.Wrapf(err, "error finding local image %q", args[0])
	}
	if err := util.AddImageNames(store, "", systemContext, img, args[1:]); err != nil {
		return errors.Wrapf(err, "error adding names %v to image %q", args[1:], args[0])
	}
	return nil
}

func init() {
	tagCommand.SetUsageTemplate(UsageTemplate())
	rootCmd.AddCommand(tagCommand)
}
