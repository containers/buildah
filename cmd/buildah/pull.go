package main

import (
	"fmt"
	"os"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pullResults struct {
	allTags         bool
	authfile        string
	blobCache       string
	certDir         string
	creds           string
	signaturePolicy string
	quiet           bool
	tlsVerify       bool
}

func init() {
	var (
		opts pullResults

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
	flags.StringVar(&opts.authfile, "authfile", buildahcli.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.blobCache, "blob-cache", "", "store copies of pulled image blobs in the specified directory")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pulling images")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry")
	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}

	rootCmd.AddCommand(pullCommand)
}

func pullCmd(c *cobra.Command, args []string, iopts pullResults) error {
	if len(args) == 0 {
		return errors.Errorf("an image name must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := buildah.PullOptions{
		SignaturePolicyPath: iopts.signaturePolicy,
		Store:               store,
		SystemContext:       systemContext,
		BlobDirectory:       iopts.blobCache,
		AllTags:             iopts.allTags,
		ReportWriter:        os.Stderr,
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
