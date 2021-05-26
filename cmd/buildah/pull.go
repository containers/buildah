package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	allTags          bool
	authfile         string
	blobCache        string
	certDir          string
	creds            string
	signaturePolicy  string
	quiet            bool
	removeSignatures bool
	tlsVerify        bool
	decryptionKeys   []string
	pullPolicy       string
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
			return pullCmd(cmd, args, opts)
		},
		Example: `buildah pull imagename
  buildah pull docker-daemon:imagename:imagetag
  buildah pull myregistry/myrepository/imagename:imagetag`,
	}
	pullCommand.SetUsageTemplate(UsageTemplate())

	flags := pullCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolVarP(&opts.allTags, "all-tags", "a", false, "download all tagged images in the repository")
	flags.StringVar(&opts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.blobCache, "blob-cache", "", "store copies of pulled image blobs in the specified directory")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVar(&opts.pullPolicy, "policy", "missing", "missing, always, or never.")
	flags.BoolVarP(&opts.removeSignatures, "remove-signatures", "", false, "don't copy signatures when pulling image")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	flags.StringSliceVar(&opts.decryptionKeys, "decryption-key", nil, "key needed to decrypt the image")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pulling images")
	flags.String("os", runtime.GOOS, "prefer `OS` instead of the running OS for choosing images")
	flags.String("arch", runtime.GOARCH, "prefer `ARCH` instead of the architecture of the machine for choosing images")
	flags.String("variant", "", "override the `variant` of the specified image")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}

	rootCmd.AddCommand(pullCommand)
}

func pullCmd(c *cobra.Command, args []string, iopts pullOptions) error {
	if len(args) == 0 {
		return errors.Errorf("an image name must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if c.Flag("authfile").Changed {
		if err := auth.CheckAuthFile(iopts.authfile); err != nil {
			return err
		}
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	decConfig, err := getDecryptConfig(iopts.decryptionKeys)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain decrypt config")
	}

	policy, ok := define.PolicyMap[iopts.pullPolicy]
	if !ok {
		return fmt.Errorf("unsupported pull policy %q", iopts.pullPolicy)
	}
	options := buildah.PullOptions{
		SignaturePolicyPath: iopts.signaturePolicy,
		Store:               store,
		SystemContext:       systemContext,
		BlobDirectory:       iopts.blobCache,
		AllTags:             iopts.allTags,
		ReportWriter:        os.Stderr,
		RemoveSignatures:    iopts.removeSignatures,
		MaxRetries:          maxPullPushRetries,
		RetryDelay:          pullPushRetryDelay,
		OciDecryptConfig:    decConfig,
		PullPolicy:          policy,
	}

	if iopts.quiet {
		options.ReportWriter = nil // Turns off logging output
	}

	id, err := buildah.Pull(getContext(), args[0], options)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", id)
	return nil
}
