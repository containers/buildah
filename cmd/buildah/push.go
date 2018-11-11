package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/image/manifest"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage/pkg/archive"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	pushFlags = []cli.Flag{
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
			Name:  "disable-compression, D",
			Usage: "don't compress layers",
		},
		cli.StringFlag{
			Name:  "format, f",
			Usage: "manifest type (oci, v2s1, or v2s2) to use when saving image using the 'dir:' transport (default is manifest type of source)",
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
			Usage: "require HTTPS and verify certificates when accessing the registry",
		},
	}
	pushDescription = fmt.Sprintf(`
   Pushes an image to a specified location.

   The Image "DESTINATION" uses a "transport":"details" format. If not specified, will reuse source IMAGE as DESTINATION.

   Supported transports:
   %s

   See buildah-push(1) section "DESTINATION" for the expected format
`, getListOfTransports())

	pushCommand = cli.Command{
		Name:                   "push",
		Usage:                  "Push an image to a specified destination",
		Description:            pushDescription,
		Flags:                  sortFlags(pushFlags),
		Action:                 pushCmd,
		ArgsUsage:              "IMAGE [DESTINATION]",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func pushCmd(c *cli.Context) error {
	var src, destSpec string
	args := c.Args()

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, pushFlags); err != nil {
		return err
	}

	switch len(args) {
	case 0:
		return errors.New("At least a source image ID must be specified")
	case 1:
		src = args[0]
		destSpec = src
		logrus.Debugf("Destination argument not specified, assuming the same as the source: %s", destSpec)
	case 2:
		src = args[0]
		destSpec = args[1]
	default:
		return errors.New("Only two arguments are necessary to push: source and destination")
	}

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
		destTransport := strings.Split(destSpec, ":")[0]
		if t := transports.Get(destTransport); t != nil {
			return err
		}

		if strings.Contains(destSpec, "://") {
			return err
		}

		destSpec = "docker://" + destSpec
		dest2, err2 := alltransports.ParseImageName(destSpec)
		if err2 != nil {
			return err
		}
		dest = dest2
		logrus.Debugf("Assuming docker:// as the transport method for DESTINATION: %s", destSpec)
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	var manifestType string
	if c.IsSet("format") {
		switch c.String("format") {
		case "oci":
			manifestType = imgspecv1.MediaTypeImageManifest
		case "v2s1":
			manifestType = manifest.DockerV2Schema1SignedMediaType
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return fmt.Errorf("unknown format %q. Choose on of the supported formats: 'oci', 'v2s1', or 'v2s2'", c.String("format"))
		}
	}

	options := buildah.PushOptions{
		Compression:         compress,
		ManifestType:        manifestType,
		SignaturePolicyPath: c.String("signature-policy"),
		Store:               store,
		SystemContext:       systemContext,
	}
	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}

	ref, digest, err := buildah.Push(getContext(), src, dest, options)
	if err != nil {
		return util.GetFailureCause(err, errors.Wrapf(err, "error pushing image %q to %q", src, destSpec))
	}
	if ref != nil {
		logrus.Debugf("pushed image %q with digest %s", ref, digest.String())
	} else {
		logrus.Debugf("pushed image with digest %s", digest.String())
	}

	return nil
}

// getListOfTransports gets the transports supported from the image library
// and strips of the "tarball" transport from the string of transports returned
func getListOfTransports() string {
	allTransports := strings.Join(transports.ListNames(), ",")
	return strings.Replace(allTransports, ",tarball", "", 1)
}
