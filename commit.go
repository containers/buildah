package buildah

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/containers/buildah/pkg/blobcache"
	"github.com/containers/buildah/util"
	cp "github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/manifest"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
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
	// IIDFile tells the builder to write the image ID to the specified file
	IIDFile string
	// Squash tells the builder to produce an image with a single layer
	// instead of with possibly more than one layer.
	Squash bool
	// BlobDirectory is the name of a directory in which we'll look for
	// prebuilt copies of layer blobs that we might otherwise need to
	// regenerate from on-disk layers.  If blobs are available, the
	// manifest of the new image will reference the blobs rather than
	// on-disk layers.
	BlobDirectory string

	// OnBuild is a list of commands to be run by images based on this image
	OnBuild []string
	// Parent is the base image that this image was created by.
	Parent string
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
	// BlobDirectory is the name of a directory in which we'll look for
	// prebuilt copies of layer blobs that we might otherwise need to
	// regenerate from on-disk layers, substituting them in the list of
	// blobs to copy whenever possible.
	BlobDirectory string
}

// Commit writes the contents of the container, along with its updated
// configuration, to a new image in the specified location, and if we know how,
// add any additional tags that were specified. Returns the ID of the new image
// if commit was successful and the image destination was local
func (b *Builder) Commit(ctx context.Context, dest types.ImageReference, options CommitOptions) (string, reference.Canonical, digest.Digest, error) {
	var imgID string

	systemContext := getSystemContext(options.SystemContext, options.SignaturePolicyPath)

	blocked, err := isReferenceBlocked(dest, systemContext)
	if err != nil {
		return "", nil, "", errors.Wrapf(err, "error checking if committing to registry for %q is blocked", transports.ImageName(dest))
	}
	if blocked {
		return "", nil, "", errors.Errorf("commit access to registry for %q is blocked by configuration", transports.ImageName(dest))
	}

	policy, err := signature.DefaultPolicy(systemContext)
	if err != nil {
		return imgID, nil, "", errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return imgID, nil, "", errors.Wrapf(err, "error creating new signature policy context")
	}
	defer func() {
		if err2 := policyContext.Destroy(); err2 != nil {
			logrus.Debugf("error destroying signature policy context: %v", err2)
		}
	}()
	// Check if the base image is already in the destination and it's some kind of local
	// storage.  If so, we can skip recompressing any layers that come from the base image.
	exportBaseLayers := true
	if transport, destIsStorage := dest.Transport().(is.StoreTransport); destIsStorage && b.FromImageID != "" {
		if baseref, err := transport.ParseReference(b.FromImageID); baseref != nil && err == nil {
			if img, err := transport.GetImage(baseref); img != nil && err == nil {
				logrus.Debugf("base image %q is already present in local storage, no need to copy its layers", b.FromImageID)
				exportBaseLayers = false
			}
		}
	}
	src, err := b.makeImageRef(options.PreferredManifestType, options.Parent, exportBaseLayers, options.Squash, options.BlobDirectory, options.Compression, options.HistoryTimestamp)
	if err != nil {
		return imgID, nil, "", errors.Wrapf(err, "error computing layer digests and building metadata for container %q", b.ContainerID)
	}
	var maybeCachedSrc types.ImageReference = src
	var maybeCachedDest types.ImageReference = dest
	if options.BlobDirectory != "" {
		compress := types.PreserveOriginal
		if options.Compression != archive.Uncompressed {
			compress = types.Compress
		}
		cache, err := blobcache.NewBlobCache(src, options.BlobDirectory, compress)
		if err != nil {
			return imgID, nil, "", errors.Wrapf(err, "error wrapping image reference %q in blob cache at %q", transports.ImageName(src), options.BlobDirectory)
		}
		maybeCachedSrc = cache
		cache, err = blobcache.NewBlobCache(dest, options.BlobDirectory, compress)
		if err != nil {
			return imgID, nil, "", errors.Wrapf(err, "error wrapping image reference %q in blob cache at %q", transports.ImageName(dest), options.BlobDirectory)
		}
		maybeCachedDest = cache
	}
	// "Copy" our image to where it needs to be.
	switch options.Compression {
	case archive.Uncompressed:
		systemContext.OCIAcceptUncompressedLayers = true
	case archive.Gzip:
		systemContext.DirForceCompress = true
	}
	var manifestBytes []byte
	if manifestBytes, err = cp.Image(ctx, policyContext, maybeCachedDest, maybeCachedSrc, getCopyOptions(options.ReportWriter, maybeCachedSrc, nil, maybeCachedDest, systemContext, "")); err != nil {
		return imgID, nil, "", errors.Wrapf(err, "error copying layers and metadata for container %q", b.ContainerID)
	}
	if len(options.AdditionalTags) > 0 {
		switch dest.Transport().Name() {
		case is.Transport.Name():
			img, err := is.Transport.GetStoreImage(b.store, dest)
			if err != nil {
				return imgID, nil, "", errors.Wrapf(err, "error locating just-written image %q", transports.ImageName(dest))
			}
			if err = util.AddImageNames(b.store, "", systemContext, img, options.AdditionalTags); err != nil {
				return imgID, nil, "", errors.Wrapf(err, "error setting image names to %v", append(img.Names, options.AdditionalTags...))
			}
			logrus.Debugf("assigned names %v to image %q", img.Names, img.ID)
		default:
			logrus.Warnf("don't know how to add tags to images stored in %q transport", dest.Transport().Name())
		}
	}

	img, err := is.Transport.GetStoreImage(b.store, dest)
	if err != nil && err != storage.ErrImageUnknown {
		return imgID, nil, "", errors.Wrapf(err, "error locating image %q in local storage", transports.ImageName(dest))
	}

	if err == nil {
		imgID = img.ID

		if options.IIDFile != "" {
			if err = ioutil.WriteFile(options.IIDFile, []byte(img.ID), 0644); err != nil {
				return imgID, nil, "", errors.Wrapf(err, "failed to write image ID to file %q", options.IIDFile)
			}
		}
	}

	manifestDigest, err := manifest.Digest(manifestBytes)
	if err != nil {
		return imgID, nil, "", errors.Wrapf(err, "error computing digest of manifest of new image %q", transports.ImageName(dest))
	}

	var ref reference.Canonical
	if name := dest.DockerReference(); name != nil {
		ref, err = reference.WithDigest(name, manifestDigest)
		if err != nil {
			logrus.Warnf("error generating canonical reference with name %q and digest %s: %v", name, manifestDigest.String(), err)
		}
	}

	return imgID, ref, manifestDigest, nil
}

