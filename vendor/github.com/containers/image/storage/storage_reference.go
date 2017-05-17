package storage

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
)

// A storageReference holds an arbitrary name and/or an ID, which is a 32-byte
// value hex-encoded into a 64-character string, and a reference to a Store
// where an image is, or would be, kept.
type storageReference struct {
	transport storageTransport
	reference string
	id        string
	name      reference.Named
}

func newReference(transport storageTransport, reference, id string, name reference.Named) *storageReference {
	// We take a copy of the transport, which contains a pointer to the
	// store that it used for resolving this reference, so that the
	// transport that we'll return from Transport() won't be affected by
	// further calls to the original transport's SetStore() method.
	return &storageReference{
		transport: transport,
		reference: reference,
		id:        id,
		name:      name,
	}
}

// Resolve the reference's name to an image ID in the store, if there's already
// one present with the same name or ID, and return the image.
func (s *storageReference) resolveImage() (*storage.Image, error) {
	if s.id == "" {
		image, err := s.transport.store.Image(s.reference)
		if image != nil && err == nil {
			s.id = image.ID
		}
	}
	if s.id == "" {
		logrus.Errorf("reference %q does not resolve to an image ID", s.StringWithinTransport())
		return nil, ErrNoSuchImage
	}
	img, err := s.transport.store.Image(s.id)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading image %q", s.id)
	}
	if s.reference != "" {
		nameMatch := false
		for _, name := range img.Names {
			if name == s.reference {
				nameMatch = true
				break
			}
		}
		if !nameMatch {
			logrus.Errorf("no image matching reference %q found", s.StringWithinTransport())
			return nil, ErrNoSuchImage
		}
	}
	return img, nil
}

// Return a Transport object that defaults to using the same store that we used
// to build this reference object.
func (s storageReference) Transport() types.ImageTransport {
	return &storageTransport{
		store: s.transport.store,
	}
}

// Return a name with a tag, if we have a name to base them on.
func (s storageReference) DockerReference() reference.Named {
	return s.name
}

// Return a name with a tag, prefixed with the graph root and driver name, to
// disambiguate between images which may be present in multiple stores and
// share only their names.
func (s storageReference) StringWithinTransport() string {
	storeSpec := "[" + s.transport.store.GraphDriverName() + "@" + s.transport.store.GraphRoot() + "]"
	if s.name == nil {
		return storeSpec + "@" + s.id
	}
	if s.id == "" {
		return storeSpec + s.reference
	}
	return storeSpec + s.reference + "@" + s.id
}

func (s storageReference) PolicyConfigurationIdentity() string {
	return s.StringWithinTransport()
}

// Also accept policy that's tied to the combination of the graph root and
// driver name, to apply to all images stored in the Store, and to just the
// graph root, in case we're using multiple drivers in the same directory for
// some reason.
func (s storageReference) PolicyConfigurationNamespaces() []string {
	storeSpec := "[" + s.transport.store.GraphDriverName() + "@" + s.transport.store.GraphRoot() + "]"
	driverlessStoreSpec := "[" + s.transport.store.GraphRoot() + "]"
	namespaces := []string{}
	if s.name != nil {
		if s.id != "" {
			// The reference without the ID is also a valid namespace.
			namespaces = append(namespaces, storeSpec+s.reference)
		}
		components := strings.Split(s.name.Name(), "/")
		for len(components) > 0 {
			namespaces = append(namespaces, storeSpec+strings.Join(components, "/"))
			components = components[:len(components)-1]
		}
	}
	namespaces = append(namespaces, storeSpec)
	namespaces = append(namespaces, driverlessStoreSpec)
	return namespaces
}

func (s storageReference) NewImage(ctx *types.SystemContext) (types.Image, error) {
	return newImage(s)
}

func (s storageReference) DeleteImage(ctx *types.SystemContext) error {
	img, err := s.resolveImage()
	if err != nil {
		return err
	}
	layers, err := s.transport.store.DeleteImage(img.ID, true)
	if err == nil {
		logrus.Debugf("deleted image %q", img.ID)
		for _, layer := range layers {
			logrus.Debugf("deleted layer %q", layer)
		}
	}
	return err
}

func (s storageReference) NewImageSource(ctx *types.SystemContext, requestedManifestMIMETypes []string) (types.ImageSource, error) {
	return newImageSource(s)
}

func (s storageReference) NewImageDestination(ctx *types.SystemContext) (types.ImageDestination, error) {
	return newImageDestination(s)
}
