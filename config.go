package buildah

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	digest "github.com/opencontainers/go-digest"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/docker"
)

// makeOCIv1Image builds the best OCIv1 image structure we can from the
// contents of the docker image structure.
func makeOCIv1Image(dimage *docker.V2Image) (ociv1.Image, error) {
	config := dimage.Config
	if config == nil {
		config = &dimage.ContainerConfig
	}
	image := ociv1.Image{
		Created:      dimage.Created.UTC(),
		Author:       dimage.Author,
		Architecture: dimage.Architecture,
		OS:           dimage.OS,
		Config: ociv1.ImageConfig{
			User:         config.User,
			ExposedPorts: map[string]struct{}{},
			Env:          config.Env,
			Entrypoint:   config.Entrypoint,
			Cmd:          config.Cmd,
			Volumes:      config.Volumes,
			WorkingDir:   config.WorkingDir,
			Labels:       config.Labels,
		},
		RootFS: ociv1.RootFS{
			Type:    "",
			DiffIDs: []string{},
		},
		History: []ociv1.History{},
	}
	for port, what := range config.ExposedPorts {
		image.Config.ExposedPorts[string(port)] = what
	}
	RootFS := docker.V2S2RootFS{}
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
		ohistory := ociv1.History{
			Created:    history.Created.UTC(),
			CreatedBy:  history.CreatedBy,
			Author:     history.Author,
			Comment:    history.Comment,
			EmptyLayer: history.EmptyLayer,
		}
		image.History = append(image.History, ohistory)
	}
	return image, nil
}

// makeDockerV2S2Image builds the best docker image structure we can from the
// contents of the OCI image structure.
func makeDockerV2S2Image(oimage *ociv1.Image) (docker.V2Image, error) {
	image := docker.V2Image{
		V1Image: docker.V1Image{Created: oimage.Created.UTC(),
			Author:       oimage.Author,
			Architecture: oimage.Architecture,
			OS:           oimage.OS,
			ContainerConfig: docker.Config{
				User:         oimage.Config.User,
				ExposedPorts: docker.PortSet{},
				Env:          oimage.Config.Env,
				Entrypoint:   oimage.Config.Entrypoint,
				Cmd:          oimage.Config.Cmd,
				Volumes:      oimage.Config.Volumes,
				WorkingDir:   oimage.Config.WorkingDir,
				Labels:       oimage.Config.Labels,
			},
		},
		RootFS: &docker.V2S2RootFS{
			Type:    "",
			DiffIDs: []digest.Digest{},
		},
		History: []docker.V2S2History{},
	}
	for port, what := range oimage.Config.ExposedPorts {
		image.ContainerConfig.ExposedPorts[docker.Port(port)] = what
	}
	if oimage.RootFS.Type == docker.TypeLayers {
		image.RootFS.Type = docker.TypeLayers
		for _, id := range oimage.RootFS.DiffIDs {
			d, err := digest.Parse(id)
			if err != nil {
				return docker.V2Image{}, err
			}
			image.RootFS.DiffIDs = append(image.RootFS.DiffIDs, d)
		}
	}
	for _, history := range oimage.History {
		dhistory := docker.V2S2History{
			Created:    history.Created.UTC(),
			CreatedBy:  history.CreatedBy,
			Author:     history.Author,
			Comment:    history.Comment,
			EmptyLayer: history.EmptyLayer,
		}
		image.History = append(image.History, dhistory)
	}
	image.Config = &image.ContainerConfig
	return image, nil
}

