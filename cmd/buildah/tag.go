package main

import (
	"fmt"

	"github.com/projectatomic/buildah/util"
	"github.com/urfave/cli"
)

var (
	tagDescription = "Adds one or more additional names to locally-stored image"
	tagCommand     = cli.Command{
		Name:        "tag",
		Usage:       "Add an additional name to a local image",
		Description: tagDescription,
		Action:      tagCmd,
		ArgsUsage:   "IMAGE-NAME [IMAGE-NAME ...]",
	}
)

func tagCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return fmt.Errorf("image name and at least one new name must be specified")
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}
	img, err := util.FindImage(store, args[0])
	if err != nil {
		return fmt.Errorf("error finding local image %q: %v", args[0], err)
	}
	err = util.AddImageNames(store, img, args[1:])
	if err != nil {
		return fmt.Errorf("error adding names %v to image %q: %v", args[1:], args[0], err)
	}
	return nil
}
