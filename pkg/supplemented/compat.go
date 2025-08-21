// This package is deprecated.  Its functionality has been moved to
// github.com/containers/common/pkg/supplemented, which provides the same API.
// The stubs and aliases here are present for compatibility with older code.
// New implementations should use github.com/containers/common/pkg/supplemented
// directly.
package supplemented

import (
	digest "github.com/opencontainers/go-digest"
	"go.podman.io/common/pkg/manifests"
	"go.podman.io/common/pkg/supplemented"
	cp "go.podman.io/image/v5/copy"
	"go.podman.io/image/v5/types"
)

var (
	// ErrDigestNotFound is an alias for github.com/containers/common/pkg/manifests.ErrDigestNotFound.
	ErrDigestNotFound = manifests.ErrDigestNotFound
	// ErrBlobNotFound is an alias for github.com/containers/common/pkg/supplemented.ErrBlobNotFound.
	ErrBlobNotFound = supplemented.ErrBlobNotFound
)

// Reference wraps github.com/containers/common/pkg/supplemented.Reference().
func Reference(ref types.ImageReference, supplemental []types.ImageReference, multiple cp.ImageListSelection, instances []digest.Digest) types.ImageReference {
	return supplemented.Reference(ref, supplemental, multiple, instances)
}
