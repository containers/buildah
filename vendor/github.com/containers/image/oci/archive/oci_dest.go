package archive

import (
	"context"
	"io"
	"os"

	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
)

type ociArchiveImageDestination struct {
	ref          ociArchiveReference
	unpackedDest types.ImageDestination
	tempDirRef   tempDirOCIRef
}

// newImageDestination returns an ImageDestination for writing to an existing directory.
func newImageDestination(ctx context.Context, sys *types.SystemContext, ref ociArchiveReference) (types.ImageDestination, error) {
	tempDirRef, err := createOCIRef(ref.image)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating oci reference")
	}
	unpackedDest, err := tempDirRef.ociRefExtracted.NewImageDestination(ctx, sys)
	if err != nil {
		if err := tempDirRef.deleteTempDir(); err != nil {
			return nil, errors.Wrapf(err, "error deleting temp directory", tempDirRef.tempDirectory)
		}
		return nil, err
	}
	return &ociArchiveImageDestination{ref: ref,
		unpackedDest: unpackedDest,
		tempDirRef:   tempDirRef}, nil
}

// Reference returns the reference used to set up this destination.
func (d *ociArchiveImageDestination) Reference() types.ImageReference {
	return d.ref
}

// Close removes resources associated with an initialized ImageDestination, if any
// Close deletes the temp directory of the oci-archive image
func (d *ociArchiveImageDestination) Close() error {
	defer d.tempDirRef.deleteTempDir()
	return d.unpackedDest.Close()
}

func (d *ociArchiveImageDestination) SupportedManifestMIMETypes() []string {
	return d.unpackedDest.SupportedManifestMIMETypes()
}

// SupportsSignatures returns an error (to be displayed to the user) if the destination certainly can't store signatures
func (d *ociArchiveImageDestination) SupportsSignatures(ctx context.Context) error {
	return d.unpackedDest.SupportsSignatures(ctx)
}

func (d *ociArchiveImageDestination) DesiredLayerCompression() types.LayerCompression {
	return d.unpackedDest.DesiredLayerCompression()
}

// AcceptsForeignLayerURLs returns false iff foreign layers in manifest should be actually
// uploaded to the image destination, true otherwise.
func (d *ociArchiveImageDestination) AcceptsForeignLayerURLs() bool {
	return d.unpackedDest.AcceptsForeignLayerURLs()
}

// MustMatchRuntimeOS returns true iff the destination can store only images targeted for the current runtime OS. False otherwise
func (d *ociArchiveImageDestination) MustMatchRuntimeOS() bool {
	return d.unpackedDest.MustMatchRuntimeOS()
}

// IgnoresEmbeddedDockerReference returns true iff the destination does not care about Image.EmbeddedDockerReferenceConflicts(),
// and would prefer to receive an unmodified manifest instead of one modified for the destination.
// Does not make a difference if Reference().DockerReference() is nil.
func (d *ociArchiveImageDestination) IgnoresEmbeddedDockerReference() bool {
	return d.unpackedDest.IgnoresEmbeddedDockerReference()
}

// PutBlob writes contents of stream and returns data representing the result (with all data filled in).
// inputInfo.Digest can be optionally provided if known; it is not mandatory for the implementation to verify it.
// inputInfo.Size is the expected length of stream, if known.
func (d *ociArchiveImageDestination) PutBlob(ctx context.Context, stream io.Reader, inputInfo types.BlobInfo, isConfig bool) (types.BlobInfo, error) {
	return d.unpackedDest.PutBlob(ctx, stream, inputInfo, isConfig)
}

// HasBlob returns true iff the image destination already contains a blob with the matching digest which can be reapplied using ReapplyBlob
func (d *ociArchiveImageDestination) HasBlob(ctx context.Context, info types.BlobInfo) (bool, int64, error) {
	return d.unpackedDest.HasBlob(ctx, info)
}

func (d *ociArchiveImageDestination) ReapplyBlob(ctx context.Context, info types.BlobInfo) (types.BlobInfo, error) {
	return d.unpackedDest.ReapplyBlob(ctx, info)
}

// PutManifest writes manifest to the destination
func (d *ociArchiveImageDestination) PutManifest(ctx context.Context, m []byte) error {
	return d.unpackedDest.PutManifest(ctx, m)
}

func (d *ociArchiveImageDestination) PutSignatures(ctx context.Context, signatures [][]byte) error {
	return d.unpackedDest.PutSignatures(ctx, signatures)
}

// Commit marks the process of storing the image as successful and asks for the image to be persisted
// after the directory is made, it is tarred up into a file and the directory is deleted
func (d *ociArchiveImageDestination) Commit(ctx context.Context) error {
	if err := d.unpackedDest.Commit(ctx); err != nil {
		return errors.Wrapf(err, "error storing image %q", d.ref.image)
	}

	// path of directory to tar up
	src := d.tempDirRef.tempDirectory
	// path to save tarred up file
	dst := d.ref.resolvedFile
	return tarDirectory(src, dst)
}

// tar converts the directory at src and saves it to dst
func tarDirectory(src, dst string) error {
	// input is a stream of bytes from the archive of the directory at path
	input, err := archive.Tar(src, archive.Uncompressed)
	if err != nil {
		return errors.Wrapf(err, "error retrieving stream of bytes from %q", src)
	}

	// creates the tar file
	outFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "error creating tar file %q", dst)
	}
	defer outFile.Close()

	// copies the contents of the directory to the tar file
	// TODO: This can take quite some time, and should ideally be cancellable using a context.Context.
	_, err = io.Copy(outFile, input)

	return err
}
