package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/imagebuildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/image/manifest"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	multierror "github.com/hashicorp/go-multierror"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type pushResults struct {
	allTags            bool
	authfile           string
	blobCache          string
	certDir            string
	creds              string
	disableCompression bool
	format             string
	quiet              bool
	signaturePolicy    string
	tlsVerify          bool
}

func init() {
	var (
		opts            pushResults
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
	flags.BoolVarP(&opts.allTags, "all-tags", "a", false, "push all tagged images to the repository")
	flags.StringVar(&opts.authfile, "authfile", "", "path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json")
	flags.StringVar(&opts.blobCache, "blob-cache", "", "assume image blobs in the specified directory will be available for pushing")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.BoolVarP(&opts.disableCompression, "disable-compression", "D", false, "don't compress layers")
	flags.StringVarP(&opts.format, "format", "f", "", "manifest type (oci, v2s1, or v2s2) to use when saving image using the 'dir:' transport (default is manifest type of source)")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pushing images")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	flags.MarkHidden("signature-policy")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry")

	flags.MarkHidden("blob-cache")

	rootCmd.AddCommand(pushCommand)
}

func pushCmd(c *cobra.Command, args []string, iopts pushResults) error {
	var src, destSpec string

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
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
	default:
		return errors.New("Only two arguments are necessary to push: source and destination")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	// Map of image tags found for the specified image.
	// tags stored in the key
	var imageTags []string
	if iopts.allTags {
		if err := getAllTags(src, &imageTags, store); err != nil {
			return err
		}
		logrus.Debugf("For the image: [%s] found tags: %v", src, imageTags)
	}

	var errs *multierror.Error
	imageName := src
	destTarget := destSpec
	compress := imagebuildah.Gzip
	if iopts.disableCompression {
		compress = imagebuildah.Uncompressed
	}

	if len(imageTags) > 1 {
		for _, tagKey := range imageTags {
			src = imageName + ":" + tagKey
			destSpec = destTarget + ":" + tagKey
			if err = pushImage(c, src, destSpec, store, compress, iopts); err != nil {
				errs = multierror.Append(errs, err)
			}
		}
	} else {
		if err = pushImage(c, src, destSpec, store, compress, iopts); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	return errs.ErrorOrNil()
}

func pushImage(c *cobra.Command, src string, destSpec string, store storage.Store, compress archive.Compression, iopts pushResults) error {
	dest, err := alltransports.ParseImageName(destSpec)
	// add the docker:// transport to see if they neglected it.
	if err != nil {
		destTransport := strings.Split(destSpec, ":")[0]
		if t := transports.Get(destTransport); t != nil {
			return err
		}

		if strings.Contains(destSpec, "://") {
			return err
		}

		destSpec = "docker://" + destSpec
		dest2, err2 := alltransports.ParseImageName(destSpec)
		if err2 != nil {
			return err
		}
		dest = dest2
		logrus.Debugf("Assuming docker:// as the transport method for DESTINATION: %s", destSpec)
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
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
			return fmt.Errorf("unknown format %q. Choose on of the supported formats: 'oci', 'v2s1', or 'v2s2'", iopts.format)
		}
	}

	options := buildah.PushOptions{
		Compression:         compress,
		ManifestType:        manifestType,
		SignaturePolicyPath: iopts.signaturePolicy,
		Store:               store,
		SystemContext:       systemContext,
		BlobDirectory:       iopts.blobCache,
	}
	if !iopts.quiet {
		options.ReportWriter = os.Stderr
	}

	ref, digest, err := buildah.Push(getContext(), src, dest, options)
	if err != nil {
		return util.GetFailureCause(err, errors.Wrapf(err, "error pushing image %q to %q", src, destSpec))
	}
	if ref != nil {
		logrus.Debugf("pushed image %q with digest %s", ref, digest.String())
	} else {
		logrus.Debugf("pushed image with digest %s", digest.String())
	}

	return nil
}

// getListOfTransports gets the transports supported from the image library
// and strips of the "tarball" transport from the string of transports returned
func getListOfTransports() string {
	allTransports := strings.Join(transports.ListNames(), ",")
	return strings.Replace(allTransports, ",tarball", "", 1)
}

// getAllTags gets all of the locally tagged images and adds them
// to the passed in slice.
func getAllTags(imageName string, imageTags *[]string, store storage.Store) error {

	imageName = "/" + imageName
	images, err := store.Images()
	if err != nil {
		return errors.Wrapf(err, "error reading images")
	}
	for _, image := range images {

		if len(image.Names) < 1 {
			continue
		}
		// Name should look like: "docker.io/library/alpine:latest"
		for _, name := range image.Names {
			splitName := strings.Split(name, ":")
			if !strings.HasSuffix(splitName[0], imageName) {
				break
			}
			if len(splitName) > 1 {
				*imageTags = append(*imageTags, splitName[1])
			}
		}

	}
	return nil
}
