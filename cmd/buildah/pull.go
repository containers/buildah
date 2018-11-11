package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	util "github.com/containers/buildah/util"
	is "github.com/containers/image/storage"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	pullFlags = []cli.Flag{
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
			Usage: "use `[username[:password]]` for accessing the registry",
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
	pullDescription = `Pulls an image from a registry and stores it locally.
An image can be pulled using its tag or digest. If a tag is not
specified, the image with the 'latest' tag (if it exists) is pulled.`

	pullCommand = cli.Command{
		Name:           "pull",
		Usage:          "Pull an image from the specified location",
		Description:    pullDescription,
		Flags:          sortFlags(append(pullFlags)),
		Action:         pullCmd,
		ArgsUsage:      "IMAGE",
		SkipArgReorder: true,
	}
)

func pullCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("an image name must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := parse.ValidateFlags(c, pullFlags); err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	signaturePolicy := c.String("signature-policy")

	store, err := getStore(c)
	if err != nil {
		return err
	}

	transport := util.DefaultTransport
	arr := strings.SplitN(args[0], ":", 2)
	if len(arr) == 2 {
		if _, ok := util.Transports[arr[0]]; ok {
			transport = arr[0]
		}
	}

	options := buildah.PullOptions{
		Transport:           transport,
		SignaturePolicyPath: signaturePolicy,
		Store:               store,
		SystemContext:       systemContext,
	}

	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}

	ref, err := buildah.Pull(getContext(), args[0], options)
	if err != nil {
		return err
	}

	img, err := is.Transport.GetStoreImage(store, ref)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", img.ID)
	return nil
}
