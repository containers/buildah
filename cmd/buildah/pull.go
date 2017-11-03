package main

import (
	"fmt"
	"os"

	"github.com/containers/image/transports"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	pullFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "cert-dir",
			Value: "",
			Usage: "use certificates at the specified path to access the registry",
		},
		cli.StringFlag{
			Name:  "creds",
			Value: "",
			Usage: "use `username[:password]` for accessing the registry",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when pulling images",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.BoolTFlag{
			Name:  "tls-verify",
			Usage: "Require HTTPS and verify certificates when accessing the registry",
		},
	}
	pullDescription = "Pull selected image, you can specify the source, please consult the man page for more info"

	pullCommand = cli.Command{
		Name:        "pull",
		Usage:       "Pull selected image",
		Description: pullDescription,
		Flags:       pullFlags,
		Action:      pullCmd,
		ArgsUsage:   "IMAGE",
	}
)

func pullCmd(c *cli.Context) error {

	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("an image name must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := validateFlags(c, fromFlags); err != nil {
		return err
	}

	systemContext, err := systemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	signaturePolicy := c.String("signature-policy")

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := buildah.BuilderOptions{
		FromImage:           args[0],
		SignaturePolicyPath: signaturePolicy,
		SystemContext:       systemContext,
	}
	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}
	pulledReference, err := buildah.PullImage(store, options, systemContext)
	if err != nil {
		return errors.Wrapf(err, "error pulling image %q", args[0])
	}

	if err != nil {
		return err
	}

	// FIXME: I want to print the image digest here but am struggling to figure it out
	fmt.Printf("%s\n", transports.ImageName(pulledReference))
	return nil
}
