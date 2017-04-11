package buildah

import (
	"encoding/json"
	"runtime"
	"time"

	"github.com/Sirupsen/logrus"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/projectatomic/buildah/docker"
)

func copyDockerImageConfig(dimage *docker.Image) (ociv1.Image, error) {
	image := ociv1.Image{
		Created:      dimage.Created.UTC(),
		Author:       dimage.Author,
		Architecture: dimage.Architecture,
		OS:           dimage.OS,
		Config: ociv1.ImageConfig{
			User:         dimage.Config.User,
			ExposedPorts: map[string]struct{}{},
			Env:          dimage.Config.Env,
			Entrypoint:   dimage.Config.Entrypoint,
			Cmd:          dimage.Config.Cmd,
			Volumes:      dimage.Config.Volumes,
			WorkingDir:   dimage.Config.WorkingDir,
			Labels:       dimage.Config.Labels,
		},
		RootFS: ociv1.RootFS{
			Type:    "",
			DiffIDs: []string{},
		},
		History: []ociv1.History{},
	}
	for port, what := range dimage.Config.ExposedPorts {
		image.Config.ExposedPorts[string(port)] = what
	}
	RootFS := docker.RootFS{}
	if dimage.RootFS != nil {
		RootFS = *dimage.RootFS
	}
	if RootFS.Type == docker.TypeLayers {
		image.RootFS.Type = docker.TypeLayers
		for _, id := range RootFS.DiffIDs {
			image.RootFS.DiffIDs = append(image.RootFS.DiffIDs, id.String())
		}
	}
	for _, history := range dimage.History {
		created := history.Created.UTC()
		ohistory := ociv1.History{
			Created:    created,
			CreatedBy:  history.CreatedBy,
			Author:     history.Author,
			Comment:    history.Comment,
			EmptyLayer: history.EmptyLayer,
		}
		image.History = append(image.History, ohistory)
	}
	return image, nil
}

func (b *Builder) updatedConfig() []byte {
	image := ociv1.Image{}
	dimage := docker.Image{}
	if len(b.Config) > 0 {
		// Try to parse the image configuration. If we fail start over from scratch.
		if err := json.Unmarshal(b.Config, &dimage); err == nil && dimage.DockerVersion != "" {
			if image, err = copyDockerImageConfig(&dimage); err != nil {
				image = ociv1.Image{}
			}
		} else {
			if err := json.Unmarshal(b.Config, &image); err != nil {
				image = ociv1.Image{}
			}
		}
	}
	image.Created = time.Now().UTC()
	if image.Architecture == "" {
		image.Architecture = runtime.GOARCH
	}
	if image.OS == "" {
		image.OS = runtime.GOOS
	}
	if b.Architecture != "" {
		image.Architecture = b.Architecture
	}
	if b.OS != "" {
		image.OS = b.OS
	}
	if b.Maintainer != "" {
		image.Author = b.Maintainer
	}
	if b.User != "" {
		image.Config.User = b.User
	}
	if len(b.Volumes) > 0 {
		for _, volSpec := range b.Volumes {
			image.Config.Volumes[volSpec] = struct{}{}
		}
	}
	if b.Workdir != "" {
		image.Config.WorkingDir = b.Workdir
	}
	if len(b.Env) > 0 {
		image.Config.Env = append(image.Config.Env, b.Env...)
	}
	if len(b.Cmd) > 0 {
		image.Config.Cmd = b.Cmd
	}
	if len(b.Entrypoint) > 0 {
		image.Config.Entrypoint = b.Entrypoint
	}
	if len(b.Expose) > 0 {
		if image.Config.ExposedPorts == nil {
			image.Config.ExposedPorts = make(map[string]struct{})
		}
		for k := range b.Expose {
			image.Config.ExposedPorts[k] = struct{}{}
		}
	}
	if len(b.Labels) > 0 {
		if image.Config.Labels == nil {
			image.Config.Labels = make(map[string]string)
		}
		for k, v := range b.Labels {
			image.Config.Labels[k] = v
		}
	}
	updatedImageConfig, err := json.Marshal(&image)
	if err != nil {
		logrus.Errorf("error exporting updated image configuration, using original configuration")
		return b.Config
	}
	return updatedImageConfig
}

// UpdatedEnv returns the environment list from the source image, with the
// builder's own list appended to it.
func (b *Builder) UpdatedEnv() []string {
	config := b.updatedConfig()
	image := ociv1.Image{}
	if err := json.Unmarshal(config, &image); err != nil {
		logrus.Errorf("error parsing updated image information")
		return []string{}
	}
	return append(image.Config.Env, b.Env...)
}
