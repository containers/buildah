package main

import (
	"fmt"

	"github.com/nalind/buildah"
	"github.com/urfave/cli"
)

const (
	DefaultRegistry = "docker://"
)

var (
	fromFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "set a name for the working container",
		},
		cli.StringFlag{
			Name:  "image",
			Usage: "name of the starting image",
		},
		cli.BoolFlag{
			Name:  "pull",
			Usage: "pull the image if not present",
		},
		cli.BoolFlag{
			Name:  "pull-always",
			Usage: "pull the image, even if a version is present",
		},
		cli.StringFlag{
			Name:  "registry",
			Usage: "prefix to prepend to the image name in order to pull the image",
			Value: DefaultRegistry,
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "signature policy path",
		},
		cli.BoolFlag{
			Name:  "mount",
			Usage: "mount the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "name of a symlink to create to the root directory of the container",
		},
	}
)

func fromCmd(c *cli.Context) error {
	image := ""
	if c.IsSet("image") {
		image = c.String("image")
	} else {
		return fmt.Errorf("an image name (or \"scratch\") must be specified")
	}
	registry := DefaultRegistry
	if c.IsSet("registry") {
		registry = c.String("registry")
	}
	pull := false
	if c.IsSet("pull") {
		pull = c.Bool("pull")
	}
	pullAlways := false
	if c.IsSet("pull-always") {
		pull = c.Bool("pull-always")
	}
	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	mount := false
	if c.IsSet("mount") {
		mount = c.Bool("mount")
	}
	link := ""
	if c.IsSet("link") {
		link = c.String("link")
		if link == "" {
			return fmt.Errorf("link location can not be empty")
		}
	}
	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := buildah.BuilderOptions{
		FromImage:           image,
		Container:           name,
		PullIfMissing:       pull,
		PullAlways:          pullAlways,
		Mount:               mount,
		Link:                link,
		Registry:            registry,
		SignaturePolicyPath: signaturePolicy,
	}

	builder, err := buildah.NewBuilder(store, options)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", builder.Container)
	if options.Mount {
		fmt.Printf("%s\n", builder.MountPoint)
	}

	return builder.Save()
}
