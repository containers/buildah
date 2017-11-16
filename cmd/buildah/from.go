package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	fromFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "authfile",
			Usage: "path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json",
		},
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
			Usage: "pull the image even if named image is present in store (supersedes pull option)",
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
			Usage: "require HTTPS and verify certificates when accessing the registry",
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
		return errors.Errorf("an image name (or \"scratch\") must be specified")
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

	pullPolicy := buildah.PullNever
	if c.BoolT("pull") {
		pullPolicy = buildah.PullIfMissing
	}
	if c.Bool("pull-always") {
		pullPolicy = buildah.PullAlways
	}

	signaturePolicy := c.String("signature-policy")

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := buildah.BuilderOptions{
		FromImage:             args[0],
		Container:             c.String("name"),
		PullPolicy:            pullPolicy,
		SignaturePolicyPath:   signaturePolicy,
		SystemContext:         systemContext,
		DefaultMountsFilePath: c.GlobalString("default-mounts-file"),
	}
	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}

	builder, err := buildah.NewBuilder(store, options)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", builder.Container)
	return builder.Save()
}
