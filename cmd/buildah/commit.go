package main

import (
	"fmt"
	"os"
	"time"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/image/storage"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	commitFlags = []cli.Flag{
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
			Usage: "`format` of the image manifest and metadata",
			Value: defaultFormat(),
		},
		cli.StringFlag{
			Name:  "iidfile",
			Usage: "Write the image ID to the file",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when writing images",
		},
		cli.StringFlag{
			Name:   "reference-time",
			Usage:  "set the timestamp on the image to match the named `file`",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:  "rm",
			Usage: "remove the container and its content after committing it to an image. Default leaves the container and its content in place.",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.BoolFlag{
			Name:  "squash",
			Usage: "produce an image with only one layer",
		},
		cli.BoolTFlag{
			Name:  "tls-verify",
			Usage: "Require HTTPS and verify certificates when accessing the registry",
		},
	}
	commitDescription = "Writes a new image using the container's read-write layer and, if it is based\n   on an image, the layers of that image"
	commitCommand     = cli.Command{
		Name:                   "commit",
		Usage:                  "Create an image from a working container",
		Description:            commitDescription,
		Flags:                  sortFlags(commitFlags),
		Action:                 commitCmd,
		ArgsUsage:              "CONTAINER-NAME-OR-ID IMAGE",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func commitCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	name := args[0]
	args = args.Tail()
	if len(args) == 0 {
		return errors.Errorf("an image name must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	image := args[0]
	if err := parse.ValidateFlags(c, commitFlags); err != nil {
		return err
	}

	compress := archive.Gzip
	if c.Bool("disable-compression") {
		compress = archive.Uncompressed
	}
	timestamp := time.Now().UTC()
	if c.IsSet("reference-time") {
		referenceFile := c.String("reference-time")
		finfo, err := os.Stat(referenceFile)
		if err != nil {
			return errors.Wrapf(err, "error reading timestamp of file %q", referenceFile)
		}
		timestamp = finfo.ModTime().UTC()
	}

	format, err := getFormat(c)
	if err != nil {
		return err
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	ctx := getContext()

	builder, err := openBuilder(ctx, store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	dest, err := alltransports.ParseImageName(image)
	if err != nil {
		candidates, _, err := util.ResolveName(image, "", systemContext, store)
		if err != nil {
			return errors.Wrapf(err, "error parsing target image name %q", image)
		}
		if len(candidates) == 0 {
			return errors.Errorf("error parsing target image name %q", image)
		}
		dest2, err2 := storage.Transport.ParseStoreReference(store, candidates[0])
		if err2 != nil {
			return errors.Wrapf(err, "error parsing target image name %q", image)
		}
		dest = dest2
	}

	options := buildah.CommitOptions{
		PreferredManifestType: format,
		Compression:           compress,
		SignaturePolicyPath:   c.String("signature-policy"),
		HistoryTimestamp:      &timestamp,
		SystemContext:         systemContext,
		IIDFile:               c.String("iidfile"),
		Squash:                c.Bool("squash"),
	}
	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}
	id, ref, _, err := builder.Commit(ctx, dest, options)
	if err != nil {
		return util.GetFailureCause(err, errors.Wrapf(err, "error committing container %q to %q", builder.Container, image))
	}
	if ref != nil {
		logrus.Debugf("wrote image %s with ID %s", ref, id)
	} else {
		logrus.Debugf("wrote image with ID %s", id)
	}
	if options.IIDFile == "" && id != "" {
		fmt.Printf("%s\n", id)
	}

	if c.Bool("rm") {
		return builder.Delete()
	}
	return nil
}
