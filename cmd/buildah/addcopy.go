package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/storage"
	"github.com/openshift/imagebuilder"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type addCopyResults struct {
	addHistory       bool
	chmod            string
	chown            string
	quiet            bool
	ignoreFile       string
	contextdir       string
	from             string
	blobCache        string
	decryptionKeys   []string
	removeSignatures bool
	signaturePolicy  string
	authfile         string
	creds            string
	tlsVerify        bool
	certDir          string
}

func createCommand(addCopy string, desc string, short string, opts *addCopyResults) *cobra.Command {
	return &cobra.Command{
		Use:   addCopy,
		Short: short,
		Long:  desc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addAndCopyCmd(cmd, args, strings.ToUpper(addCopy), *opts)
		},
		Example: `buildah ` + addCopy + ` containerID '/myapp/app.conf'
  buildah ` + addCopy + ` containerID '/myapp/app.conf' '/myapp/app.conf'`,
		Args: cobra.MinimumNArgs(1),
	}
}

func applyFlagVars(flags *pflag.FlagSet, opts *addCopyResults) {
	flags.SetInterspersed(false)
	flags.BoolVar(&opts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	flags.StringVar(&opts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	if err := flags.MarkHidden("authfile"); err != nil {
		panic(fmt.Sprintf("error marking authfile as hidden: %v", err))
	}
	flags.StringVar(&opts.blobCache, "blob-cache", "", "store copies of pulled image blobs in the specified directory")
	if err := flags.MarkHidden("blob-cache"); err != nil {
		panic(fmt.Sprintf("error marking blob-cache as hidden: %v", err))
	}
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access registries")
	if err := flags.MarkHidden("cert-dir"); err != nil {
		panic(fmt.Sprintf("error marking cert-dir as hidden: %v", err))
	}
	flags.StringVar(&opts.chown, "chown", "", "set the user and group ownership of the destination content")
	flags.StringVar(&opts.chmod, "chmod", "", "set the access permissions of the destination content")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing registries when pulling images")
	if err := flags.MarkHidden("creds"); err != nil {
		panic(fmt.Sprintf("error marking creds as hidden: %v", err))
	}
	flags.StringVar(&opts.from, "from", "", "use the specified container's or image's root directory as the source root directory")
	flags.StringSliceVar(&opts.decryptionKeys, "decryption-key", nil, "key needed to decrypt a pulled image")
	if err := flags.MarkHidden("decryption-key"); err != nil {
		panic(fmt.Sprintf("error marking decryption-key as hidden: %v", err))
	}
	flags.StringVar(&opts.ignoreFile, "ignorefile", "", "path to .containerignore file")
	flags.StringVar(&opts.contextdir, "contextdir", "", "context directory path")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output a digest of the newly-added/copied content")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing registries when pulling images. TLS verification cannot be used when talking to an insecure registry.")
	if err := flags.MarkHidden("tls-verify"); err != nil {
		panic(fmt.Sprintf("error marking tls-verify as hidden: %v", err))
	}
	flags.BoolVarP(&opts.removeSignatures, "remove-signatures", "", false, "don't copy signatures when pulling image")
	if err := flags.MarkHidden("remove-signatures"); err != nil {
		panic(fmt.Sprintf("error marking remove-signatures as hidden: %v", err))
	}
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
}

func init() {
	var (
		addDescription  = "\n  Adds the contents of a file, URL, or directory to a container's working\n  directory.  If a local file appears to be an archive, its contents are\n  extracted and added instead of the archive file itself."
		copyDescription = "\n  Copies the contents of a file, URL, or directory into a container's working\n  directory."
		shortAdd        = "Add content to the container"
		shortCopy       = "Copy content into the container"
		addOpts         addCopyResults
		copyOpts        addCopyResults
	)
	addCommand := createCommand("add", addDescription, shortAdd, &addOpts)
	addCommand.SetUsageTemplate(UsageTemplate())

	copyCommand := createCommand("copy", copyDescription, shortCopy, &copyOpts)
	copyCommand.SetUsageTemplate(UsageTemplate())

	addFlags := addCommand.Flags()
	applyFlagVars(addFlags, &addOpts)

	copyFlags := copyCommand.Flags()
	applyFlagVars(copyFlags, &copyOpts)

	rootCmd.AddCommand(addCommand)
	rootCmd.AddCommand(copyCommand)
}

func addAndCopyCmd(c *cobra.Command, args []string, verb string, iopts addCopyResults) error {
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := args[0]
	args = Tail(args)
	if len(args) == 0 {
		return errors.Errorf("src must be specified")
	}

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}

	// If list is greater than one, the last item is the destination
	dest := ""
	size := len(args)
	if size > 1 {
		dest = args[size-1]
		args = args[:size-1]
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	var from *buildah.Builder
	unmountFrom := false
	removeFrom := false
	var idMappingOptions *buildah.IDMappingOptions
	contextdir := iopts.contextdir
	if iopts.ignoreFile != "" && contextdir == "" {
		return errors.Errorf("--ignorefile option requires that you specify a context dir using --contextdir")
	}

	if iopts.from != "" {
		if from, err = openBuilder(getContext(), store, iopts.from); err != nil && errors.Cause(err) == storage.ErrContainerUnknown {
			systemContext, err2 := parse.SystemContextFromOptions(c)
			if err2 != nil {
				return errors.Wrap(err2, "error building system context")
			}
			decConfig, err2 := getDecryptConfig(iopts.decryptionKeys)
			if err2 != nil {
				return errors.Wrapf(err2, "unable to obtain decrypt config")
			}
			options := buildah.BuilderOptions{
				FromImage:           iopts.from,
				BlobDirectory:       iopts.blobCache,
				SignaturePolicyPath: iopts.signaturePolicy,
				SystemContext:       systemContext,
				MaxPullRetries:      maxPullPushRetries,
				PullRetryDelay:      pullPushRetryDelay,
				OciDecryptConfig:    decConfig,
			}
			if !iopts.quiet {
				options.ReportWriter = os.Stderr
			}
			if from, err = buildah.NewBuilder(getContext(), store, options); err != nil {
				return errors.Wrapf(err, "no container named %q, error copying content from image %q", iopts.from, iopts.from)
			}
			removeFrom = true
			defer func() {
				if !removeFrom {
					return
				}
				if err := from.Delete(); err != nil {
					logrus.Errorf("error deleting %q temporary working container %q", iopts.from, from.Container)
				}
			}()
		}
		if err != nil {
			return errors.Wrapf(err, "error reading build container %q", iopts.from)
		}
		fromMountPoint, err := from.Mount(from.MountLabel)
		if err != nil {
			return errors.Wrapf(err, "error mounting %q container %q", iopts.from, from.Container)
		}
		unmountFrom = true
		defer func() {
			if !unmountFrom {
				return
			}
			if err := from.Unmount(); err != nil {
				logrus.Errorf("error unmounting %q container %q", iopts.from, from.Container)
			}
			if err := from.Save(); err != nil {
				logrus.Errorf("error saving information about %q container %q", iopts.from, from.Container)
			}
		}()
		idMappingOptions = &from.IDMappingOptions
		contextdir = filepath.Join(fromMountPoint, iopts.contextdir)
		for i := range args {
			args[i] = filepath.Join(fromMountPoint, args[i])
		}
	}

	builder, err := openBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	builder.ContentDigester.Restart()

	options := buildah.AddAndCopyOptions{
		Chmod:            iopts.chmod,
		Chown:            iopts.chown,
		ContextDir:       contextdir,
		IDMappingOptions: idMappingOptions,
	}
	if iopts.contextdir != "" {
		var excludes []string
		if iopts.ignoreFile != "" {
			excludes, err = parseIgnore(iopts.ignoreFile)
		} else {
			excludes, err = imagebuilder.ParseDockerignore(contextdir)
		}
		if err != nil {
			return err
		}
		options.Excludes = excludes
	}

	extractLocalArchives := verb == "ADD"
	err = builder.Add(dest, extractLocalArchives, options, args...)
	if err != nil {
		return errors.Wrapf(err, "error adding content to container %q", builder.Container)
	}
	if unmountFrom {
		if err := from.Unmount(); err != nil {
			return errors.Wrapf(err, "error unmounting %q container %q", iopts.from, from.Container)
		}
		if err := from.Save(); err != nil {
			return errors.Wrapf(err, "error saving information about %q container %q", iopts.from, from.Container)
		}
		unmountFrom = false
	}
	if removeFrom {
		if err := from.Delete(); err != nil {
			return errors.Wrapf(err, "error deleting %q temporary working container %q", iopts.from, from.Container)
		}
		removeFrom = false
	}

	contentType, digest := builder.ContentDigester.Digest()
	if !iopts.quiet {
		fmt.Printf("%s\n", digest.Hex())
	}
	if contentType != "" {
		contentType = contentType + ":"
	}
	conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) %s %s%s", verb, contentType, digest.Hex())
	return builder.Save()
}

func parseIgnore(ignoreFile string) ([]string, error) {
	var excludes []string
	ignore, err := ioutil.ReadFile(ignoreFile)
	if err != nil {
		return excludes, err
	}
	for _, e := range strings.Split(string(ignore), "\n") {
		if len(e) == 0 || e[0] == '#' {
			continue
		}
		excludes = append(excludes, e)
	}
	return excludes, nil
}
