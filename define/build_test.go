package define

import (
	"go.podman.io/common/libimage"
	"go.podman.io/common/libimage/manifests"
)

// We changed a field of the latter type to the former, so make sure they're
// still type aliases so that doing so doesn't break API.
var _ libimage.LookupReferenceFunc = manifests.LookupReferenceFunc(nil)
