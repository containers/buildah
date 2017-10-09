package buildah

import (
	"bytes"
	"fmt"
	"io"
	"time"

	cp "github.com/containers/image/copy"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/util"
	"github.com/sirupsen/logrus"
)

var (
	// gzippedEmptyLayer is a gzip-compressed version of an empty tar file (just 1024 zero bytes).  This
	// comes from github.com/docker/distribution/manifest/schema1/config_builder.go by way of
	// github.com/containers/image/image/docker_schema2.go; there is a non-zero embedded timestamp; we could
	// zero that, but that would just waste storage space in registries, so let’s use the same values.
	gzippedEmptyLayer = []byte{
		31, 139, 8, 0, 0, 9, 110, 136, 0, 255, 98, 24, 5, 163, 96, 20, 140, 88,
		0, 8, 0, 0, 255, 255, 46, 175, 181, 239, 0, 4, 0, 0,
	}
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
}

// shallowCopy copies the most recent layer, the configuration, and the manifest from one image to another.
// For local storage, which doesn't care about histories and the manifest's contents, that's sufficient, but
// almost any other destination has higher expectations.
// We assume that "dest" is a reference to a local image (specifically, a containers/image/storage.storageReference),
// and will fail if it isn't.
func (b *Builder) shallowCopy(dest types.ImageReference, src types.ImageReference, systemContext *types.SystemContext) error {
	var names []string
	// Read the target image name.
	if dest.DockerReference() != nil {
		names = []string{dest.DockerReference().String()}
	}
	// Open the source for reading and the new image for writing.
	srcImage, err := src.NewImage(systemContext)
	if err != nil {
		return errors.Wrapf(err, "error reading configuration to write to image %q", transports.ImageName(dest))
	}
	defer srcImage.Close()
	destImage, err := dest.NewImageDestination(systemContext)
	if err != nil {
		return errors.Wrapf(err, "error opening image %q for writing", transports.ImageName(dest))
	}
	// Write an empty filesystem layer, because the image layer requires at least one.
	_, err = destImage.PutBlob(bytes.NewReader(gzippedEmptyLayer), types.BlobInfo{Size: int64(len(gzippedEmptyLayer))})
	if err != nil {
		return errors.Wrapf(err, "error writing dummy layer for image %q", transports.ImageName(dest))
	}
	// Read the newly-generated configuration blob.
	config, err := srcImage.ConfigBlob()
	if err != nil {
		return errors.Wrapf(err, "error reading new configuration for image %q", transports.ImageName(dest))
	}
	if len(config) == 0 {
		return errors.Errorf("error reading new configuration for image %q: it's empty", transports.ImageName(dest))
	}
	logrus.Debugf("read configuration blob %q", string(config))
	// Write the configuration to the new image.
	configBlobInfo := types.BlobInfo{
		Digest: digest.Canonical.FromBytes(config),
		Size:   int64(len(config)),
	}
	_, err = destImage.PutBlob(bytes.NewReader(config), configBlobInfo)
	if err != nil && len(config) > 0 {
		return errors.Wrapf(err, "error writing image configuration for temporary copy of %q", transports.ImageName(dest))
	}
	// Read the newly-generated, mostly fake, manifest.
	manifest, _, err := srcImage.Manifest()
	if err != nil {
		return errors.Wrapf(err, "error reading new manifest for image %q", transports.ImageName(dest))
	}
	// Write the manifest to the new image.
	err = destImage.PutManifest(manifest)
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
	// Locate the new image in the lower-level API.  Extract its items.
	destImg, err := is.Transport.GetStoreImage(b.store, dest)
	if err != nil {
		return errors.Wrapf(err, "error locating new image %q", transports.ImageName(dest))
	}
	items, err := b.store.ListImageBigData(destImg.ID)
	if err != nil {
		return errors.Wrapf(err, "error reading list of named data for image %q", destImg.ID)
	}
	bigdata := make(map[string][]byte)
	for _, itemName := range items {
		var data []byte
		data, err = b.store.ImageBigData(destImg.ID, itemName)
		if err != nil {
			return errors.Wrapf(err, "error reading named data %q for image %q", itemName, destImg.ID)
		}
		bigdata[itemName] = data
	}
	// Delete the image so that we can recreate it.
	_, err = b.store.DeleteImage(destImg.ID, true)
	if err != nil {
		return errors.Wrapf(err, "error deleting image %q for rewriting", destImg.ID)
	}
	// Look up the container's read-write layer.
	container, err := b.store.Container(b.ContainerID)
	if err != nil {
		return errors.Wrapf(err, "error reading information about working container %q", b.ContainerID)
	}
	parentLayer := ""
	// Look up the container's source image's layer, if there is a source image.
	if container.ImageID != "" {
		img, err2 := b.store.Image(container.ImageID)
		if err2 != nil {
			return errors.Wrapf(err2, "error reading information about working container %q's source image", b.ContainerID)
		}
		parentLayer = img.TopLayer
	}
	// Extract the read-write layer's contents.
	layerDiff, err := b.store.Diff(parentLayer, container.LayerID, nil)
	if err != nil {
		return errors.Wrapf(err, "error reading layer %q from source image %q", container.LayerID, transports.ImageName(src))
	}
	defer layerDiff.Close()
	// Write a copy of the layer for the new image to reference.
	layer, _, err := b.store.PutLayer("", parentLayer, []string{}, "", false, layerDiff)
	if err != nil {
		return errors.Wrapf(err, "error creating new read-only layer from container %q", b.ContainerID)
	}
	// Create a low-level image record that uses the new layer, discarding the old metadata.
	image, err := b.store.CreateImage(destImg.ID, []string{}, layer.ID, "{}", nil)
	if err != nil {
		err2 := b.store.DeleteLayer(layer.ID)
		if err2 != nil {
			logrus.Debugf("error removing layer %q: %v", layer, err2)
		}
		return errors.Wrapf(err, "error creating new low-level image %q", transports.ImageName(dest))
	}
	logrus.Debugf("(re-)created image ID %q using layer %q", image.ID, layer.ID)
	defer func() {
		if err != nil {
			_, err2 := b.store.DeleteImage(image.ID, true)
			if err2 != nil {
				logrus.Debugf("error removing image %q: %v", image.ID, err2)
			}
		}
	}()
	// Store the configuration and manifest, which are big data items, along with whatever else is there.
	for itemName, data := range bigdata {
		err = b.store.SetImageBigData(image.ID, itemName, data)
		if err != nil {
			return errors.Wrapf(err, "error saving data item %q", itemName)
		}
		logrus.Debugf("saved data item %q to %q", itemName, image.ID)
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
	policy, err := signature.DefaultPolicy(getSystemContext(options.SignaturePolicyPath))
	if err != nil {
		return errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrapf(err, "error creating new signature policy context")
	}
	defer func() {
		if err2 := policyContext.Destroy(); err2 != nil {
			logrus.Debugf("error destroying signature polcy context: %v", err2)
		}
	}()
	// Check if we're keeping everything in local storage.  If so, we can take certain shortcuts.
	_, destIsStorage := dest.Transport().(is.StoreTransport)
	exporting := !destIsStorage
	src, err := b.makeContainerImageRef(options.PreferredManifestType, exporting, options.Compression, options.HistoryTimestamp)
	if err != nil {
		return errors.Wrapf(err, "error computing layer digests and building metadata")
	}
	if exporting {
		// Copy everything.
		err = cp.Image(policyContext, dest, src, getCopyOptions(options.ReportWriter, nil, options.SystemContext))
		if err != nil {
			return errors.Wrapf(err, "error copying layers and metadata")
		}
	} else {
		// Copy only the most recent layer, the configuration, and the manifest.
		err = b.shallowCopy(dest, src, getSystemContext(options.SignaturePolicyPath))
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
	systemContext := getSystemContext(options.SignaturePolicyPath)
	policy, err := signature.DefaultPolicy(systemContext)
	if err != nil {
		return errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return errors.Wrapf(err, "error creating new signature policy context")
	}
	defer func() {
		if err2 := policyContext.Destroy(); err2 != nil {
			logrus.Debugf("error destroying signature polcy context: %v", err2)
		}
	}()
	importOptions := ImportFromImageOptions{
		Image:               image,
		SignaturePolicyPath: options.SignaturePolicyPath,
	}
	builder, err := importBuilderFromImage(options.Store, importOptions)
	if err != nil {
		return errors.Wrap(err, "error importing builder information from image")
	}
	// Look up the image name and its layer.
	ref, err := is.Transport.ParseStoreReference(options.Store, image)
	if err != nil {
		return errors.Wrapf(err, "error parsing reference to image %q", image)
	}
	img, err := is.Transport.GetStoreImage(options.Store, ref)
	if err != nil {
		return errors.Wrapf(err, "error locating image %q", image)
	}
	// Give the image we're producing the same ancestors as its source image.
	builder.FromImage = builder.Docker.ContainerConfig.Image
	builder.FromImageID = string(builder.Docker.Parent)
	// Prep the layers and manifest for export.
	src, err := builder.makeImageImageRef(options.Compression, img.Names, img.TopLayer, nil)
	if err != nil {
		return errors.Wrapf(err, "error recomputing layer digests and building metadata")
	}
	// Copy everything.
	err = cp.Image(policyContext, dest, src, getCopyOptions(options.ReportWriter, nil, options.SystemContext))
	if err != nil {
		return errors.Wrapf(err, "error copying layers and metadata")
	}
	if options.ReportWriter != nil {
		fmt.Fprintf(options.ReportWriter, "\n")
	}
	return nil
}
