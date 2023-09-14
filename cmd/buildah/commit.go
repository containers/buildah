package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/completion"
	"github.com/containers/image/v5/pkg/shortnames"
	storageTransport "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type commitInputOptions struct {
	authfile           string
	omitHistory        bool
	blobCache          string
	certDir            string
	creds              string
	cwOptions          string
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
	identityLabel      bool
	encryptionKeys     []string
	encryptLayers      []int
	unsetenvs          []string
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
	commitListFlagSet(commitCommand, &opts)
	rootCmd.AddCommand(commitCommand)

}

func commitListFlagSet(cmd *cobra.Command, opts *commitInputOptions) {
	flags := cmd.Flags()
	flags.SetInterspersed(false)

	flags.StringVar(&opts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	_ = cmd.RegisterFlagCompletionFunc("authfile", completion.AutocompleteDefault)
	flags.StringVar(&opts.blobCache, "blob-cache", "", "assume image blobs in the specified directory will be available for pushing")
	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}
	flags.StringSliceVar(&opts.encryptionKeys, "encryption-key", nil, "key with the encryption protocol to use needed to encrypt the image (e.g. jwe:/path/to/key.pem)")
	_ = cmd.RegisterFlagCompletionFunc("encryption-key", completion.AutocompleteDefault)
	flags.IntSliceVar(&opts.encryptLayers, "encrypt-layer", nil, "layers to encrypt, 0-indexed layer indices with support for negative indexing (e.g. 0 is the first layer, -1 is the last layer). If not defined, will encrypt all layers if encryption-key flag is specified")
	_ = cmd.RegisterFlagCompletionFunc("encryption-key", completion.AutocompleteNone)

	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	_ = cmd.RegisterFlagCompletionFunc("cert-dir", completion.AutocompleteDefault)
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	_ = cmd.RegisterFlagCompletionFunc("creds", completion.AutocompleteNone)
	flags.StringVar(&opts.cwOptions, "cw", "", "confidential workload `options`")
	flags.BoolVarP(&opts.disableCompression, "disable-compression", "D", true, "don't compress layers")
	flags.StringVarP(&opts.format, "format", "f", defaultFormat(), "`format` of the image manifest and metadata")
	_ = cmd.RegisterFlagCompletionFunc("format", completion.AutocompleteNone)
	flags.StringVar(&opts.manifest, "manifest", "", "adds created image to the specified manifest list. Creates manifest list if it does not exist")
	_ = cmd.RegisterFlagCompletionFunc("manifest", completion.AutocompleteNone)
	flags.StringVar(&opts.iidfile, "iidfile", "", "write the image ID to the file")
	_ = cmd.RegisterFlagCompletionFunc("iidfile", completion.AutocompleteDefault)
	flags.BoolVar(&opts.omitTimestamp, "omit-timestamp", false, "set created timestamp to epoch 0 to allow for deterministic builds")
	flags.Int64Var(&opts.timestamp, "timestamp", 0, "set created timestamp to epoch seconds to allow for deterministic builds, defaults to current time")
	_ = cmd.RegisterFlagCompletionFunc("timestamp", completion.AutocompleteNone)
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when writing images")
	flags.StringVar(&opts.referenceTime, "reference-time", "", "set the timestamp on the image to match the named `file`")
	_ = cmd.RegisterFlagCompletionFunc("reference-time", completion.AutocompleteNone)
	flags.StringVar(&opts.signBy, "sign-by", "", "sign the image using a GPG key with the specified `FINGERPRINT`")
	_ = cmd.RegisterFlagCompletionFunc("sign-by", completion.AutocompleteNone)
	if err := flags.MarkHidden("omit-timestamp"); err != nil {
		panic(fmt.Sprintf("error marking omit-timestamp as hidden: %v", err))
	}
	if err := flags.MarkHidden("reference-time"); err != nil {
		panic(fmt.Sprintf("error marking reference-time as hidden: %v", err))
	}

	flags.BoolVar(&opts.omitHistory, "omit-history", false, "omit build history information from the built image (default false)")
	flags.BoolVar(&opts.identityLabel, "identity-label", true, "add default builder label (default true)")
	flags.BoolVar(&opts.rm, "rm", false, "remove the container and its content after committing it to an image. Default leaves the container and its content in place.")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	_ = cmd.RegisterFlagCompletionFunc("signature-policy", completion.AutocompleteDefault)

	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}

	flags.BoolVar(&opts.squash, "squash", false, "produce an image with only one layer")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")

	flags.StringSliceVar(&opts.unsetenvs, "unsetenv", nil, "unset env from final image")
	_ = cmd.RegisterFlagCompletionFunc("unsetenv", completion.AutocompleteNone)
}

