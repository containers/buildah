package source

import (
	"context"
	"os"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/pkg/shortnames"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
)

// PullOptions includes data to alter certain knobs when pulling a source
// image.
type PullOptions struct {
	// Require HTTPS and verify certificates when accessing the registry.
	TLSVerify bool
	// [username[:password] to use when connecting to the registry.
	Credentials string
}

// Pull `imageInput` from a container registry to `sourcePath`.
func Pull(ctx context.Context, imageInput string, sourcePath string, options PullOptions) error {
	if _, err := os.Stat(sourcePath); err == nil {
		return errors.Errorf("%q already exists", sourcePath)
	}

	srcRef, err := stringToImageReference(imageInput)
	if err != nil {
		return err
	}

	sysCtx := &types.SystemContext{
		DockerInsecureSkipTLSVerify: types.NewOptionalBool(!options.TLSVerify),
	}
	if options.Credentials != "" {
		authConf, err := parse.AuthConfig(options.Credentials)
		if err != nil {
			return err
		}
		sysCtx.DockerAuthConfig = authConf
	}

	if err := validateSourceImageReference(ctx, srcRef, sysCtx); err != nil {
		return err
	}

	ociDest, err := openOrCreateSourceImage(ctx, sourcePath)
	if err != nil {
		return err
	}

	policy, err := signature.DefaultPolicy(sysCtx)
	if err != nil {
		return errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrapf(err, "error creating new signature policy context")
	}

	copyOpts := copy.Options{
		SourceCtx: sysCtx,
	}
	if _, err := copy.Image(ctx, policyContext, ociDest.Reference(), srcRef, &copyOpts); err != nil {
		return errors.Wrap(err, "error pulling source image")
	}

	return nil
}

func stringToImageReference(imageInput string) (types.ImageReference, error) {
	if shortnames.IsShortName(imageInput) {
		return nil, errors.Errorf("pulling source images by short name (%q) is not supported, please use a fully-qualified name", imageInput)
	}

	ref, err := alltransports.ParseImageName("docker://" + imageInput)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing image name")
	}

	return ref, nil
}

func validateSourceImageReference(ctx context.Context, ref types.ImageReference, sysCtx *types.SystemContext) error {
	src, err := ref.NewImageSource(ctx, sysCtx)
	if err != nil {
		return errors.Wrap(err, "error creating image source from reference")
	}
	defer src.Close()

	ociManifest, _, _, err := readManifestFromImageSource(ctx, src)
	if err != nil {
		return err
	}

	if ociManifest.Config.MediaType != MediaTypeSourceImageConfig {
		return errors.Errorf("invalid media type of image config %q (expected: %q)", ociManifest.Config.MediaType, MediaTypeSourceImageConfig)
	}

	return nil
}
