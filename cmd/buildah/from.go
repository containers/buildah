package main

import (
	"fmt"

	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

const (
	// DefaultRegistry is a prefix that we apply to an image name if we
	// can't find one in the local Store, in order to generate a source
	// reference for the image that we can then copy to the local Store.
	DefaultRegistry = "docker://"
)

var (
	fromFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "`name` for the working container",
		},
		cli.BoolTFlag{
			Name:  "pull",
			Usage: "pull the image if not present",
		},
		cli.BoolFlag{
			Name:  "pull-always",
			Usage: "pull the image even if one with the same name is already present",
		},
		cli.StringFlag{
			Name:  "registry",
			Usage: "`prefix` to prepend to the image name in order to pull the image",
			Value: DefaultRegistry,
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.BoolFlag{
			Name:  "mount",
			Usage: "mount the working container",
		},
	}
	fromDescription = "Creates a new working container, either from scratch or using a specified\n   image as a starting point"

	fromCommand = cli.Command{
		Name:        "from",
		Usage:       "Create a working container based on an image",
		Description: fromDescription,
		Flags:       fromFlags,
		Action:      fromCmd,
		ArgsUsage:   "IMAGE",
	}
)

func fromCmd(c *cli.Context) error {

	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("an image name (or \"scratch\") must be specified")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments specified")
	}
	image := args[0]

	registry := DefaultRegistry
	if c.IsSet("registry") {
		registry = c.String("registry")
	}
	pull := true
	if c.IsSet("pull") {
		pull = c.BoolT("pull")
	}
	pullAlways := false
	if c.IsSet("pull-always") {
		pull = c.Bool("pull-always")
	}

	pullPolicy := buildah.PullNever
	if pull {
		pullPolicy = buildah.PullIfMissing
	}
	if pullAlways {
		pullPolicy = buildah.PullAlways
	}

	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	mount := false
	if c.IsSet("mount") {
		mount = c.Bool("mount")
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
		PullPolicy:          pullPolicy,
		Mount:               mount,
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
