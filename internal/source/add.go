package source

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/v5/types"
	"github.com/containers/storage/pkg/archive"
	specV1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// AddOptions include data to alter certain knobs when adding a source artifact
// to a source image.
type AddOptions struct {
	// Annotations for the source artifact.
	Annotations []string
}

// annotations parses the specified annotations and transforms them into a map.
// A given annotation can be specified only once.
func (o *AddOptions) annotations() (map[string]string, error) {
	annotations := make(map[string]string)

	for _, unparsed := range o.Annotations {
		parsed := strings.SplitN(unparsed, "=", 2)
		if len(parsed) != 2 {
			return nil, errors.Errorf("invalid annotation %q (expected format is \"key=value\")", unparsed)
		}
		if _, exists := annotations[parsed[0]]; exists {
			return nil, errors.Errorf("annotation %q specified more than once", parsed[0])
		}
		annotations[parsed[0]] = parsed[1]
	}

	return annotations, nil
}

// Add adds the specified source artifact at `artifactPath` to the source image
// at `sourcePath`.  Note that the artifact will be added as a gzip-compressed
// tar ball.  Add attempts to auto-tar and auto-compress only if necessary.
func Add(ctx context.Context, sourcePath string, artifactPath string, options AddOptions) error {
	// Let's first make sure `sourcePath` exists and that we can access it.
	if _, err := os.Stat(sourcePath); err != nil {
		return err
	}

	annotations, err := options.annotations()
	if err != nil {
		return err
	}

	ociDest, err := openOrCreateSourceImage(ctx, sourcePath)
	if err != nil {
		return err
	}
	defer ociDest.Close()

	tarStream, err := archive.TarWithOptions(artifactPath, &archive.TarOptions{Compression: archive.Gzip})
	if err != nil {
		return errors.Wrap(err, "error creating compressed tar stream")
	}

	info := types.BlobInfo{
		Size: -1, // "unknown": we'll get that information *after* adding
	}
	addedBlob, err := ociDest.PutBlob(ctx, tarStream, info, nil, false)
	if err != nil {
		return errors.Wrap(err, "error adding source artifact")
	}

	// Add the new layers to the source image's manifest.
	manifest, oldManifestDigest, _, err := readManifestFromOCIPath(ctx, sourcePath)
	if err != nil {
		return err
	}
	manifest.Layers = append(manifest.Layers,
		specV1.Descriptor{
			MediaType:   specV1.MediaTypeImageLayerGzip,
			Digest:      addedBlob.Digest,
			Size:        addedBlob.Size,
			Annotations: annotations,
		},
	)
	manifestDigest, manifestSize, err := writeManifest(ctx, manifest, ociDest)
	if err != nil {
		return err
	}

	// Now, as we've written the updated manifest, we can delete the
	// previous one.  `types.ImageDestination` doesn't expose a high-level
	// API to manage multi-manifest destination, so we need to do it
	// manually.  Not an issue, since paths are predictable for an OCI
	// layout.
	if err := removeBlob(oldManifestDigest, sourcePath); err != nil {
		return errors.Wrap(err, "error removing old manifest")
	}

	manifestDescriptor := specV1.Descriptor{
		MediaType: specV1.MediaTypeImageManifest,
		Digest:    *manifestDigest,
		Size:      manifestSize,
	}
	if err := updateIndexWithNewManifestDescriptor(&manifestDescriptor, sourcePath); err != nil {
		return err
	}

	return nil
}

func updateIndexWithNewManifestDescriptor(manifest *specV1.Descriptor, sourcePath string) error {
	index := specV1.Index{}
	indexPath := filepath.Join(sourcePath, "index.json")

	rawData, err := ioutil.ReadFile(indexPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawData, &index); err != nil {
		return err
	}

	index.Manifests = []specV1.Descriptor{*manifest}
	rawData, err = json.Marshal(&index)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(indexPath, rawData, 0644)
}
