package main

import (
	"fmt"

	"github.com/containers/image/transports"
	"github.com/containers/storage/pkg/archive"
	"github.com/nalind/buildah"
	"github.com/urfave/cli"
)

var (
	commitFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "name of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "root directory of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "symlink to the root directory of the working container",
		},
		cli.BoolFlag{
			Name:  "do-not-compress",
			Usage: "don't compress layers",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "image to create",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "signature policy path",
		},
	}
)

func commitCmd(c *cli.Context) error {
	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	root := ""
	if c.IsSet("root") {
		root = c.String("root")
	}
	link := ""
	if c.IsSet("link") {
		link = c.String("link")
	}
	output := ""
	if c.IsSet("output") {
		output = c.String("output")
	}
	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}
	compress := archive.Uncompressed
	if !c.IsSet("do-not-compress") || !c.Bool("do-not-compress") {
		compress = archive.Gzip
	}
	if output == "" {
		return fmt.Errorf("the --output flag must be specified")
	}
	if name == "" && root == "" && link == "" {
		return fmt.Errorf("either --name or --root or --link, or some combination, must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name, root, link)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	dest, err := transports.ParseImageName(output)
	if err != nil {
		return fmt.Errorf("error parsing target image name %q: %v", name, err)
	}

	options := buildah.CommitOptions{
		Compression:         compress,
		SignaturePolicyPath: signaturePolicy,
	}
	updateConfig(builder, c)
	err = builder.Commit(dest, options)
	if err != nil {
		return fmt.Errorf("error committing container to %q: %v", output, err)
	}

	return nil
}