func commitCmd(c *cobra.Command, args []string, iopts commitInputOptions) error {
	var dest types.ImageReference
	if len(args) == 0 {
		return errors.New("container ID must be specified")
	}
	if err := cli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if err := auth.CheckAuthFile(iopts.authfile); err != nil {
		return err
	}

	name := args[0]
	args = Tail(args)
	if len(args) > 1 {
		return errors.New("too many arguments specified")
	}
	image := ""
	if len(args) > 0 {
		image = args[0]
	}
	compress := define.Gzip
	if iopts.disableCompression {
		compress = define.Uncompressed
	}

	format, err := cli.GetFormat(iopts.format)
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
		return fmt.Errorf("reading build container %q: %w", name, err)
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return fmt.Errorf("building system context: %w", err)
	}

	// If the user specified an image, we may need to massage it a bit if
	// no transport is specified.
	if image != "" {
		if dest, err = alltransports.ParseImageName(image); err != nil {
			candidates, err2 := shortnames.ResolveLocally(systemContext, image)
			if err2 != nil {
				return err2
			}
			if len(candidates) == 0 {
				return fmt.Errorf("parsing target image name %q", image)
			}
			dest2, err2 := storageTransport.Transport.ParseStoreReference(store, candidates[0].String())
			if err2 != nil {
				return fmt.Errorf("parsing target image name %q: %w", image, err)
			}
			dest = dest2
		}
	}

	// Add builder identity information.
	if iopts.identityLabel {
		builder.SetLabel(buildah.BuilderIdentityAnnotation, define.Version)
	}

	encConfig, encLayers, err := cli.EncryptConfig(iopts.encryptionKeys, iopts.encryptLayers)
	if err != nil {
		return fmt.Errorf("unable to obtain encryption config: %w", err)
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
		OmitHistory:           iopts.omitHistory,
		SignBy:                iopts.signBy,
		OciEncryptConfig:      encConfig,
		OciEncryptLayers:      encLayers,
		UnsetEnvs:             iopts.unsetenvs,
	}
	exclusiveFlags := 0
	if c.Flag("reference-time").Changed {
		exclusiveFlags++
		referenceFile := iopts.referenceTime
		finfo, err := os.Stat(referenceFile)
		if err != nil {
			return fmt.Errorf("reading timestamp of file %q: %w", referenceFile, err)
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

	if iopts.cwOptions != "" {
		confidentialWorkloadOptions, err := parse.GetConfidentialWorkloadOptions(iopts.cwOptions)
		if err != nil {
			return fmt.Errorf("parsing --cw arguments: %w", err)
		}
		options.ConfidentialWorkloadOptions = confidentialWorkloadOptions
	}

	if exclusiveFlags > 1 {
		return errors.New("can not use more then one timestamp option at at time")
	}

	if !iopts.quiet {
		options.ReportWriter = os.Stderr
	}
	id, ref, _, err := builder.Commit(ctx, dest, options)
	if err != nil {
		return util.GetFailureCause(err, fmt.Errorf("committing container %q to %q: %w", builder.Container, image, err))
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
