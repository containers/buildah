package buildah

import (
	"fmt"

	is "github.com/containers/image/storage"
	"github.com/containers/storage/storage"
)

func importBuilder(store storage.Store, options ImportOptions) (*Builder, error) {
	manifest := []byte{}
	config := []byte{}
	name := ""
	image := ""

	if options.Container == "" {
		return nil, fmt.Errorf("container name must be specified")
	}

	c, err := store.GetContainer(options.Container)
	if err != nil {
		return nil, err
	}

	systemContext := getSystemContext(options.SignaturePolicyPath)

	if c.ImageID != "" {
		ref, err := is.Transport.ParseStoreReference(store, "@"+c.ImageID)
		if err != nil {
			return nil, fmt.Errorf("no such image %q: %v", "@"+c.ImageID, err)
		}
		src, err := ref.NewImage(systemContext)
		if err != nil {
			return nil, fmt.Errorf("error instantiating image: %v", err)
		}
		defer src.Close()
		config, err = src.ConfigBlob()
		if err != nil {
			return nil, fmt.Errorf("error reading image configuration: %v", err)
		}
		manifest, _, err = src.Manifest()
		if err != nil {
			return nil, fmt.Errorf("error reading image manifest: %v", err)
		}
	}

	name = options.Container

	builder := &Builder{
		store:       store,
		Type:        containerType,
		FromImage:   image,
		Config:      config,
		Manifest:    manifest,
		Container:   name,
		ContainerID: c.ID,
		Mounts:      []string{},
		Links:       []string{},
		Annotations: map[string]string{},
		Env:         []string{},
		Cmd:         []string{},
		Entrypoint:  []string{},
		Expose:      map[string]interface{}{},
		Labels:      map[string]string{},
		Volumes:     []string{},
		Arg:         map[string]string{},
	}

	err = builder.Save()
	if err != nil {
		return nil, fmt.Errorf("error saving builder state: %v", err)
	}

	return builder, nil
}