// makeDockerV2S1Image builds the best docker image structure we can from the
// contents of the V2S1 image structure.
func makeDockerV2S1Image(manifest docker.V2S1Manifest) (docker.V2Image, error) {
	// Treat the most recent (first) item in the history as a description of the image.
	if len(manifest.History) == 0 {
		return docker.V2Image{}, errors.Errorf("error parsing image configuration from manifest")
	}
	dimage := docker.V2Image{}
	err := json.Unmarshal([]byte(manifest.History[0].V1Compatibility), &dimage)
	if err != nil {
		return docker.V2Image{}, err
	}
	if dimage.DockerVersion == "" {
		return docker.V2Image{}, errors.Errorf("error parsing image configuration from history")
	}
	// The DiffID list is intended to contain the sums of _uncompressed_ blobs, and these are most
	// likely compressed, so leave the list empty to avoid potential confusion later on.  We can
	// construct a list with the correct values when we prep layers for pushing, so we don't lose.
	// information by leaving this part undone.
	rootFS := &docker.V2S2RootFS{
		Type:    docker.TypeLayers,
		DiffIDs: []digest.Digest{},
	}
	// Build a filesystem history.
	history := []docker.V2S2History{}
	for i := range manifest.History {
		h := docker.V2S2History{
			Created:    time.Now().UTC(),
			Author:     "",
			CreatedBy:  "",
			Comment:    "",
			EmptyLayer: false,
		}
		dcompat := docker.V1Compatibility{}
		if err2 := json.Unmarshal([]byte(manifest.History[i].V1Compatibility), &dcompat); err2 == nil {
			h.Created = dcompat.Created.UTC()
			h.Author = dcompat.Author
			h.Comment = dcompat.Comment
			if len(dcompat.ContainerConfig.Cmd) > 0 {
				h.CreatedBy = fmt.Sprintf("%v", dcompat.ContainerConfig.Cmd)
			}
			h.EmptyLayer = dcompat.ThrowAway
		}
		// Prepend this layer to the list, because a v2s1 format manifest's list is in reverse order
		// compared to v2s2, which lists earlier layers before later ones.
		history = append([]docker.V2S2History{h}, history...)
	}
	dimage.RootFS = rootFS
	dimage.History = history
	return dimage, nil
}

func (b *Builder) initConfig() {
	image := ociv1.Image{}
	dimage := docker.V2Image{}
	if len(b.Config) > 0 {
		// Try to parse the image configuration. If we fail start over from scratch.
		if err := json.Unmarshal(b.Config, &dimage); err == nil && dimage.DockerVersion != "" {
			if image, err = makeOCIv1Image(&dimage); err != nil {
				image = ociv1.Image{}
			}
		} else {
			if err := json.Unmarshal(b.Config, &image); err != nil {
				if dimage, err = makeDockerV2S2Image(&image); err != nil {
					dimage = docker.V2Image{}
				}
			}
		}
		b.OCIv1 = image
		b.Docker = dimage
	} else {
		// Try to dig out the image configuration from the manifest.
		manifest := docker.V2S1Manifest{}
		if err := json.Unmarshal(b.Manifest, &manifest); err == nil && manifest.SchemaVersion == 1 {
			if dimage, err = makeDockerV2S1Image(manifest); err == nil {
				if image, err = makeOCIv1Image(&dimage); err != nil {
					image = ociv1.Image{}
				}
			}
		}
		b.OCIv1 = image
		b.Docker = dimage
	}
	if len(b.Manifest) > 0 {
		// Attempt to recover format-specific data from the manifest.
		v1Manifest := ociv1.Manifest{}
		if json.Unmarshal(b.Manifest, &v1Manifest) == nil {
			b.ImageAnnotations = v1Manifest.Annotations
		}
	}
	b.fixupConfig()
}

