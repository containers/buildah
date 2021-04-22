package main

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/containers/buildah/pkg/blobcache"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/libimage"
	libimageTypes "github.com/containers/common/libimage/types"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	// We can feed many flags directly to the options of libmage.
	libimage.PullOptions

	// Other flags need some massaging and validation.
	blobCache      string
	pullPolicy     string
	decryptionKeys []string
	tlsVerify      bool
	quiet          bool
}

func init() {
	var (
		opts pullOptions

		pullDescription = `  Pulls an image from a registry and stores it locally.
  An image can be pulled using its tag or digest. If a tag is not
  specified, the image with the 'latest' tag (if it exists) is pulled.`
	)

	pullCommand := &cobra.Command{
		Use:   "pull",
		Short: "Pull an image from the specified location",
		Long:  pullDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pullCmd(cmd, args, &opts)
		},
		Example: `buildah pull imagename
  buildah pull docker-daemon:imagename:imagetag
  buildah pull myregistry/myrepository/imagename:imagetag`,
	}
	pullCommand.SetUsageTemplate(UsageTemplate())

	flags := pullCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolVarP(&opts.AllTags, "all-tags", "a", false, "download all tagged images in the repository")
	flags.StringVar(&opts.AuthFilePath, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.blobCache, "blob-cache", "", "store copies of pulled image blobs in the specified directory")
	flags.StringVar(&opts.CertDirPath, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.Credentials, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVar(&opts.pullPolicy, "policy", "missing", "missing, always, or never.")
	flags.BoolVarP(&opts.RemoveSignatures, "remove-signatures", "", false, "don't copy signatures when pulling image")
	flags.StringVar(&opts.SignaturePolicyPath, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	flags.StringSliceVar(&opts.decryptionKeys, "decryption-key", nil, "key needed to decrypt the image")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pulling images")
	flags.StringVar(&opts.OS, "os", runtime.GOOS, "prefer `OS` instead of the running OS for choosing images")
	flags.StringVar(&opts.Architecture, "arch", runtime.GOARCH, "prefer `ARCH` instead of the architecture of the machine for choosing images")
	flags.StringVar(&opts.Variant, "variant", "", "override the `variant` of the specified image")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}

	rootCmd.AddCommand(pullCommand)
}

func pullCmd(c *cobra.Command, args []string, options *pullOptions) error {
	var err error
	if len(args) == 0 {
		return errors.Errorf("an image name must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := auth.CheckAuthFile(options.AuthFilePath); err != nil {
		return err
	}

	options.OciDecryptConfig, err = getDecryptConfig(options.decryptionKeys)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain decrypt config")
	}

	options.Writer = os.Stderr
	if options.quiet {
		options.Writer = nil
	}

	if options.blobCache != "" {
		//		options.SourceLookupReferenceFunc = blobcache.CacheLookupReferenceFunc(options.blobCache, types.PreserveOriginal)
		options.DestinationLookupReferenceFunc = blobcache.CacheLookupReferenceFunc(options.blobCache, types.PreserveOriginal)
	}

	pullPolicy, err := libimageTypes.ParsePullPolicy(options.pullPolicy)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	pulledImages, err := runtime.Pull(context.Background(), args[0], pullPolicy, &options.PullOptions)
	if err != nil {
		return err
	}

	for _, pulledImage := range pulledImages {
		fmt.Printf("%s\n", pulledImage.ID())
	}

	return nil
}
