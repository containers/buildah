package main

import (
	"fmt"
	"os"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/pkg/shortnames"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type commitInputOptions struct {
	authfile           string
	blobCache          string
	certDir            string
	creds              string
	disableCompression bool
	format             string
	iidfile            string
	manifest           string
	omitTimestamp      bool
	timestamp          int64
	quiet              bool
	referenceTime      string
	rm                 bool
	signaturePolicy    string
	signBy             string
	squash             bool
	tlsVerify          bool
	encryptionKeys     []string
	encryptLayers      []int
}

func init() {
	var (
		opts              commitInputOptions
		commitDescription = "\n  Writes a new image using the container's read-write layer and, if it is based\n  on an image, the layers of that image."
	)
	commitCommand := &cobra.Command{
		Use:   "commit",
		Short: "Create an image from a working container",
		Long:  commitDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commitCmd(cmd, args, opts)
		},
		Example: `buildah commit containerID
  buildah commit containerID newImageName
  buildah commit containerID docker://localhost:5000/imageId`,
	}
	commitCommand.SetUsageTemplate(UsageTemplate())
	flags := commitCommand.Flags()
	flags.SetInterspersed(false)

	flags.StringVar(&opts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.blobCache, "blob-cache", "", "assume image blobs in the specified directory will be available for pushing")
	flags.StringSliceVar(&opts.encryptionKeys, "encryption-key", nil, "key with the encryption protocol to use needed to encrypt the image (e.g. jwe:/path/to/key.pem)")
	flags.IntSliceVar(&opts.encryptLayers, "encrypt-layer", nil, "layers to encrypt, 0-indexed layer indices with support for negative indexing (e.g. 0 is the first layer, -1 is the last layer). If not defined, will encrypt all layers if encryption-key flag is specified")

	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}

	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.BoolVarP(&opts.disableCompression, "disable-compression", "D", true, "don't compress layers")
	flags.StringVarP(&opts.format, "format", "f", defaultFormat(), "`format` of the image manifest and metadata")
	flags.StringVar(&opts.manifest, "manifest", "", "create image with as part of the specified manifest list. Creates manifest if it does not exist")
	flags.StringVar(&opts.iidfile, "iidfile", "", "Write the image ID to the file")
	flags.BoolVar(&opts.omitTimestamp, "omit-timestamp", false, "set created timestamp to epoch 0 to allow for deterministic builds")
	flags.Int64Var(&opts.timestamp, "timestamp", 0, "set created timestamp to epoch seconds to allow for deterministic builds, defaults to current time")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when writing images")
	flags.StringVar(&opts.referenceTime, "reference-time", "", "set the timestamp on the image to match the named `file`")
	flags.StringVar(&opts.signBy, "sign-by", "", "sign the image using a GPG key with the specified `FINGERPRINT`")

	if err := flags.MarkHidden("omit-timestamp"); err != nil {
		panic(fmt.Sprintf("error marking omit-timestamp as hidden: %v", err))
	}
	if err := flags.MarkHidden("reference-time"); err != nil {
		panic(fmt.Sprintf("error marking reference-time as hidden: %v", err))
	}

	flags.BoolVar(&opts.rm, "rm", false, "remove the container and its content after committing it to an image. Default leaves the container and its content in place.")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")

	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}

	flags.BoolVar(&opts.squash, "squash", false, "produce an image with only one layer")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "Require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")

	rootCmd.AddCommand(commitCommand)

}

func commitCmd(c *cobra.Command, args []string, iopts commitInputOptions) error {
	var dest types.ImageReference
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if c.Flag("authfile").Changed {
		if err := auth.CheckAuthFile(iopts.authfile); err != nil {
			return err
		}
	}

	name := args[0]
	args = Tail(args)
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	image := ""
	if len(args) > 0 {
		image = args[0]
	}
	compress := define.Gzip
	if iopts.disableCompression {
		compress = define.Uncompressed
	}

	format, err := getFormat(iopts.format)
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

	// If the user specified an image, we may need to massage it a bit if
	// no transport is specified.
	if image != "" {
		if dest, err = alltransports.ParseImageName(image); err != nil {
			candidates, err := shortnames.ResolveLocally(systemContext, image)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				return errors.Errorf("error parsing target image name %q", image)
			}
			dest2, err2 := storageTransport.Transport.ParseStoreReference(store, candidates[0].String())
			if err2 != nil {
				return errors.Wrapf(err, "error parsing target image name %q", image)
			}
			dest = dest2
		}
	}

	// Add builder identity information.
	builder.SetLabel(buildah.BuilderIdentityAnnotation, define.Version)

	encConfig, encLayers, err := getEncryptConfig(iopts.encryptionKeys, iopts.encryptLayers)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain encryption config")
	}

	options := buildah.CommitOptions{
		PreferredManifestType: format,
		Manifest:              iopts.manifest,
		Compression:           compress,
		SignaturePolicyPath:   iopts.signaturePolicy,
		SystemContext:         systemContext,
		IIDFile:               iopts.iidfile,
		Squash:                iopts.squash,
		BlobDirectory:         iopts.blobCache,
		SignBy:                iopts.signBy,
		OciEncryptConfig:      encConfig,
		OciEncryptLayers:      encLayers,
	}
	exclusiveFlags := 0
	if c.Flag("reference-time").Changed {
		exclusiveFlags++
		referenceFile := iopts.referenceTime
		finfo, err := os.Stat(referenceFile)
		if err != nil {
			return errors.Wrapf(err, "error reading timestamp of file %q", referenceFile)
		}
		timestamp := finfo.ModTime().UTC()
		options.HistoryTimestamp = &timestamp
	}
	if c.Flag("timestamp").Changed {
		exclusiveFlags++
		timestamp := time.Unix(iopts.timestamp, 0).UTC()
		options.HistoryTimestamp = &timestamp
	}
	if iopts.omitTimestamp {
		exclusiveFlags++
		timestamp := time.Unix(0, 0).UTC()
		options.HistoryTimestamp = &timestamp
	}

	if exclusiveFlags > 1 {
		return errors.Errorf("can not use more then one timestamp option at at time")
	}

	if !iopts.quiet {
		options.ReportWriter = os.Stderr
	}
	id, ref, _, err := builder.Commit(ctx, dest, options)
	if err != nil {
		return util.GetFailureCause(err, errors.Wrapf(err, "error committing container %q to %q", builder.Container, image))
	}
	if ref != nil && id != "" {
		logrus.Debugf("wrote image %s with ID %s", ref, id)
	} else if ref != nil {
		logrus.Debugf("wrote image %s", ref)
	} else if id != "" {
		logrus.Debugf("wrote image with ID %s", id)
	} else {
		logrus.Debugf("wrote image")
	}
	if options.IIDFile == "" && id != "" {
		fmt.Printf("%s\n", id)
	}

	if iopts.rm {
		return builder.Delete()
	}
	return nil
}
