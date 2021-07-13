package source

import (
	"context"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	"github.com/pkg/errors"
)

// PushOptions includes data to alter certain knobs when pushing a source
// image.
type PushOptions struct {
	// Require HTTPS and verify certificates when accessing the registry.
	TLSVerify bool
	// [username[:password] to use when connecting to the registry.
	Credentials string
}

// Push the source image at `sourcePath` to `imageInput` at a container
// registry.
func Push(ctx context.Context, sourcePath string, imageInput string, options PushOptions) error {
	ociSource, err := openOrCreateSourceImage(ctx, sourcePath)
	if err != nil {
		return err
	}

	destRef, err := stringToImageReference(imageInput)
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

	policy, err := signature.DefaultPolicy(sysCtx)
	if err != nil {
		return errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrapf(err, "error creating new signature policy context")
	}

	copyOpts := &copy.Options{
		DestinationCtx: sysCtx,
	}
	if _, err := copy.Image(ctx, policyContext, destRef, ociSource.Reference(), copyOpts); err != nil {
		return errors.Wrap(err, "error pushing source image")
	}

	return nil
}