func (b *Builder) fixupConfig() {
	if b.Docker.Config != nil {
		// Prefer image-level settings over those from the container it was built from.
		b.Docker.ContainerConfig = *b.Docker.Config
	}
	b.Docker.Config = &b.Docker.ContainerConfig
	b.Docker.DockerVersion = ""
	if b.FromImageID != "" {
		if d, err := digest.Parse(b.FromImageID); err == nil {
			b.Docker.Parent = docker.ID(d)
		} else {
			b.Docker.Parent = docker.ID(digest.NewDigestFromHex(digest.Canonical.String(), b.FromImageID))
		}
	}
	now := time.Now().UTC()
	if b.Docker.Created.IsZero() {
		b.Docker.Created = now
	}
	if b.FromImage != "" {
		b.Docker.Config.Image = b.FromImage
	}
	if b.OCIv1.Created.IsZero() {
		b.OCIv1.Created = now
	}
	if b.OS() == "" {
		b.SetOS(runtime.GOOS)
	}
	if b.Architecture() == "" {
		b.SetArchitecture(runtime.GOARCH)
	}
	if b.WorkDir() == "" {
		b.SetWorkDir(string(filepath.Separator))
	}
}

// Annotations returns a set of key-value pairs from the image's manifest.
func (b *Builder) Annotations() map[string]string {
	return copyStringStringMap(b.ImageAnnotations)
}

// SetAnnotation adds or overwrites a key's value from the image's manifest.
// Note: this setting is not present in the Docker v2 image format, so it is
// discarded when writing images using Docker v2 formats.
func (b *Builder) SetAnnotation(key, value string) {
	if b.ImageAnnotations == nil {
		b.ImageAnnotations = map[string]string{}
	}
	b.ImageAnnotations[key] = value
}

// UnsetAnnotation removes a key and its value from the image's manifest, if
// it's present.
func (b *Builder) UnsetAnnotation(key string) {
	delete(b.ImageAnnotations, key)
}

// ClearAnnotations removes all keys and their values from the image's
// manifest.
func (b *Builder) ClearAnnotations() {
	b.ImageAnnotations = map[string]string{}
}

// CreatedBy returns a description of how this image was built.
func (b *Builder) CreatedBy() string {
	return b.ImageCreatedBy
}

// SetCreatedBy sets the description of how this image was built.
func (b *Builder) SetCreatedBy(how string) {
	b.ImageCreatedBy = how
}

// OS returns a name of the OS on which the container, or a container built
// using an image built from this container, is intended to be run.
func (b *Builder) OS() string {
	return b.OCIv1.OS
}

// SetOS sets the name of the OS on which the container, or a container built
// using an image built from this container, is intended to be run.
func (b *Builder) SetOS(os string) {
	b.OCIv1.OS = os
	b.Docker.OS = os
}

// Architecture returns a name of the architecture on which the container, or a
// container built using an image built from this container, is intended to be
// run.
func (b *Builder) Architecture() string {
	return b.OCIv1.Architecture
}

// SetArchitecture sets the name of the architecture on which the container, or
// a container built using an image built from this container, is intended to
// be run.
func (b *Builder) SetArchitecture(arch string) {
	b.OCIv1.Architecture = arch
	b.Docker.Architecture = arch
}

// Maintainer returns contact information for the person who built the image.
func (b *Builder) Maintainer() string {
	return b.OCIv1.Author
}

// SetMaintainer sets contact information for the person who built the image.
func (b *Builder) SetMaintainer(who string) {
	b.OCIv1.Author = who
	b.Docker.Author = who
}

// User returns information about the user as whom the container, or a
// container built using an image built from this container, should be run.
func (b *Builder) User() string {
	return b.OCIv1.Config.User
}

// SetUser sets information about the user as whom the container, or a
// container built using an image built from this container, should be run.
// Acceptable forms are a user name or ID, optionally followed by a colon and a
// group name or ID.
func (b *Builder) SetUser(spec string) {
	b.OCIv1.Config.User = spec
	b.Docker.Config.User = spec
}

// WorkDir returns the default working directory for running commands in the
// container, or in a container built using an image built from this container.
func (b *Builder) WorkDir() string {
	return b.OCIv1.Config.WorkingDir
}

