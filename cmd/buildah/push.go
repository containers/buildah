package main

import (
	"os"

	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	pushFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "disable-compression, D",
			Usage: "don't compress layers",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when pushing images",
		},
	}
	pushDescription = "Pushes an image to a specified location."
	pushCommand     = cli.Command{
		Name:        "push",
		Usage:       "Push an image to a specified location",
		Description: pushDescription,
		Flags:       pushFlags,
		Action:      pushCmd,
		ArgsUsage:   "IMAGE [TRANSPORT:]IMAGE",
	}
)

func pushCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return errors.New("source and destination image IDs must be specified")
	}
	src := args[0]
	destSpec := args[1]

	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}
	compress := archive.Uncompressed
	if !c.IsSet("disable-compression") || !c.Bool("disable-compression") {
		compress = archive.Gzip
	}
	quiet := false
	if c.IsSet("quiet") {
		quiet = c.Bool("quiet")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}
	dest, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		return err
	}

	options := buildah.PushOptions{
		Compression:         compress,
		SignaturePolicyPath: signaturePolicy,
		Store:               store,
	}
	if !quiet {
		options.ReportWriter = os.Stderr
	}

	err = buildah.Push(src, dest, options)
	if err != nil {
		return errors.Wrapf(err, "error pushing image %q to %q", src, destSpec)
	}

	return nil
}
