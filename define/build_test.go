package define

import (
	"github.com/containers/common/libimage"
	"github.com/containers/common/libimage/manifests"
)

// We changed a field of the latter type to the former, so make sure they're
// still type aliases so that doing so doesn't break API.
var _ libimage.LookupReferenceFunc = manifests.LookupReferenceFunc(nil)