// SetWorkDir sets the location of the default working directory for running
// commands in the container, or in a container built using an image built from
// this container.
func (b *Builder) SetWorkDir(there string) {
	b.OCIv1.Config.WorkingDir = there
	b.Docker.Config.WorkingDir = there
}

// Env returns a list of key-value pairs to be set when running commands in the
// container, or in a container built using an image built from this container.
func (b *Builder) Env() []string {
	return copyStringSlice(b.OCIv1.Config.Env)
}

// SetEnv adds or overwrites a value to the set of environment strings which
// should be set when running commands in the container, or in a container
// built using an image built from this container.
func (b *Builder) SetEnv(k string, v string) {
	reset := func(s *[]string) {
		n := []string{}
		for i := range *s {
			if !strings.HasPrefix((*s)[i], k+"=") {
				n = append(n, (*s)[i])
			}
		}
		n = append(n, k+"="+v)
		*s = n
	}
	reset(&b.OCIv1.Config.Env)
	reset(&b.Docker.Config.Env)
}

// UnsetEnv removes a value from the set of environment strings which should be
// set when running commands in this container, or in a container built using
// an image built from this container.
func (b *Builder) UnsetEnv(k string) {
	unset := func(s *[]string) {
		n := []string{}
		for i := range *s {
			if !strings.HasPrefix((*s)[i], k+"=") {
				n = append(n, (*s)[i])
			}
		}
		*s = n
	}
	unset(&b.OCIv1.Config.Env)
	unset(&b.Docker.Config.Env)
}

// ClearEnv removes all values from the set of environment strings which should
// be set when running commands in this container, or in a container built
// using an image built from this container.
func (b *Builder) ClearEnv() {
	b.OCIv1.Config.Env = []string{}
	b.Docker.Config.Env = []string{}
}

// Cmd returns the default command, or command parameters if an Entrypoint is
// set, to use when running a container built from an image built from this
// container.
func (b *Builder) Cmd() []string {
	return copyStringSlice(b.OCIv1.Config.Cmd)
}

// SetCmd sets the default command, or command parameters if an Entrypoint is
// set, to use when running a container built from an image built from this
// container.
func (b *Builder) SetCmd(cmd []string) {
	b.OCIv1.Config.Cmd = copyStringSlice(cmd)
	b.Docker.Config.Cmd = copyStringSlice(cmd)
}

// Entrypoint returns the command to be run for containers built from images
// built from this container.
func (b *Builder) Entrypoint() []string {
	return copyStringSlice(b.OCIv1.Config.Entrypoint)
}

// SetEntrypoint sets the command to be run for in containers built from images
// built from this container.
func (b *Builder) SetEntrypoint(ep []string) {
	b.OCIv1.Config.Entrypoint = copyStringSlice(ep)
	b.Docker.Config.Entrypoint = copyStringSlice(ep)
}

// Labels returns a set of key-value pairs from the image's runtime
// configuration.
func (b *Builder) Labels() map[string]string {
	return copyStringStringMap(b.OCIv1.Config.Labels)
}

// SetLabel adds or overwrites a key's value from the image's runtime
// configuration.
func (b *Builder) SetLabel(k string, v string) {
	if b.OCIv1.Config.Labels == nil {
		b.OCIv1.Config.Labels = map[string]string{}
	}
	b.OCIv1.Config.Labels[k] = v
	if b.Docker.Config.Labels == nil {
		b.Docker.Config.Labels = map[string]string{}
	}
	b.Docker.Config.Labels[k] = v
}

// UnsetLabel removes a key and its value from the image's runtime
// configuration, if it's present.
func (b *Builder) UnsetLabel(k string) {
	delete(b.OCIv1.Config.Labels, k)
	delete(b.Docker.Config.Labels, k)
}

// ClearLabels removes all keys and their values from the image's runtime
// configuration.
func (b *Builder) ClearLabels() {
	b.OCIv1.Config.Labels = map[string]string{}
	b.Docker.Config.Labels = map[string]string{}
}

