package main

import (
	"fmt"

	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage/pkg/archive"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	commitFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "disable-compression",
			Usage: "don't compress layers",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
	}
	commitDescription = "Writes a new image using the container's read-write layer and, if it is based\n   on an image, the layers of that image"
	commitCommand     = cli.Command{
		Name:        "commit",
		Usage:       "Create an image from a working container",
		Description: commitDescription,
		Flags:       commitFlags,
		Action:      commitCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID IMAGE",
	}
)

func commitCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container ID must be specified")
	}
	name := args[0]
	args = args.Tail()
	if len(args) == 0 {
		return fmt.Errorf("an image name must be specified")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments specified")
	}
	image := args[0]

	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}
	compress := archive.Uncompressed
	if !c.IsSet("disable-compression") || !c.Bool("disable-compression") {
		compress = archive.Gzip
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	dest, err := alltransports.ParseImageName(image)
	if err != nil {
		return fmt.Errorf("error parsing target image name %q: %v", image, err)
	}

	options := buildah.CommitOptions{
		Compression:         compress,
		SignaturePolicyPath: signaturePolicy,
	}
	err = builder.Commit(dest, options)
	if err != nil {
		return fmt.Errorf("error committing container %q to %q: %v", builder.Container, image, err)
	}

	return nil
}
