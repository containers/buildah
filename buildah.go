package buildah

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/containers/storage/storage"
)

const (
	Package       = "buildah"
	ContainerType = Package + " 0.0.0"
)

type Builder struct {
	store storage.Store

	Type      string
	FromImage string
	Config    []byte
	Manifest  []byte

	Container   string
	ContainerID string
	MountPoint  string
	Mounts      []string
	Links       []string
	Annotations map[string]string

	CreatedBy    string
	OS           string
	Architecture string
	Maintainer   string
	User         string
	Workdir      string
	Env          []string
	Cmd          []string
	Entrypoint   []string
	Expose       map[string]interface{}
	Labels       map[string]string
	Volumes      []string
	Arg          map[string]string
}

type BuilderOptions struct {
	FromImage           string
	Container           string
	PullIfMissing       bool
	PullAlways          bool
	Mount               bool
	Link                string
	Registry            string
	SignaturePolicyPath string
}

func NewBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	return newBuilder(store, options)
}

func OpenBuilder(store storage.Store, container string) (*Builder, error) {
	c, err := store.GetContainer(container)
	if err != nil {
		return nil, err
	}
	buildstate, err := store.GetMetadata(c.ID)
	if err != nil {
		return nil, err
	}
	b := &Builder{}
	err = json.Unmarshal([]byte(buildstate), &b)
	if err != nil {
		return nil, err
	}
	if b.Type != ContainerType {
		return nil, fmt.Errorf("container is not a %s container", Package)
	}
	b.store = store
	return b, nil
}

func OpenBuilderByPath(store storage.Store, path string) (*Builder, error) {
	containers, err := store.Containers()
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	builderMatchesPath := func(b *Builder, path string) bool {
		if b.MountPoint == path {
			return true
		}
		for _, m := range b.Mounts {
			if m == path {
				return true
			}
		}
		for _, l := range b.Links {
			if l == path {
				return true
			}
		}
		return false
	}
	for _, container := range containers {
		buildstate, err := store.GetMetadata(container.ID)
		if err != nil {
			return nil, err
		}
		b := &Builder{}
		err = json.Unmarshal([]byte(buildstate), &b)
		if err == nil && b.Type == ContainerType && builderMatchesPath(b, abs) {
			b.store = store
			return b, nil
		}
	}
	return nil, storage.ErrContainerUnknown
}

func (b *Builder) Save() error {
	buildstate, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return b.store.SetMetadata(b.ContainerID, string(buildstate))
}
