package buildah

import (
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	"github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/projectatomic/buildah/util"
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
	// ReportWriter is an io.Writer which will be used to log the writing
	// of the new image.
	ReportWriter io.Writer
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
		return fmt.Errorf("error recomputing layer digests and building metadata: %v", err)
	}
	err = copy.Image(policyContext, dest, src, getCopyOptions(options.ReportWriter))
	if err != nil {
		return fmt.Errorf("error copying layers and metadata: %v", err)
	}
	if len(options.AdditionalTags) > 0 {
		switch dest.Transport().Name() {
		case storage.Transport.Name():
			img, err := storage.Transport.GetStoreImage(b.store, dest)
			if err != nil {
				return fmt.Errorf("error locating just-written image %q: %v", transports.ImageName(dest), err)
			}
			err = util.AddImageNames(b.store, img, options.AdditionalTags)
			if err != nil {
				return fmt.Errorf("error setting image names to %v: %v", append(img.Names, options.AdditionalTags...), err)
			}
		default:
			logrus.Warnf("don't know how to add tags to images stored in %q transport", dest.Transport().Name())
		}
	}
	return nil
}