// Ports returns the set of ports which should be exposed when a container
// based on an image built from this container is run.
func (b *Builder) Ports() []string {
	p := []string{}
	for k := range b.OCIv1.Config.ExposedPorts {
		p = append(p, k)
	}
	return p
}

// SetPort adds or overwrites an exported port in the set of ports which should
// be exposed when a container based on an image built from this container is
// run.
func (b *Builder) SetPort(p string) {
	if b.OCIv1.Config.ExposedPorts == nil {
		b.OCIv1.Config.ExposedPorts = map[string]struct{}{}
	}
	b.OCIv1.Config.ExposedPorts[p] = struct{}{}
	if b.Docker.Config.ExposedPorts == nil {
		b.Docker.Config.ExposedPorts = make(docker.PortSet)
	}
	b.Docker.Config.ExposedPorts[docker.Port(p)] = struct{}{}
}

// UnsetPort removes an exposed port from the set of ports which should be
// exposed when a container based on an image built from this container is run.
func (b *Builder) UnsetPort(p string) {
	delete(b.OCIv1.Config.ExposedPorts, p)
	delete(b.Docker.Config.ExposedPorts, docker.Port(p))
}

// ClearPorts empties the set of ports which should be exposed when a container
// based on an image built from this container is run.
func (b *Builder) ClearPorts() {
	b.OCIv1.Config.ExposedPorts = map[string]struct{}{}
	b.Docker.Config.ExposedPorts = docker.PortSet{}
}

// Volumes returns a list of filesystem locations which should be mounted from
// outside of the container when a container built from an image built from
// this container is run.
func (b *Builder) Volumes() []string {
	v := []string{}
	for k := range b.OCIv1.Config.Volumes {
		v = append(v, k)
	}
	return v
}

// AddVolume adds a location to the image's list of locations which should be
// mounted from outside of the container when a container based on an image
// built from this container is run.
func (b *Builder) AddVolume(v string) {
	if b.OCIv1.Config.Volumes == nil {
		b.OCIv1.Config.Volumes = map[string]struct{}{}
	}
	b.OCIv1.Config.Volumes[v] = struct{}{}
	if b.Docker.Config.Volumes == nil {
		b.Docker.Config.Volumes = map[string]struct{}{}
	}
	b.Docker.Config.Volumes[v] = struct{}{}
}

// RemoveVolume removes a location from the list of locations which should be
// mounted from outside of the container when a container based on an image
// built from this container is run.
func (b *Builder) RemoveVolume(v string) {
	delete(b.OCIv1.Config.Volumes, v)
	delete(b.Docker.Config.Volumes, v)
}

// ClearVolumes removes all locations from the image's list of locations which
// should be mounted from outside of the container when a container based on an
// image built from this container is run.
func (b *Builder) ClearVolumes() {
	b.OCIv1.Config.Volumes = map[string]struct{}{}
	b.Docker.Config.Volumes = map[string]struct{}{}
}

// Hostname returns the hostname which will be set in the container and in
// containers built using images built from the container.
func (b *Builder) Hostname() string {
	return b.Docker.Config.Hostname
}

// SetHostname sets the hostname which will be set in the container and in
// containers built using images built from the container.
// Note: this setting is not present in the OCIv1 image format, so it is
// discarded when writing images using OCIv1 formats.
func (b *Builder) SetHostname(name string) {
	b.Docker.Config.Hostname = name
}

// Domainname returns the domainname which will be set in the container and in
// containers built using images built from the container.
func (b *Builder) Domainname() string {
	return b.Docker.Config.Domainname
}

// SetDomainname sets the domainname which will be set in the container and in
// containers built using images built from the container.
// Note: this setting is not present in the OCIv1 image format, so it is
// discarded when writing images using OCIv1 formats.
func (b *Builder) SetDomainname(name string) {
	b.Docker.Config.Domainname = name
}
