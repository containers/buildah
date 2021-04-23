// This package is deprecated.  Its functionality has been moved to
// github.com/containers/common/libimage/manifests, which provides the same
// API.  The stubs here are present for compatibility with older code.  New
// implementations should use github.com/containers/common/libimage/manifests
// directly.
package manifests

import (
	"github.com/containers/common/libimage/manifests"
	"github.com/containers/storage"
)

type (
	// List is an alias for github.com/containers/common/libimage/manifests.List.
	List = manifests.List
	// PushOptions is an alias for github.com/containers/common/libimage/manifests.PushOptions.
	PushOptions = manifests.PushOptions
)

var (
	// ErrListImageUnknown is an alias for github.com/containers/common/libimage/manifests.ErrListImageUnknown
	ErrListImageUnknown = manifests.ErrListImageUnknown
)

// Create wraps github.com/containers/common/libimage/manifests.Create().
func Create() List {
	return manifests.Create()
}

// LoadFromImage wraps github.com/containers/common/libimage/manifests.LoadFromImage().
func LoadFromImage(store storage.Store, image string) (string, List, error) {
	return manifests.LoadFromImage(store, image)
}
