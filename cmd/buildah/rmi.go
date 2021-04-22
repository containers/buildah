package main

import (
	"context"
	"fmt"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/libimage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type rmiOptions struct {
	all   bool
	prune bool
	force bool
}

func init() {
	var (
		rmiDescription = "\n  Removes one or more locally stored images."
		opts           rmiOptions
	)
	rmiCommand := &cobra.Command{
		Use:   "rmi",
		Short: "Remove one or more images from local storage",
		Long:  rmiDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rmiCmd(cmd, args, opts)
		},
		Example: `buildah rmi imageID
  buildah rmi --all --force
  buildah rmi imageID1 imageID2 imageID3`,
	}
	rmiCommand.SetUsageTemplate(UsageTemplate())

	flags := rmiCommand.Flags()
	flags.SetInterspersed(false)

	flags.BoolVarP(&opts.all, "all", "a", false, "remove all images")
	flags.BoolVarP(&opts.prune, "prune", "p", false, "prune dangling images")
	flags.BoolVarP(&opts.force, "force", "f", false, "force removal of the image and any containers using the image")

	rootCmd.AddCommand(rmiCommand)
}

func rmiCmd(c *cobra.Command, args []string, iopts rmiOptions) error {
	if len(args) == 0 && !iopts.all && !iopts.prune {
		return errors.Errorf("image name or ID must be specified")
	}
	if len(args) > 0 && iopts.all {
		return errors.Errorf("when using the --all switch, you may not pass any images names or IDs")
	}
	if iopts.all && iopts.prune {
		return errors.Errorf("when using the --all switch, you may not use --prune switch")
	}
	if len(args) > 0 && iopts.prune {
		return errors.Errorf("when using the --prune switch, you may not pass any images names or IDs")
	}

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return err
	}
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	options := &libimage.RemoveImagesOptions{
		Filters: []string{"readonly=false"},
	}
	if iopts.prune {
		options.Filters = append(options.Filters, "dangling=true")
	} else if !iopts.all {
		options.Filters = append(options.Filters, "intermediate=false")
	}
	options.Force = iopts.force

	untagged, removed, err := runtime.RemoveImages(context.Background(), args, options)
	for _, u := range untagged {
		fmt.Printf("untagged: %s\n", u)
	}
	for _, r := range removed {
		fmt.Printf("%s\n", r)
	}
	return err
}