// Push copies the contents of the image to a new location.
func Push(ctx context.Context, image string, dest types.ImageReference, options PushOptions) (reference.Canonical, digest.Digest, error) {
	systemContext := getSystemContext(options.SystemContext, options.SignaturePolicyPath)

	blocked, err := isReferenceBlocked(dest, systemContext)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error checking if pushing to registry for %q is blocked", transports.ImageName(dest))
	}
	if blocked {
		return nil, "", errors.Errorf("push access to registry for %q is blocked by configuration", transports.ImageName(dest))
	}

	policy, err := signature.DefaultPolicy(systemContext)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error obtaining default signature policy")
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error creating new signature policy context")
	}
	// Look up the image.
	src, _, err := util.FindImage(options.Store, "", systemContext, image)
	if err != nil {
		return nil, "", err
	}
	var maybeCachedSrc types.ImageReference = src
	if options.BlobDirectory != "" {
		compress := types.PreserveOriginal
		if options.Compression != archive.Uncompressed {
			compress = types.Compress
		}
		cache, err := blobcache.NewBlobCache(src, options.BlobDirectory, compress)
		if err != nil {
			return nil, "", errors.Wrapf(err, "error wrapping image reference %q in blob cache at %q", transports.ImageName(src), options.BlobDirectory)
		}
		maybeCachedSrc = cache
	}
	// Copy everything.
	switch options.Compression {
	case archive.Uncompressed:
		systemContext.OCIAcceptUncompressedLayers = true
	case archive.Gzip:
		systemContext.DirForceCompress = true
	}
	var manifestBytes []byte
	if manifestBytes, err = cp.Image(ctx, policyContext, dest, maybeCachedSrc, getCopyOptions(options.ReportWriter, maybeCachedSrc, nil, dest, systemContext, options.ManifestType)); err != nil {
		return nil, "", errors.Wrapf(err, "error copying layers and metadata from %q to %q", transports.ImageName(maybeCachedSrc), transports.ImageName(dest))
	}
	if options.ReportWriter != nil {
		fmt.Fprintf(options.ReportWriter, "")
	}
	manifestDigest, err := manifest.Digest(manifestBytes)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error computing digest of manifest of new image %q", transports.ImageName(dest))
	}
	var ref reference.Canonical
	if name := dest.DockerReference(); name != nil {
		ref, err = reference.WithDigest(name, manifestDigest)
		if err != nil {
			logrus.Warnf("error generating canonical reference with name %q and digest %s: %v", name, manifestDigest.String(), err)
		}
	}
	fmt.Printf("Successfully pushed %s@%s\n", dest.StringWithinTransport(), manifestDigest.String())
	return ref, manifestDigest, nil
}
