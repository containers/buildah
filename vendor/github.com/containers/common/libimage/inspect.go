package libimage

import (
	"context"
	"encoding/json"

	libimageTypes "github.com/containers/common/libimage/types"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/types"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// Inspect inspects the image.  Use `withSize` to also perform the
// comparatively expensive size computation of the image.
func (i *Image) Inspect(ctx context.Context, withSize bool) (*libimageTypes.ImageData, error) {
	logrus.Debugf("Inspecting image %s", i.ID())

	if i.cached.completeInspectData != nil {
		return i.cached.completeInspectData, nil
	}

	// First assemble data that does not depend on the format of the image.
	info, err := i.inspectInfo(ctx)
	if err != nil {
		return nil, err
	}
	ociImage, err := i.toOCI(ctx)
	if err != nil {
		return nil, err
	}
	parentImage, err := i.Parent(ctx)
	if err != nil {
		return nil, err
	}
	repoTags, err := i.RepoTags()
	if err != nil {
		return nil, err
	}
	repoDigests, err := i.RepoDigests()
	if err != nil {
		return nil, err
	}
	driverData, err := i.driverData()
	if err != nil {
		return nil, err
	}

	size := int64(-1)
	if withSize {
		size, err = i.Size()
		if err != nil {
			return nil, err
		}
	}

	data := &libimageTypes.ImageData{
		ID:           i.ID(),
		RepoTags:     repoTags,
		RepoDigests:  repoDigests,
		Created:      ociImage.Created,
		Author:       ociImage.Author,
		Architecture: ociImage.Architecture,
		Os:           ociImage.OS,
		Config:       &ociImage.Config,
		Version:      info.DockerVersion,
		Size:         size,
		VirtualSize:  size, // TODO: they should be different (inherited from Podman)
		Digest:       i.Digest(),
		Labels:       info.Labels,
		RootFS: &libimageTypes.RootFS{
			Type:   ociImage.RootFS.Type,
			Layers: ociImage.RootFS.DiffIDs,
		},
		GraphDriver:  driverData,
		User:         ociImage.Config.User,
		History:      ociImage.History,
		NamesHistory: i.NamesHistory(),
	}

	if parentImage != nil {
		data.Parent = parentImage.ID()
	}

	// Determine the format of the image.  How we determine certain data
	// depends on the format (e.g., Docker v2s2, OCI v1).
	src, err := i.source(ctx)
	if err != nil {
		return nil, err
	}
	manifestRaw, manifestType, err := src.GetManifest(ctx, nil)
	if err != nil {
		return nil, err
	}

	data.ManifestType = manifestType

	switch manifestType {
	// OCI image
	case ociv1.MediaTypeImageManifest:
		var ociManifest ociv1.Manifest
		if err := json.Unmarshal(manifestRaw, &ociManifest); err != nil {
			return nil, err
		}
		data.Annotations = ociManifest.Annotations
		if len(ociImage.History) > 0 {
			data.Comment = ociImage.History[0].Comment
		}

	// Docker image
	case manifest.DockerV2Schema2MediaType:
		var dockerManifest manifest.Schema2Image
		if err := json.Unmarshal(manifestRaw, &dockerManifest); err != nil {
			return nil, err
		}
		data.Comment = dockerManifest.Comment
		data.HealthCheck = dockerManifest.ContainerConfig.Healthcheck
	}

	i.cached.completeInspectData = data

	return data, nil
}

// inspectInfo returns the image inspect info.
func (i *Image) inspectInfo(ctx context.Context) (*types.ImageInspectInfo, error) {
	if i.cached.partialInspectData != nil {
		return i.cached.partialInspectData, nil
	}

	ref, err := i.StorageReference()
	if err != nil {

		return nil, err
	}

	img, err := ref.NewImage(ctx, &i.runtime.systemContext)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	data, err := img.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	i.cached.partialInspectData = data
	return data, nil
}
