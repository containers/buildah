package archive

import (
	"io"
	"os"

	"github.com/containers/image/docker/tarfile"
	"github.com/containers/image/types"
	"github.com/pkg/errors"
)

type archiveImageDestination struct {
	*tarfile.Destination // Implements most of types.ImageDestination
	ref                  archiveReference
	writer               io.Closer
}

func newImageDestination(ctx *types.SystemContext, ref archiveReference) (types.ImageDestination, error) {
	if ref.destinationRef == nil {
		return nil, errors.Errorf("docker-archive: destination reference not supplied (must be of form <path>:<reference:tag>)")
	}
	fh, err := os.OpenFile(ref.path, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0644)
	if err != nil {
		// FIXME: It should be possible to modify archives, but the only really
		//        sane way of doing it is to create a copy of the image, modify
		//        it and then do a rename(2).
		if os.IsExist(err) {
			err = errors.New("docker-archive doesn't support modifying existing images")
		}
		return nil, err
	}

	return &archiveImageDestination{
		Destination: tarfile.NewDestination(fh, ref.destinationRef),
		ref:         ref,
		writer:      fh,
	}, nil
}

// Reference returns the reference used to set up this destination.  Note that this should directly correspond to user's intent,
// e.g. it should use the public hostname instead of the result of resolving CNAMEs or following redirects.
func (d *archiveImageDestination) Reference() types.ImageReference {
	return d.ref
}

// Close removes resources associated with an initialized ImageDestination, if any.
func (d *archiveImageDestination) Close() error {
	return d.writer.Close()
}

// Commit marks the process of storing the image as successful and asks for the image to be persisted.
// WARNING: This does not have any transactional semantics:
// - Uploaded data MAY be visible to others before Commit() is called
// - Uploaded data MAY be removed or MAY remain around if Close() is called without Commit() (i.e. rollback is allowed but not guaranteed)
func (d *archiveImageDestination) Commit() error {
	return d.Destination.Commit()
}
