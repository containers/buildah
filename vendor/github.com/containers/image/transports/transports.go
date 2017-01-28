package transports

import (
	"fmt"
	"strings"

	"github.com/containers/image/directory"
	"github.com/containers/image/docker"
	"github.com/containers/image/docker/daemon"
	ociLayout "github.com/containers/image/oci/layout"
	"github.com/containers/image/openshift"
	"github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/pkg/errors"
)

// KnownTransports is a registry of known ImageTransport instances.
var KnownTransports map[string]types.ImageTransport

func init() {
	KnownTransports = make(map[string]types.ImageTransport)
	// NOTE: Make sure docs/policy.json.md is updated when adding or updating
	// a transport.
	for _, t := range []types.ImageTransport{
		directory.Transport,
		docker.Transport,
		daemon.Transport,
		ociLayout.Transport,
		openshift.Transport,
		storage.Transport,
	} {
		name := t.Name()
		if _, ok := KnownTransports[name]; ok {
			panic(fmt.Sprintf("Duplicate image transport name %s", name))
		}
		KnownTransports[name] = t
	}
}

// ParseImageName converts a URL-like image name to a types.ImageReference.
func ParseImageName(imgName string) (types.ImageReference, error) {
	parts := strings.SplitN(imgName, ":", 2)
	if len(parts) != 2 {
		return nil, errors.Errorf(`Invalid image name "%s", expected colon-separated transport:reference`, imgName)
	}
	transport, ok := KnownTransports[parts[0]]
	if !ok {
		return nil, errors.Errorf(`Invalid image name "%s", unknown transport "%s"`, imgName, parts[0])
	}
	return transport.ParseReference(parts[1])
}

// ImageName converts a types.ImageReference into an URL-like image name, which MUST be such that
// ParseImageName(ImageName(reference)) returns an equivalent reference.
//
// This is the generally recommended way to refer to images in the UI.
//
// NOTE: The returned string is not promised to be equal to the original input to ParseImageName;
// e.g. default attribute values omitted by the user may be filled in in the return value, or vice versa.
func ImageName(ref types.ImageReference) string {
	return ref.Transport().Name() + ":" + ref.StringWithinTransport()
}
