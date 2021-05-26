package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/common/libimage"
	"github.com/containers/common/libimage/manifests"
	"github.com/containers/common/pkg/auth"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/hashicorp/go-multierror"
	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type manifestCreateOpts = struct {
	os, arch string
	all      bool
}
type manifestAddOpts = struct {
	authfile, certDir, creds, os, arch, variant, osVersion string
	features, osFeatures, annotations                      []string
	tlsVerify, all                                         bool
}
type manifestRemoveOpts = struct{}
type manifestAnnotateOpts = struct {
	os, arch, variant, osVersion      string
	features, osFeatures, annotations []string
}
type manifestInspectOpts = struct{}

func init() {
	var (
		manifestDescription         = "\n  Creates, modifies, and pushes manifest lists and image indexes."
		manifestCreateDescription   = "\n  Creates manifest lists and image indexes."
		manifestAddDescription      = "\n  Adds an image to a manifest list or image index."
		manifestRemoveDescription   = "\n  Removes an image from a manifest list or image index."
		manifestAnnotateDescription = "\n  Adds or updates information about an entry in a manifest list or image index."
		manifestInspectDescription  = "\n  Display the contents of a manifest list or image index."
		manifestPushDescription     = "\n  Pushes manifest lists and image indexes to registries."
		manifestRmDescription       = "\n  Remove one or more manifest lists from local storage."
		manifestCreateOpts          manifestCreateOpts
		manifestAddOpts             manifestAddOpts
		manifestRemoveOpts          manifestRemoveOpts
		manifestAnnotateOpts        manifestAnnotateOpts
		manifestInspectOpts         manifestInspectOpts
		manifestPushOpts            pushOptions
	)
	manifestCommand := &cobra.Command{
		Use:   "manifest",
		Short: "Manipulate manifest lists and image indexes",
		Long:  manifestDescription,
		Example: `buildah manifest create localhost/list
  buildah manifest add localhost/list localhost/image
  buildah manifest annotate --annotation A=B localhost/list localhost/image
  buildah manifest annotate --annotation A=B localhost/list sha256:entryManifestDigest
  buildah manifest inspect localhost/list
  buildah manifest push localhost/list transport:destination
  buildah manifest remove localhost/list sha256:entryManifestDigest
  buildah manifest rm localhost/list`,
	}
	manifestCommand.SetUsageTemplate(UsageTemplate())
	rootCmd.AddCommand(manifestCommand)

	manifestCreateCommand := &cobra.Command{
		Use:   "create",
		Short: "Create manifest list or image index",
		Long:  manifestCreateDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestCreateCmd(cmd, args, manifestCreateOpts)
		},
		Example: `buildah manifest create mylist:v1.11
  buildah manifest create mylist:v1.11 arch-specific-image-to-add
  buildah manifest create --all mylist:v1.11 transport:tagged-image-to-add`,
		Args: cobra.MinimumNArgs(1),
	}
	manifestCreateCommand.SetUsageTemplate(UsageTemplate())
	flags := manifestCreateCommand.Flags()
	flags.BoolVar(&manifestCreateOpts.all, "all", false, "add all of the lists' images if the images to add are lists")
	flags.StringVar(&manifestCreateOpts.os, "os", "", "if any of the specified images is a list, choose the one for `os`")
	if err := flags.MarkHidden("os"); err != nil {
		panic(fmt.Sprintf("error marking --os as hidden: %v", err))
	}
	flags.StringVar(&manifestCreateOpts.arch, "arch", "", "if any of the specified images is a list, choose the one for `arch`")
	if err := flags.MarkHidden("arch"); err != nil {
		panic(fmt.Sprintf("error marking --arch as hidden: %v", err))
	}
	flags.SetNormalizeFunc(cli.AliasFlags)
	manifestCommand.AddCommand(manifestCreateCommand)

	manifestAddCommand := &cobra.Command{
		Use:   "add",
		Short: "Add images to a manifest list or image index",
		Long:  manifestAddDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestAddCmd(cmd, args, manifestAddOpts)
		},
		Example: `buildah manifest add mylist:v1.11 image:v1.11-amd64
  buildah manifest add mylist:v1.11 transport:imageName`,
		Args: cobra.MinimumNArgs(2),
	}
	manifestAddCommand.SetUsageTemplate(UsageTemplate())
	flags = manifestAddCommand.Flags()
	flags.StringVar(&manifestAddOpts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&manifestAddOpts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&manifestAddOpts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVar(&manifestAddOpts.os, "os", "", "override the `OS` of the specified image")
	flags.StringVar(&manifestAddOpts.arch, "arch", "", "override the `architecture` of the specified image")
	flags.StringVar(&manifestAddOpts.variant, "variant", "", "override the `variant` of the specified image")
	flags.StringVar(&manifestAddOpts.osVersion, "os-version", "", "override the OS `version` of the specified image")
	flags.StringSliceVar(&manifestAddOpts.features, "features", nil, "override the `features` of the specified image")
	flags.StringSliceVar(&manifestAddOpts.osFeatures, "os-features", nil, "override the OS `features` of the specified image")
	flags.StringSliceVar(&manifestAddOpts.annotations, "annotation", nil, "set an `annotation` for the specified image")
	flags.BoolVar(&manifestAddOpts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	flags.BoolVar(&manifestAddOpts.all, "all", false, "add all of the list's images if the image is a list")
	flags.SetNormalizeFunc(cli.AliasFlags)
	manifestCommand.AddCommand(manifestAddCommand)

	manifestRemoveCommand := &cobra.Command{
		Use:   "remove",
		Short: "Remove an entry from a manifest list or image index",
		Long:  manifestRemoveDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestRemoveCmd(cmd, args, manifestRemoveOpts)
		},
		Example: `buildah manifest remove mylist:v1.11 sha256:15352d97781ffdf357bf3459c037be3efac4133dc9070c2dce7eca7c05c3e736`,
		Args:    cobra.MinimumNArgs(2),
	}
	manifestRemoveCommand.SetUsageTemplate(UsageTemplate())
	manifestCommand.AddCommand(manifestRemoveCommand)

	manifestAnnotateCommand := &cobra.Command{
		Use:   "annotate",
		Short: "Add or update information about an entry in a manifest list or image index",
		Long:  manifestAnnotateDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestAnnotateCmd(cmd, args, manifestAnnotateOpts)
		},
		Example: `buildah manifest annotate --annotation left=right mylist:v1.11 image:v1.11-amd64`,
		Args:    cobra.MinimumNArgs(2),
	}
	flags = manifestAnnotateCommand.Flags()
	flags.StringVar(&manifestAnnotateOpts.os, "os", "", "override the `OS` of the specified image")
	flags.StringVar(&manifestAnnotateOpts.arch, "arch", "", "override the `Architecture` of the specified image")
	flags.StringVar(&manifestAnnotateOpts.variant, "variant", "", "override the `Variant` of the specified image")
	flags.StringVar(&manifestAnnotateOpts.osVersion, "os-version", "", "override the os `version` of the specified image")
	flags.StringSliceVar(&manifestAnnotateOpts.features, "features", nil, "override the `features` of the specified image")
	flags.StringSliceVar(&manifestAnnotateOpts.osFeatures, "os-features", nil, "override the os `features` of the specified image")
	flags.StringSliceVar(&manifestAnnotateOpts.annotations, "annotation", nil, "set an `annotation` for the specified image")
	manifestAnnotateCommand.SetUsageTemplate(UsageTemplate())
	manifestCommand.AddCommand(manifestAnnotateCommand)

	manifestInspectCommand := &cobra.Command{
		Use:   "inspect",
		Short: "Display the contents of a manifest list or image index",
		Long:  manifestInspectDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestInspectCmd(cmd, args, manifestInspectOpts)
		},
		Example: `buildah manifest inspect mylist:v1.11`,
		Args:    cobra.MinimumNArgs(1),
	}
	manifestInspectCommand.SetUsageTemplate(UsageTemplate())
	manifestCommand.AddCommand(manifestInspectCommand)

	manifestPushCommand := &cobra.Command{
		Use:   "push",
		Short: "Push a manifest list or image index to a registry",
		Long:  manifestPushDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestPushCmd(cmd, args, manifestPushOpts)
		},
		Example: `buildah manifest push mylist:v1.11 transport:imageName`,
		Args:    cobra.MinimumNArgs(2),
	}
	manifestPushCommand.SetUsageTemplate(UsageTemplate())
	flags = manifestPushCommand.Flags()
	flags.BoolVar(&manifestPushOpts.rm, "rm", false, "remove the manifest list if push succeeds")
	flags.BoolVar(&manifestPushOpts.all, "all", false, "also push the images in the list")
	flags.StringVar(&manifestPushOpts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&manifestPushOpts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&manifestPushOpts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVar(&manifestPushOpts.digestfile, "digestfile", "", "after copying the image, write the digest of the resulting digest to the file")
	flags.StringVarP(&manifestPushOpts.format, "format", "f", "", "manifest type (oci or v2s2) to attempt to use when pushing the manifest list (default is manifest type of source)")
	flags.BoolVarP(&manifestPushOpts.removeSignatures, "remove-signatures", "", false, "don't copy signatures when pushing images")
	flags.StringVar(&manifestPushOpts.signBy, "sign-by", "", "sign the image using a GPG key with the specified `FINGERPRINT`")
	flags.StringVar(&manifestPushOpts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
	flags.BoolVar(&manifestPushOpts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	flags.BoolVarP(&manifestPushOpts.quiet, "quiet", "q", false, "don't output progress information when pushing lists")
	flags.SetNormalizeFunc(cli.AliasFlags)
	manifestCommand.AddCommand(manifestPushCommand)

	manifestRmCommand := &cobra.Command{
		Use:   "rm",
		Short: "Remove manifest list",
		Long:  manifestRmDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return manifestRmCmd(cmd, args)
		},
		Example: `buildah manifest rm mylist:v1.11`,
		Args:    cobra.MinimumNArgs(1),
	}
	manifestRmCommand.SetUsageTemplate(UsageTemplate())
	manifestCommand.AddCommand(manifestRmCommand)
}

func manifestCreateCmd(c *cobra.Command, args []string, opts manifestCreateOpts) error {
	if len(args) == 0 {
		return errors.New("At least a name must be specified for the list")
	}
	listImageSpec := args[0]
	imageSpecs := args[1:]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	list := manifests.Create()

	names, err := util.ExpandNames([]string{listImageSpec}, systemContext, store)
	if err != nil {
		return errors.Wrapf(err, "error encountered while expanding image name %q", listImageSpec)
	}

	for _, imageSpec := range imageSpecs {
		ref, err := alltransports.ParseImageName(imageSpec)
		if err != nil {
			if ref, err = alltransports.ParseImageName(util.DefaultTransport + imageSpec); err != nil {
				// check if the local image exists
				if ref, _, err = util.FindImage(store, "", systemContext, imageSpec); err != nil {
					return err
				}
			}
		}
		if _, err = list.Add(getContext(), systemContext, ref, opts.all); err != nil {
			return err
		}
	}

	imageID, err := list.SaveToImage(store, "", names, manifest.DockerV2ListMediaType)
	if err == nil {
		fmt.Printf("%s\n", imageID)
	}
	return err
}

func manifestAddCmd(c *cobra.Command, args []string, opts manifestAddOpts) error {
	if c.Flag("authfile").Changed {
		if err := auth.CheckAuthFile(opts.authfile); err != nil {
			return err
		}
	}

	listImageSpec := ""
	imageSpec := ""
	switch len(args) {
	case 0, 1:
		return errors.New("At least a list image and an image to add must be specified")
	case 2:
		listImageSpec = args[0]
		if listImageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[0])
		}
		imageSpec = args[1]
		if imageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[1])
		}
	default:
		return errors.New("At least two arguments are necessary: list and image to add to list")
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

	manifestList, err := runtime.LookupManifestList(listImageSpec)
	if err != nil {
		return err
	}
	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	ref, err := alltransports.ParseImageName(imageSpec)
	if err != nil {
		if ref, err = alltransports.ParseImageName(util.DefaultTransport + imageSpec); err != nil {
			// check if the local image exists
			if ref, _, err = util.FindImage(store, "", systemContext, imageSpec); err != nil {
				return err
			}
		}
	}

	digest, err := list.Add(getContext(), systemContext, ref, opts.all)
	if err != nil {
		var storeErr error
		// check if the local image exists
		if ref, _, storeErr = util.FindImage(store, "", systemContext, imageSpec); storeErr != nil {
			return err
		}
		digest, storeErr = list.Add(getContext(), systemContext, ref, opts.all)
		if storeErr != nil {
			return err
		}
	}

	if opts.os != "" {
		if err := list.SetOS(digest, opts.os); err != nil {
			return err
		}
	}
	if opts.osVersion != "" {
		if err := list.SetOSVersion(digest, opts.osVersion); err != nil {
			return err
		}
	}
	if len(opts.osFeatures) != 0 {
		if err := list.SetOSFeatures(digest, opts.osFeatures); err != nil {
			return err
		}
	}
	if opts.arch != "" {
		if err := list.SetArchitecture(digest, opts.arch); err != nil {
			return err
		}
	}
	if opts.variant != "" {
		if err := list.SetVariant(digest, opts.variant); err != nil {
			return err
		}
	}
	if len(opts.features) != 0 {
		if err := list.SetFeatures(digest, opts.features); err != nil {
			return err
		}
	}
	if len(opts.annotations) != 0 {
		annotations := make(map[string]string)
		for _, annotationSpec := range opts.annotations {
			spec := strings.SplitN(annotationSpec, "=", 2)
			if len(spec) != 2 {
				return errors.Errorf("no value given for annotation %q", spec[0])
			}
			annotations[spec[0]] = spec[1]
		}
		if err := list.SetAnnotations(&digest, annotations); err != nil {
			return err
		}
	}

	updatedListID, err := list.SaveToImage(store, manifestList.ID(), nil, "")
	if err == nil {
		fmt.Printf("%s: %s\n", updatedListID, digest.String())
	}

	return err
}

