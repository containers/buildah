package main

import (
	"os"
	"strings"
	"time"

	"github.com/containers/image/storage"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	commitFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "disable-compression, D",
			Usage: "don't compress layers",
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
		cli.BoolTFlag{
			Name:  "tls-verify",
			Usage: "Require HTTPS and verify certificates when accessing the registry",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.StringFlag{
			Name:  "format, f",
			Usage: "`format` of the image manifest and metadata",
		},
		cli.StringFlag{
			Name:   "reference-time",
			Usage:  "set the timestamp on the image to match the named `file`",
			Hidden: true,
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when writing images",
		},
		cli.BoolFlag{
			Name:  "rm",
			Usage: "remove the container and its content after committing it to an image. Default leaves the container and its content in place.",
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
		return errors.Errorf("container ID must be specified")
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
	format := "oci"
	if c.IsSet("format") {
		format = c.String("format")
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
	if strings.HasPrefix(strings.ToLower(format), "oci") {
		format = buildah.OCIv1ImageManifest
	} else if strings.HasPrefix(strings.ToLower(format), "docker") {
		format = buildah.Dockerv2ImageManifest
	} else {
		return errors.Errorf("unrecognized image type %q", format)
	}
	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	dest, err := alltransports.ParseImageName(image)
	if err != nil {
		dest2, err2 := storage.Transport.ParseStoreReference(store, image)
		if err2 != nil {
			return errors.Wrapf(err, "error parsing target image name %q", image)
		}
		dest = dest2
	}

	systemContext, err := systemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	options := buildah.CommitOptions{
		PreferredManifestType: format,
		Compression:           compress,
		SignaturePolicyPath:   signaturePolicy,
		HistoryTimestamp:      &timestamp,
		SystemContext:         systemContext,
	}
	if !quiet {
		options.ReportWriter = os.Stderr
	}
	err = builder.Commit(dest, options)
	if err != nil {
		return errors.Wrapf(err, "error committing container %q to %q", builder.Container, image)
	}

	if c.Bool("rm") {
		return builder.Delete()
	}
	return nil
}
