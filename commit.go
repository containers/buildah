package buildah

import (
	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/archive"
)

// CommitOptions can be used to alter how an image is committed.
type CommitOptions struct {
	// Compression specifies the type of compression which is applied to
	// layer blobs.  The default is to not use compression, but
	// archive.Gzip is recommended.
	Compression archive.Compression
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
}

// Commit writes the contents of the container, along with its updated
// configuration, to a new image in the specified location.
func (b *Builder) Commit(dest types.ImageReference, options CommitOptions) error {
	policy, err := signature.DefaultPolicy(getSystemContext(options.SignaturePolicyPath))
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}
	src, err := b.makeContainerImageRef(options.Compression)
	if err != nil {
		return err
	}
	err = copy.Image(policyContext, dest, src, getCopyOptions())
	return err
}
