package buildah

import (
	"fmt"
	"io"
	"time"

	cp "github.com/containers/image/copy"
	"github.com/containers/image/manifest"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/util"
	"github.com/sirupsen/logrus"
)

// CommitOptions can be used to alter how an image is committed.
type CommitOptions struct {
	// PreferredManifestType is the preferred type of image manifest.  The
	// image configuration format will be of a compatible type.
	PreferredManifestType string
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
	// HistoryTimestamp is the timestamp used when creating new items in the
	// image's history.  If unset, the current time will be used.
	HistoryTimestamp *time.Time
	// github.com/containers/image/types SystemContext to hold credentials
	// and other authentication/authorization information.
	SystemContext *types.SystemContext
}

// PushOptions can be used to alter how an image is copied somewhere.
type PushOptions struct {
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
	// ReportWriter is an io.Writer which will be used to log the writing
	// of the new image.
	ReportWriter io.Writer
	// Store is the local storage store which holds the source image.
	Store storage.Store
	// github.com/containers/image/types SystemContext to hold credentials
	// and other authentication/authorization information.
	SystemContext *types.SystemContext
	// ManifestType is the format to use when saving the imge using the 'dir' transport
	// possible options are oci, v2s1, and v2s2
	ManifestType string
}

// shallowCopy copies the most recent layer, the configuration, and the manifest from one image to another.
// For local storage, which doesn't care about histories and the manifest's contents, that's sufficient, but
// almost any other destination has higher expectations.
// We assume that "dest" is a reference to a local image (specifically, a containers/image/storage.storageReference),
// and will fail if it isn't.
func (b *Builder) shallowCopy(dest types.ImageReference, src types.ImageReference, systemContext *types.SystemContext) error {
	var names []string
	var layerDiff, config io.ReadCloser
	// Read the target image name.
	if dest.DockerReference() != nil {
		names = []string{dest.DockerReference().String()}
	}
	// Open the source for reading and the new image for writing.
	srcImage, err := src.NewImageSource(systemContext)
	if err != nil {
		return errors.Wrapf(err, "error reading configuration to write to image %q", transports.ImageName(dest))
	}
	defer srcImage.Close()
	destImage, err := dest.NewImageDestination(systemContext)
	if err != nil {
		return errors.Wrapf(err, "error opening image %q for writing", transports.ImageName(dest))
	}
	// Read the newly-generated manifest, which already contains a layer entry for the read-write layer.
	manifestBlob, manifestType, err := srcImage.GetManifest(nil)
	if err != nil {
		return errors.Wrapf(err, "error reading the manifest we just generated")
	}
	m, err := manifest.FromBlob(manifestBlob, manifestType)
	if err != nil {
		return errors.Wrapf(err, "error parsing the manifest we just generated")
	}
	// Read the read-write layer blob.
	layerInfos := m.LayerInfos()
	if len(layerInfos) > 0 {
		layerDiff, _, err = srcImage.GetBlob(layerInfos[len(layerInfos)-1])
		if err != nil {
			return errors.Wrapf(err, "error reading the container's layer")
		}
		defer layerDiff.Close()
	}
	// Write a copy of the layer as a blob, for the new image to reference.
	if layerDiff != nil {
		if _, err = destImage.PutBlob(layerDiff, types.BlobInfo{Digest: "", Size: -1}); err != nil {
			return errors.Wrapf(err, "error creating new read-only layer from container %q", b.ContainerID)
		}
	}
	// Read the newly-generated configuration blob.
	configInfo := m.ConfigInfo()
	if configInfo.Size == 0 {
		return errors.Wrapf(err, "error reading new configuration info for image %q", transports.ImageName(dest))
	}
	config, _, err = srcImage.GetBlob(configInfo)
	if err != nil {
		return errors.Wrapf(err, "error reading the new configuration info for image %q", transports.ImageName(dest))
	}
	defer config.Close()
	logrus.Debugf("read configuration blob %q", configInfo.Digest)
	// Write the configuration to the new image.
	if _, err = destImage.PutBlob(config, configInfo); err != nil {
		return errors.Wrapf(err, "error writing image configuration for temporary copy of %q", transports.ImageName(dest))
	}
	// Write the manifest to the new image.
	err = destImage.PutManifest(manifestBlob)
	if err != nil {
		return errors.Wrapf(err, "error writing new manifest to image %q", transports.ImageName(dest))
	}
	// Save the new image.
	err = destImage.Commit()
	if err != nil {
		return errors.Wrapf(err, "error committing new image %q", transports.ImageName(dest))
	}
	err = destImage.Close()
	if err != nil {
		return errors.Wrapf(err, "error closing new image %q", transports.ImageName(dest))
	}
	image, err := is.Transport.GetStoreImage(b.store, dest)
	if err != nil {
		return errors.Wrapf(err, "error locating just-written image %q", transports.ImageName(dest))
	}
	// Add the target name(s) to the new image.
	if len(names) > 0 {
		err = util.AddImageNames(b.store, image, names)
		if err != nil {
			return errors.Wrapf(err, "error assigning names %v to new image", names)
		}
		logrus.Debugf("assigned names %v to image %q", names, image.ID)
	}
	return nil
}

