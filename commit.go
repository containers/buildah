package buildah

import (
	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/archive"
)

type CommitOptions struct {
	Compression         archive.Compression
	SignaturePolicyPath string
}

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
