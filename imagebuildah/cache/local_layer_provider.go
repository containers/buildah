package cache

import (
	"context"
	"sync"

	"github.com/containers/image/v5/manifest"
	is "github.com/containers/image/v5/storage"
	"github.com/containers/storage"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// LocalLayerProvider is an implementation of the LayerProvider that
type LocalLayerProvider struct {
	store storage.Store

	localLayers     map[string]string
	localLayersLock sync.Mutex
}

// NewLocalLayerProvider creates a new instance of the LocalLayerProvider.
func NewLocalLayerProvider(store storage.Store) LayerProvider {
	return &LocalLayerProvider{
		store:           store,
		localLayers:     make(map[string]string),
		localLayersLock: sync.Mutex{},
	}
}

// CalculateLocalLayerKey returns the layer key for image in the local store
func CalculateLocalLayerKey(manifestType string, parentLayerID string, history []v1.History, digests []digest.Digest) string {
	lastElement := len(history) - 1
	buildAddsLayer := !history[lastElement].EmptyLayer
	createdBy := history[lastElement].CreatedBy
	return CalculateBuildLayerKey(manifestType, buildAddsLayer, parentLayerID, createdBy, history[:lastElement], digests)
}

// getImageTypeAndHistoryAndDiffIDs returns the manifest type, history, and diff IDs list of imageID.
func (llp *LocalLayerProvider) getImageTypeAndHistoryAndDiffIDs(ctx context.Context, imageID string) (string, []v1.History, []digest.Digest, error) {
	imageRef, err := is.Transport.ParseStoreReference(llp.store, "@"+imageID)
	if err != nil {
		return "", nil, nil, errors.Wrapf(err, "failed to obtain the image reference %q", imageID)
	}
	ref, err := imageRef.NewImage(ctx, nil)
	if err != nil {
		return "", nil, nil, errors.Wrapf(err, "failed to create new image from reference to image %q", imageID)
	}
	defer ref.Close()
	oci, err := ref.OCIConfig(ctx)
	if err != nil {
		return "", nil, nil, errors.Wrapf(err, "failed to obtain possibly-converted OCI config of image %q", imageID)
	}
	manifestBytes, manifestFormat, err := ref.Manifest(ctx)
	if err != nil {
		return "", nil, nil, errors.Wrapf(err, "failed to obtain the manifest of image %q", imageID)
	}
	if manifestFormat == "" && len(manifestBytes) > 0 {
		manifestFormat = manifest.GuessMIMEType(manifestBytes)
	}
	return manifestFormat, oci.History, oci.RootFS.DiffIDs, nil
}

// PopulateLayer scans the local images and adds them to the map.
func (llp *LocalLayerProvider) PopulateLayer(ctx context.Context, topLayer string) error {
	llp.localLayersLock.Lock()
	defer llp.localLayersLock.Unlock()

	// Get the list of images available in the image store
	images, err := llp.store.Images()
	if err != nil {
		return errors.Wrap(err, "failed to obtain the list of images from store")
	}

	for _, image := range images {
		manifestType, history, diffIDs, err := llp.getImageTypeAndHistoryAndDiffIDs(ctx, image.ID)
		if err != nil {
			// It's possible that this image is for another architecture, which results
			// in a custom-crafted error message that we'd have to use substring matching
			// to recognize.  Instead, ignore the image.
			logrus.Debugf("error getting history of %q (%v), ignoring it", image.ID, err)
			continue
		}

		var imageParentLayerID string
		if image.TopLayer != "" {
			// Figure out the top layer from this image

			// If we haven't added a layer, then our base
			// layer should be the same as the image's layer.
			if history[len(history)-1].EmptyLayer {
				imageParentLayerID = image.TopLayer
			} else {
				// If did add a layer, then our base layer should be the
				// same as the parent of the image's layer.
				imageTopLayer, err := llp.store.Layer(image.TopLayer)
				if err != nil {
					return errors.Wrapf(err, "failed to obtain the top layer info")
				}

				imageParentLayerID = imageTopLayer.Parent
			}
		}

		// If the parent of the top layer of an image is equal to the current build image's top layer,
		// it means that this image is potentially a cached intermediate image from a previous
		// build.
		// Note: this is just a micro optimization. The parent layer is reflected in the layerKey as well.
		if topLayer != imageParentLayerID {
			continue
		}

		layerKey := CalculateLocalLayerKey(manifestType, imageParentLayerID, history, diffIDs)

		llp.localLayers[layerKey] = image.ID
	}

	return nil
}

// Load returns the image id for the key.
func (llp *LocalLayerProvider) Load(layerKey string) (string, error) {
	llp.localLayersLock.Lock()
	defer llp.localLayersLock.Unlock()

	return llp.localLayers[layerKey], nil
}

// Store returns the image id for the key.
func (llp *LocalLayerProvider) Store(string, string) error {
	// This is noop. The build process already have the layer stored
	return nil
}
