package main

import (
	"fmt"
	"os"
	"time"

	"github.com/containers/image/storage"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	importFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "change, c",
			Usage: "Apply specified Dockerfile instructions while importing the image",
		},
		cli.StringFlag{
			Name:  "message, m",
			Usage: "Set commit message for imported image",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
	}
	importCommand = cli.Command{
		Name:        "import",
		Usage:       "Create an empty filesystem image and import the contents of the tar archive.",
		Description: `Supported archive formats: ().tar, .tar.gz, .tgz, .bzip, .tar.xz, .txz), then optionally tag it.`,
		Flags:       importFlags,
		Action:      importCmd,
		ArgsUsage:   "TARARCHIVE",
	}
)

func importCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("tar archive file must be specified")
	}
	tarFile := args[0]

	tag := ""
	if len(args) == 2 {
		tag = args[1]
	}
	if len(args) > 2 {
		return errors.Errorf("too many arguments specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := buildah.BuilderOptions{
		FromImage: "scratch",
	}

	builder, err := buildah.NewBuilder(store, options)
	if err != nil {
		return err
	}

	mountPoint, err := builder.Mount("")
	if err != nil {
		return errors.Wrapf(err, "error mounting container %q", builder.Container)
	}
	defer func() {
		if err := builder.Unmount(); err != nil {
			fmt.Printf("Failed to umount %q: %v\n", builder.Container, err)
		}
		if err := builder.Delete(); err != nil {
			fmt.Printf("Failed to delete %q: %v\n", builder.Container, err)
		}
	}()

	ioTar, err := os.Open(tarFile)
	if err != nil {
		return errors.Wrapf(err, "error opening tar file %q", tarFile)
	}
	defer ioTar.Close()
	if err := chrootarchive.Untar(ioTar, mountPoint, nil); err != nil {
		return errors.Wrapf(err, "error reading tar file %q", tarFile)
	}

	signaturePolicy := ""
	if c.IsSet("signature-policy") {
		signaturePolicy = c.String("signature-policy")
	}

	timestamp := time.Now().UTC()
	coptions := buildah.CommitOptions{
		PreferredManifestType: buildah.OCIv1ImageManifest,
		HistoryTimestamp:      &timestamp,
		SignaturePolicyPath:   signaturePolicy,
	}

	if c.IsSet("message") {
		builder.SetMessage(c.String("message"))
	}

	for _, c := range c.StringSlice("change") {
		if err = builder.ChangeConfig(c); err != nil {
			return err
		}
	}
	ref, err := storage.Transport.ParseStoreReference(store, tag)
	if err != nil {
		return errors.Wrapf(err, "error parsing StoreReference %q", tag)
	}

	err = builder.Commit(ref, coptions)
	if err != nil {
		return errors.Wrapf(err, "error committing container %q to %q", builder.Container, tag)
	}
	return err
}
