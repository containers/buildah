package main

import (
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/pkg/parse"
	"github.com/projectatomic/buildah/util"
	"github.com/urfave/cli"
)

var (
	tagDescription = "Adds one or more additional names to locally-stored image"
	tagCommand     = cli.Command{
		Name:                   "tag",
		Usage:                  "Add an additional name to a local image",
		Description:            tagDescription,
		Action:                 tagCmd,
		ArgsUsage:              "IMAGE-NAME [IMAGE-NAME ...]",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func tagCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return errors.Errorf("image name and at least one new name must be specified")
	}
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