func manifestRemoveCmd(c *cobra.Command, args []string, opts manifestRemoveOpts) error {
	listImageSpec := ""
	var instanceDigest digest.Digest
	switch len(args) {
	case 0, 1:
		return errors.New("At least a list image and one or more instance digests must be specified")
	case 2:
		listImageSpec = args[0]
		if listImageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[0])
		}
		instanceSpec := args[1]
		if instanceSpec == "" {
			return errors.Errorf(`Invalid instance "%s"`, args[1])
		}
		d, err := digest.Parse(instanceSpec)
		if err != nil {
			return errors.Errorf(`Invalid instance "%s": %v`, args[1], err)
		}
		instanceDigest = d
	default:
		return errors.New("At least two arguments are necessary: list and digest of instance to remove from list")
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
	manifestList, err := runtime.LookupManifestList(listImageSpec)
	if err != nil {
		return err
	}

	if err := manifestList.RemoveInstance(instanceDigest); err != nil {
		return err
	}

	fmt.Printf("%s: %s\n", manifestList.ID(), instanceDigest.String())

	return nil
}

func manifestRmCmd(c *cobra.Command, args []string) error {
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

	options := &libimage.RemoveImagesOptions{
		Filters: []string{"readonly=false"},
	}
	rmiReports, rmiErrors := runtime.RemoveImages(context.Background(), args, options)
	for _, r := range rmiReports {
		for _, u := range r.Untagged {
			fmt.Printf("untagged: %s\n", u)
		}
	}
	for _, r := range rmiReports {
		if r.Removed {
			fmt.Printf("%s\n", r.ID)
		}
	}

	var multiE *multierror.Error
	multiE = multierror.Append(multiE, rmiErrors...)
	return multiE.ErrorOrNil()
}

