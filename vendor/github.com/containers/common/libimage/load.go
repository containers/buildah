package libimage

import (
	"context"
	"errors"

	dirTransport "github.com/containers/image/v5/directory"
	dockerArchiveTransport "github.com/containers/image/v5/docker/archive"
	ociArchiveTransport "github.com/containers/image/v5/oci/archive"
	ociTransport "github.com/containers/image/v5/oci/layout"
	"github.com/sirupsen/logrus"
)

type LoadOptions struct {
	CopyOptions
}

// Load loads one or more images (depending on the transport) from the
// specified path.  The path may point to an image the following transports:
// oci, oci-archive, dir, docker-archive.
func (r *Runtime) Load(ctx context.Context, path string, options *LoadOptions) ([]string, error) {
	logrus.Debugf("Loading image from %q", path)

	var (
		loadedImages []string
		loadError    error
	)

	if options == nil {
		options = &LoadOptions{}
	}

	for _, f := range []func() ([]string, error){
		// OCI
		func() ([]string, error) {
			ref, err := ociTransport.NewReference(path, "")
			if err != nil {
				return nil, err
			}
			return r.copyFromDefault(ctx, ref, &options.CopyOptions)
		},

		// OCI-ARCHIVE
		func() ([]string, error) {
			ref, err := ociArchiveTransport.NewReference(path, "")
			if err != nil {
				return nil, err
			}
			return r.copyFromDefault(ctx, ref, &options.CopyOptions)
		},

		// DIR
		func() ([]string, error) {
			ref, err := dirTransport.NewReference(path)
			if err != nil {
				return nil, err
			}
			return r.copyFromDefault(ctx, ref, &options.CopyOptions)
		},

		// DOCKER-ARCHIVE
		func() ([]string, error) {
			ref, err := dockerArchiveTransport.ParseReference(path)
			if err != nil {
				return nil, err
			}
			return r.copyFromDockerArchive(ctx, ref, &options.CopyOptions)
		},

		// Give a decent error message if nothing above worked.
		func() ([]string, error) {
			return nil, errors.New("payload does not match any of the supported image formats (oci, oci-archive, dir, docker-archive)")
		},
	} {
		loadedImages, loadError = f()
		if loadError == nil {
			return loadedImages, loadError
		}
		logrus.Debugf("Error loading %s: %v", path, loadError)
	}

	return nil, loadError
}
