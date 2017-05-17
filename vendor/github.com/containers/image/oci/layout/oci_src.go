package layout

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/containers/image/types"
	"github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type ociImageSource struct {
	ref        ociReference
	descriptor imgspecv1.ManifestDescriptor
}

// newImageSource returns an ImageSource for reading from an existing directory.
func newImageSource(ref ociReference) (types.ImageSource, error) {
	descriptor, err := ref.getManifestDescriptor()
	if err != nil {
		return nil, err
	}
	return &ociImageSource{ref: ref, descriptor: descriptor}, nil
}

// Reference returns the reference used to set up this source.
func (s *ociImageSource) Reference() types.ImageReference {
	return s.ref
}

// Close removes resources associated with an initialized ImageSource, if any.
func (s *ociImageSource) Close() error {
	return nil
}

// GetManifest returns the image's manifest along with its MIME type (which may be empty when it can't be determined but the manifest is available).
// It may use a remote (= slow) service.
func (s *ociImageSource) GetManifest() ([]byte, string, error) {
	manifestPath, err := s.ref.blobPath(digest.Digest(s.descriptor.Digest))
	if err != nil {
		return nil, "", err
	}
	m, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}

	return m, s.descriptor.MediaType, nil
}

func (s *ociImageSource) GetTargetManifest(digest digest.Digest) ([]byte, string, error) {
	manifestPath, err := s.ref.blobPath(digest)
	if err != nil {
		return nil, "", err
	}

	m, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return nil, "", err
	}

	// XXX: GetTargetManifest means that we don't have the context of what
	//      mediaType the manifest has. In OCI this means that we don't know
	//      what reference it came from, so we just *assume* that its
	//      MediaTypeImageManifest.
	return m, imgspecv1.MediaTypeImageManifest, nil
}

// GetBlob returns a stream for the specified blob, and the blob's size.
func (s *ociImageSource) GetBlob(info types.BlobInfo) (io.ReadCloser, int64, error) {
	path, err := s.ref.blobPath(info.Digest)
	if err != nil {
		return nil, 0, err
	}

	r, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	fi, err := r.Stat()
	if err != nil {
		return nil, 0, err
	}
	return r, fi.Size(), nil
}

func (s *ociImageSource) GetSignatures() ([][]byte, error) {
	return [][]byte{}, nil
}
