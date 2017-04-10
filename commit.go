package buildah

import (
	"fmt"

	"github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	"github.com/containers/image/storage"
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
	// AdditionalTags is a list of additional names to add to the image, if
	// the transport to which we're writing the image gives us a way to add
	// them.
	AdditionalTags []string
}

func expandTags(tags []string) ([]string, error) {
	expanded := []string{}
	for _, tag := range tags {
		name, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return nil, fmt.Errorf("error parsing tag %q: %v", tag, err)
		}
		name = reference.TagNameOnly(name)
		tag = ""
		if tagged, ok := name.(reference.NamedTagged); ok {
			tag = ":" + tagged.Tag()
		}
		expanded = append(expanded, name.Name()+tag)
	}
	return expanded, nil
}

// Commit writes the contents of the container, along with its updated
// configuration, to a new image in the specified location, and if we know how,
// add any additional tags that were specified.
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
	switch dest.Transport().Name() {
	case storage.Transport.Name():
		tags, err := expandTags(options.AdditionalTags)
		if err != nil {
			return err
		}
		img, err := storage.Transport.GetStoreImage(b.store, dest)
		if err != nil {
			return err
		}
		err = b.store.SetNames(img.ID, append(img.Names, tags...))
		if err != nil {
			return fmt.Errorf("error setting image names to %v: %v", append(img.Names, tags...), err)
		}
	}
	return err
}