// Commit writes the contents of the container, along with its updated
// configuration, to a new image in the specified location, and if we know how,
// add any additional tags that were specified.
func (b *Builder) Commit(dest types.ImageReference, options CommitOptions) error {
	policy, err := signature.DefaultPolicy(getSystemContext(options.SystemContext, options.SignaturePolicyPath))
	if err != nil {
		return errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrapf(err, "error creating new signature policy context")
	}
	defer func() {
		if err2 := policyContext.Destroy(); err2 != nil {
			logrus.Debugf("error destroying signature policy context: %v", err2)
		}
	}()
	// Check if we're keeping everything in local storage.  If so, we can take certain shortcuts.
	_, destIsStorage := dest.Transport().(is.StoreTransport)
	exporting := !destIsStorage
	src, err := b.makeImageRef(options.PreferredManifestType, exporting, options.Compression, options.HistoryTimestamp)
	if err != nil {
		return errors.Wrapf(err, "error computing layer digests and building metadata")
	}
	if exporting {
		// Copy everything.
		err = cp.Image(policyContext, dest, src, getCopyOptions(options.ReportWriter, nil, options.SystemContext, ""))
		if err != nil {
			return errors.Wrapf(err, "error copying layers and metadata")
		}
	} else {
		// Copy only the most recent layer, the configuration, and the manifest.
		err = b.shallowCopy(dest, src, getSystemContext(options.SystemContext, options.SignaturePolicyPath))
		if err != nil {
			return errors.Wrapf(err, "error copying layer and metadata")
		}
	}
	if len(options.AdditionalTags) > 0 {
		switch dest.Transport().Name() {
		case is.Transport.Name():
			img, err := is.Transport.GetStoreImage(b.store, dest)
			if err != nil {
				return errors.Wrapf(err, "error locating just-written image %q", transports.ImageName(dest))
			}
			err = util.AddImageNames(b.store, img, options.AdditionalTags)
			if err != nil {
				return errors.Wrapf(err, "error setting image names to %v", append(img.Names, options.AdditionalTags...))
			}
			logrus.Debugf("assigned names %v to image %q", img.Names, img.ID)
		default:
			logrus.Warnf("don't know how to add tags to images stored in %q transport", dest.Transport().Name())
		}
	}
	return nil
}

// Push copies the contents of the image to a new location.
func Push(image string, dest types.ImageReference, options PushOptions) error {
	systemContext := getSystemContext(options.SystemContext, options.SignaturePolicyPath)
	policy, err := signature.DefaultPolicy(systemContext)
	if err != nil {
		return errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrapf(err, "error creating new signature policy context")
	}
	// Look up the image.
	src, err := is.Transport.ParseStoreReference(options.Store, image)
	if err != nil {
		return errors.Wrapf(err, "error parsing reference to image %q", image)
	}
	// Copy everything.
	err = cp.Image(policyContext, dest, src, getCopyOptions(options.ReportWriter, nil, options.SystemContext, options.ManifestType))
	if err != nil {
		return errors.Wrapf(err, "error copying layers and metadata")
	}
	if options.ReportWriter != nil {
		fmt.Fprintf(options.ReportWriter, "\n")
	}
	return nil
}