func manifestAnnotateCmd(c *cobra.Command, args []string, opts manifestAnnotateOpts) error {
	listImageSpec := ""
	imageSpec := ""
	switch len(args) {
	case 0:
		return errors.New("At least a list image must be specified")
	case 1:
		listImageSpec = args[0]
		if listImageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[0])
		}
	case 2:
		listImageSpec = args[0]
		if listImageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[0])
		}
		imageSpec = args[1]
		if imageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, args[1])
		}
	default:
		return errors.New("At least two arguments are necessary: list and image to add to list")
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

	manifestList, err := runtime.LookupManifestList(listImageSpec)
	if err != nil {
		return err
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	digest, err := digest.Parse(imageSpec)
	if err != nil {
		ctx := getContext()
		ref, _, err := util.FindImage(store, "", systemContext, imageSpec)
		if err != nil {
			return err
		}
		img, err := ref.NewImageSource(ctx, systemContext)
		if err != nil {
			return err
		}
		defer img.Close()
		manifestBytes, _, err := img.GetManifest(ctx, nil)
		if err != nil {
			return err
		}
		digest, err = manifest.Digest(manifestBytes)
		if err != nil {
			return err
		}
	}

	if opts.os != "" {
		if err := list.SetOS(digest, opts.os); err != nil {
			return err
		}
	}
	if opts.osVersion != "" {
		if err := list.SetOSVersion(digest, opts.osVersion); err != nil {
			return err
		}
	}
	if len(opts.osFeatures) != 0 {
		if err := list.SetOSFeatures(digest, opts.osFeatures); err != nil {
			return err
		}
	}
	if opts.arch != "" {
		if err := list.SetArchitecture(digest, opts.arch); err != nil {
			return err
		}
	}
	if opts.variant != "" {
		if err := list.SetVariant(digest, opts.variant); err != nil {
			return err
		}
	}
	if len(opts.features) != 0 {
		if err := list.SetFeatures(digest, opts.features); err != nil {
			return err
		}
	}
	if len(opts.annotations) != 0 {
		annotations := make(map[string]string)
		for _, annotationSpec := range opts.annotations {
			spec := strings.SplitN(annotationSpec, "=", 2)
			if len(spec) != 2 {
				return errors.Errorf("no value given for annotation %q", spec[0])
			}
			annotations[spec[0]] = spec[1]
		}
		if err := list.SetAnnotations(&digest, annotations); err != nil {
			return err
		}
	}

	updatedListID, err := list.SaveToImage(store, manifestList.ID(), nil, "")
	if err == nil {
		fmt.Printf("%s: %s\n", updatedListID, digest.String())
	}

	return nil
}

