package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

const (
	// DefaultTransport is a prefix that we apply to an image name if we
	// can't find one in the local Store, in order to generate a source
	// reference for the image that we can then copy to the local Store.
	DefaultTransport = "docker://"
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
			Name:  "transport",
			Usage: "`prefix` to prepend to the image name in order to pull the image",
			Value: DefaultTransport,
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
			Name:  "tls-verify",
			Usage: "Require HTTPS and verify certificates when accessing the registry",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when pulling images",
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

	image := args[0]
	transport := DefaultTransport
	if c.IsSet("transport") {
		transport = c.String("transport")
	}

	systemContext, err := systemContextFromOptions(c)
	if err != nil {
		return errors.Errorf("error building system context [%v]", err)
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
	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}

	quiet := false
	if c.IsSet("quiet") {
		quiet = c.Bool("quiet")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := buildah.BuilderOptions{
		FromImage:           image,
		Container:           name,
		PullPolicy:          pullPolicy,
		Transport:           transport,
		SignaturePolicyPath: signaturePolicy,
		SystemContext:       systemContext,
	}
	if !quiet {
		options.ReportWriter = os.Stderr
	}

	builder, err := buildah.NewBuilder(store, options)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", builder.Container)
	return builder.Save()
}
