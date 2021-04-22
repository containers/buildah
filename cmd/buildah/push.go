package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containers/buildah/pkg/blobcache"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/libimage"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	all                bool
	authfile           string
	blobCache          string
	certDir            string
	creds              string
	digestfile         string
	disableCompression bool
	format             string
	rm                 bool
	quiet              bool
	removeSignatures   bool
	signaturePolicy    string
	signBy             string
	tlsVerify          bool
	encryptionKeys     []string
	encryptLayers      []int
}

// translates the pushOptions into libimage.PushOptions.
func (iopts *pushOptions) toLibimagePushOptions() (*libimage.PushOptions, error) {
	pushOptions := &libimage.PushOptions{}
	pushOptions.PolicyAllowStorage = true
	pushOptions.AuthFilePath = iopts.authfile
	pushOptions.CertDirPath = iopts.certDir
	pushOptions.Credentials = iopts.creds
	pushOptions.RemoveSignatures = iopts.removeSignatures
	pushOptions.SignaturePolicyPath = iopts.signaturePolicy
	pushOptions.SignBy = iopts.signBy

	if iopts.blobCache != "" {
		compress := types.Compress
		if iopts.disableCompression {
			compress = types.PreserveOriginal
		}
		pushOptions.SourceLookupReferenceFunc = blobcache.CacheLookupReferenceFunc(iopts.blobCache, compress)
	}

	var manifestType string
	if iopts.format != "" {
		switch iopts.format {
		case "oci":
			manifestType = imgspecv1.MediaTypeImageManifest
		case "v2s1":
			manifestType = manifest.DockerV2Schema1SignedMediaType
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return nil, errors.Errorf("unknown format %q. Choose on of the supported formats: 'oci', 'v2s1', or 'v2s2'", iopts.format)
		}
	}
	pushOptions.ManifestMIMEType = manifestType

	encConfig, encLayers, err := getEncryptConfig(iopts.encryptionKeys, iopts.encryptLayers)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to obtain encryption config")
	}
	pushOptions.OciEncryptConfig = encConfig
	pushOptions.OciEncryptLayers = encLayers
	pushOptions.InsecureSkipTLSVerify = types.NewOptionalBool(!iopts.tlsVerify)

	if !iopts.quiet {
		pushOptions.Writer = os.Stderr
	}

	return pushOptions, nil
}

func init() {
	var (
		opts            pushOptions
		pushDescription = fmt.Sprintf(`
  Pushes an image to a specified location.

  The Image "DESTINATION" uses a "transport":"details" format. If not specified, will reuse source IMAGE as DESTINATION.

  Supported transports:
  %s

  See buildah-push(1) section "DESTINATION" for the expected format
`, getListOfTransports())
	)

	pushCommand := &cobra.Command{
		Use:   "push",
		Short: "Push an image to a specified destination",
		Long:  pushDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pushCmd(cmd, args, opts)
		},
		Example: `buildah push imageID docker://registry.example.com/repository:tag
  buildah push imageID docker-daemon:image:tagi
  buildah push imageID oci:/path/to/layout:image:tag`,
	}
	pushCommand.SetUsageTemplate(UsageTemplate())

	flags := pushCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolVar(&opts.all, "all", false, "push all of the images referenced by the manifest list")
	flags.StringVar(&opts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.blobCache, "blob-cache", "", "assume image blobs in the specified directory will be available for pushing")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVar(&opts.digestfile, "digestfile", "", "after copying the image, write the digest of the resulting image to the file")
	flags.BoolVarP(&opts.disableCompression, "disable-compression", "D", false, "don't compress layers")
	flags.StringVarP(&opts.format, "format", "f", "", "manifest type (oci, v2s1, or v2s2) to use when saving image using the 'dir:' transport (default is manifest type of source)")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pushing images")
	flags.BoolVar(&opts.rm, "rm", false, "remove the manifest list if push succeeds")
	flags.BoolVarP(&opts.removeSignatures, "remove-signatures", "", false, "don't copy signatures when pushing image")
	flags.StringVar(&opts.signBy, "sign-by", "", "sign the image using a GPG key with the specified `FINGERPRINT`")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	flags.StringSliceVar(&opts.encryptionKeys, "encryption-key", nil, "key with the encryption protocol to use needed to encrypt the image (e.g. jwe:/path/to/key.pem)")
	flags.IntSliceVar(&opts.encryptLayers, "encrypt-layer", nil, "layers to encrypt, 0-indexed layer indices with support for negative indexing (e.g. 0 is the first layer, -1 is the last layer). If not defined, will encrypt all layers if encryption-key flag is specified")

	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}

	rootCmd.AddCommand(pushCommand)
}

func pushCmd(c *cobra.Command, args []string, iopts pushOptions) error {
	var src, destSpec string

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if err := auth.CheckAuthFile(iopts.authfile); err != nil {
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
		if src == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[0])
		}
	default:
		return errors.New("Only two arguments are necessary to push: source and destination")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	pushOptions, err := iopts.toLibimagePushOptions()
	if err != nil {
		return err
	}

	pushedManifestBytes, pushError := runtime.Push(context.Background(), src, destSpec, pushOptions)
	if pushError != nil {
		// TODO: maybe we find a way to handle that transparently in libimage?
		if errors.Cause(pushError) != storage.ErrImageUnknown {
			// Image might be a manifest so attempt a manifest push
			if manifestsErr := manifestPush(systemContext, store, src, destSpec, iopts); manifestsErr == nil {
				return nil
			}
		}
		return pushError
	}

	if iopts.digestfile != "" {
		manifestDigest, err := manifest.Digest(pushedManifestBytes)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(iopts.digestfile, []byte(manifestDigest.String()), 0644); err != nil {
			return err
		}
	}

	return nil
}

func getListOfTransports() string {
	allTransports := strings.Join(transports.ListNames(), ",")
	return strings.Replace(allTransports, ",tarball", "", 1)
}