func manifestInspectCmd(c *cobra.Command, args []string, opts manifestInspectOpts) error {
	imageSpec := ""
	switch len(args) {
	case 0:
		return errors.New("At least a source list ID must be specified")
	case 1:
		imageSpec = args[0]
		if imageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, imageSpec)
		}
	default:
		return errors.New("Only one argument is necessary for inspect: an image name")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	return manifestInspect(getContext(), store, systemContext, imageSpec)
}

func manifestInspect(ctx context.Context, store storage.Store, systemContext *types.SystemContext, imageSpec string) error {
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	printManifest := func(manifest []byte) error {
		var b bytes.Buffer
		err = json.Indent(&b, manifest, "", "    ")
		if err != nil {
			return errors.Wrapf(err, "error rendering manifest for display")
		}

		fmt.Printf("%s\n", b.String())
		return nil
	}

	// Before doing a remote lookup, attempt to resolve the manifest list
	// locally.
	manifestList, err := runtime.LookupManifestList(imageSpec)
	switch errors.Cause(err) {
	case storage.ErrImageUnknown, libimage.ErrNotAManifestList:
		// We need to do the remote inspection below.
	case nil:
		schema2List, err := manifestList.Inspect()
		if err != nil {
			return err
		}

		rawSchema2List, err := json.Marshal(schema2List)
		if err != nil {
			return err
		}

		return printManifest(rawSchema2List)

	default:
		// Fatal error.
		return err
	}

	// TODO: at some point `libimage` should support resolving manifests
	// like that.  Similar to `libimage.Runtime.LookupImage` we could
	// implement a `*.LookupImageIndex`.
	refs, err := util.ResolveNameToReferences(store, systemContext, imageSpec)
	if err != nil {
		logrus.Debugf("error parsing reference to image %q: %v", imageSpec, err)
	}

	if ref, _, err := util.FindImage(store, "", systemContext, imageSpec); err == nil {
		refs = append(refs, ref)
	} else if ref, err := alltransports.ParseImageName(imageSpec); err == nil {
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		return errors.Errorf("error locating images with names %v", imageSpec)
	}

	var (
		latestErr error
		result    []byte
	)

	appendErr := func(e error) {
		if latestErr == nil {
			latestErr = e
		} else {
			latestErr = errors.Wrapf(latestErr, "tried %v\n", e)
		}
	}

	for _, ref := range refs {
		logrus.Debugf("Testing reference %q for possible manifest", transports.ImageName(ref))

		src, err := ref.NewImageSource(ctx, systemContext)
		if err != nil {
			appendErr(errors.Wrapf(err, "reading image %q", transports.ImageName(ref)))
			continue
		}
		defer src.Close()

		manifestBytes, manifestType, err := src.GetManifest(ctx, nil)
		if err != nil {
			appendErr(errors.Wrapf(err, "loading manifest %q", transports.ImageName(ref)))
			continue
		}

		if !manifest.MIMETypeIsMultiImage(manifestType) {
			appendErr(errors.Errorf("manifest is of type %s (not a list type)", manifestType))
			continue
		}
		result = manifestBytes
		break
	}
	if len(result) == 0 && latestErr != nil {
		return latestErr
	}

	return printManifest(result)
}

