package libimage

import (
	"context"

	libimageTypes "github.com/containers/common/libimage/types"
	"github.com/containers/storage"
)

// History computes the image history of the image including all of its parents.
func (i *Image) History(ctx context.Context) ([]libimageTypes.ImageHistory, error) {
	ociImage, err := i.toOCI(ctx)
	if err != nil {
		return nil, err
	}

	layerTree, err := i.runtime.layerTree()
	if err != nil {
		return nil, err
	}

	var allHistory []libimageTypes.ImageHistory
	var layer *storage.Layer
	if i.TopLayer() != "" {
		layer, err = i.runtime.store.Layer(i.TopLayer())
		if err != nil {
			return nil, err
		}
	}

	// Iterate in reverse order over the history entries, and lookup the
	// corresponding image ID, size and get the next later if needed.
	numHistories := len(ociImage.History) - 1
	usedIDs := make(map[string]bool) // prevents assigning images IDs more than once
	for x := numHistories; x >= 0; x-- {
		history := libimageTypes.ImageHistory{
			ID:        "<missing>", // may be overridden below
			Created:   ociImage.History[x].Created,
			CreatedBy: ociImage.History[x].CreatedBy,
			Comment:   ociImage.History[x].Comment,
		}

		if layer != nil {
			history.Tags = layer.Names
			if !ociImage.History[x].EmptyLayer {
				history.Size = layer.UncompressedSize
			}
			// Query the layer tree if it's the top layer of an
			// image.
			node := layerTree.node(layer.ID)
			if len(node.images) > 0 {
				id := node.images[0].ID() // always use the first one
				if _, used := usedIDs[id]; !used {
					history.ID = id
					usedIDs[id] = true
				}
			}
			if layer.Parent != "" && !ociImage.History[x].EmptyLayer {
				layer, err = i.runtime.store.Layer(layer.Parent)
				if err != nil {
					return nil, err
				}
			}
		}

		allHistory = append(allHistory, history)
	}

	return allHistory, nil
}
