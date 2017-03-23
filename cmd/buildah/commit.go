package main

import (
	"fmt"

	"github.com/containers/image/transports"
	"github.com/containers/storage/pkg/archive"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	commitFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "`name or ID` of the working container",
		},
		cli.StringFlag{
			Name:   "root",
			Usage:  "root `directory` of the working container",
			EnvVar: "BUILDAHROOT",
		},
		cli.BoolFlag{
			Name:  "disable-compression",
			Usage: "don't compress layers",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "`name` of output image to write",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
	}
	commitDescription = "Writes a new image using the container's read-write layer and, if it is based\n   on an image, the layers of that image"
)

func commitCmd(c *cli.Context) error {
	args := c.Args()
	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	root := c.String("root")

	output := ""
	if c.IsSet("output") {
		output = c.String("output")
	}
	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}
	compress := archive.Uncompressed
	if !c.IsSet("disable-compression") || !c.Bool("disable-compression") {
		compress = archive.Gzip
	}
	if name == "" && root == "" {
		if len(args) == 0 {
			return fmt.Errorf("either a container name or --root, or some combination, must be specified")
		}
		name = args[0]
		args = args.Tail()
	}
	if output == "" {
		if len(args) == 0 {
			return fmt.Errorf("an image name or the --output flag must be specified")
		}
		output = args[0]
		args = args.Tail()
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	dest, err := transports.ParseImageName(output)
	if err != nil {
		return fmt.Errorf("error parsing target image name %q: %v", output, err)
	}

	options := buildah.CommitOptions{
		Compression:         compress,
		SignaturePolicyPath: signaturePolicy,
	}
	err = builder.Commit(dest, options)
	if err != nil {
		return fmt.Errorf("error committing container %q to %q: %v", builder.Container, output, err)
	}

	return nil
}
