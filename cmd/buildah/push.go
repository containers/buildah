package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	pushFlags = []cli.Flag{
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
			Name:  "disable-compression, D",
			Usage: "don't compress layers",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when pushing images",
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
	pushDescription = fmt.Sprintf(`
   Pushes an image to a specified location.

   The Image "DESTINATION" uses a "transport":"details" format.

   Supported transports:
   %s

   See buildah-push(1) section "DESTINATION" for the expected format
`, strings.Join(transports.ListNames(), ", "))

	pushCommand = cli.Command{
		Name:        "push",
		Usage:       "Push an image to a specified destination",
		Description: pushDescription,
		Flags:       pushFlags,
		Action:      pushCmd,
		ArgsUsage:   "IMAGE DESTINATION",
	}
)

func pushCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) < 2 {
		return errors.New("source and destination image IDs must be specified")
	}
	src := args[0]
	destSpec := args[1]

	compress := archive.Gzip
	if c.Bool("disable-compression") {
		compress = archive.Uncompressed
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	dest, err := alltransports.ParseImageName(destSpec)
	// add the docker:// transport to see if they neglected it.
	if err != nil {
		if strings.Contains(destSpec, "://") {
			return err
		}

		destSpec = "docker://" + destSpec
		dest2, err2 := alltransports.ParseImageName(destSpec)
		if err2 != nil {
			return err
		}
		dest = dest2
	}

	systemContext, err := systemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	options := buildah.PushOptions{
		Compression:         compress,
		SignaturePolicyPath: c.String("signature-policy"),
		Store:               store,
		SystemContext:       systemContext,
	}
	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}

	err = buildah.Push(src, dest, options)
	if err != nil {
		return errors.Wrapf(err, "error pushing image %q to %q", src, destSpec)
	}

	return nil
}