func manifestPushCmd(c *cobra.Command, args []string, opts pushOptions) error {
	if c.Flag("authfile").Changed {
		if err := auth.CheckAuthFile(opts.authfile); err != nil {
			return err
		}
	}

	listImageSpec := ""
	destSpec := ""
	switch len(args) {
	case 0:
		return errors.New("At least a source list ID must be specified")
	case 1:
		return errors.New("Two arguments are necessary to push: source and destination")
	case 2:
		listImageSpec = args[0]
		destSpec = args[1]
		if listImageSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, listImageSpec)
		}
		if destSpec == "" {
			return errors.Errorf(`Invalid image name "%s"`, destSpec)
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

	return manifestPush(systemContext, store, listImageSpec, destSpec, opts)
}

func manifestPush(systemContext *types.SystemContext, store storage.Store, listImageSpec, destSpec string, opts pushOptions) error {
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return err
	}

	manifestList, err := runtime.LookupManifestList(listImageSpec)
	if err != nil {
		return err
	}

	_, list, err := manifests.LoadFromImage(store, manifestList.ID())
	if err != nil {
		return err
	}

	dest, err := alltransports.ParseImageName(destSpec)
	if err != nil {
		return err
	}

	var manifestType string
	if opts.format != "" {
		switch opts.format {
		case "oci":
			manifestType = imgspecv1.MediaTypeImageManifest
		case "v2s2", "docker":
			manifestType = manifest.DockerV2Schema2MediaType
		default:
			return errors.Errorf("unknown format %q. Choose on of the supported formats: 'oci' or 'v2s2'", opts.format)
		}
	}

	options := manifests.PushOptions{
		Store:              store,
		SystemContext:      systemContext,
		ImageListSelection: cp.CopySpecificImages,
		Instances:          nil,
		RemoveSignatures:   opts.removeSignatures,
		SignBy:             opts.signBy,
		ManifestType:       manifestType,
	}
	if opts.all {
		options.ImageListSelection = cp.CopyAllImages
	}
	if !opts.quiet {
		options.ReportWriter = os.Stderr
	}

	_, digest, err := list.Push(getContext(), dest, options)

	if err == nil && opts.rm {
		_, err = store.DeleteImage(manifestList.ID(), true)
	}

	if opts.digestfile != "" {
		if err = ioutil.WriteFile(opts.digestfile, []byte(digest.String()), 0644); err != nil {
			return util.GetFailureCause(err, errors.Wrapf(err, "failed to write digest to file %q", opts.digestfile))
		}
	}

	return err
}
