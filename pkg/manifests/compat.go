// This package is deprecated.  Its functionality has been moved to
// github.com/containers/common/pkg/manifests, which provides the same API.
// The stubs and aliases here are present for compatibility with older code.
// New implementations should use github.com/containers/common/pkg/manifests
// directly.
package manifests

import "github.com/containers/common/pkg/manifests"

// List is an alias for github.com/containers/common/pkg/manifests.List.
type List = manifests.List

var (
	// ErrDigestNotFound is an alias for github.com/containers/common/pkg/manifests.ErrDigestNotFound.
	ErrDigestNotFound = manifests.ErrDigestNotFound
	// ErrManifestTypeNotSupported is an alias for github.com/containers/common/pkg/manifests.ErrManifestTypeNotSupported.
	ErrManifestTypeNotSupported = manifests.ErrManifestTypeNotSupported
)

// Create wraps github.com/containers/common/pkg/manifests.Create().
func Create() List {
	return manifests.Create()
}

// FromBlob wraps github.com/containers/common/pkg/manifests.FromBlob().
func FromBlob(manifestBytes []byte) (List, error) {
	return manifests.FromBlob(manifestBytes)
}
