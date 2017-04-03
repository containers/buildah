package buildah

import (
	"fmt"

	is "github.com/containers/image/storage"
	"github.com/containers/storage/storage"
)

func importBuilder(store storage.Store, options ImportOptions) (*Builder, error) {
	manifest := []byte{}
	config := []byte{}
	image := ""
	imageName := ""

	if options.Container == "" {
		return nil, fmt.Errorf("container name must be specified")
	}

	c, err := store.GetContainer(options.Container)
	if err != nil {
		return nil, err
	}

	systemContext := getSystemContext(options.SignaturePolicyPath)

	if c.ImageID != "" {
		ref, err2 := is.Transport.ParseStoreReference(store, "@"+c.ImageID)
		if err2 != nil {
			return nil, fmt.Errorf("no such image %q: %v", "@"+c.ImageID, err2)
		}
		src, err3 := ref.NewImage(systemContext)
		if err3 != nil {
			return nil, fmt.Errorf("error instantiating image: %v", err3)
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
		image = c.ImageID
		if img, err4 := store.GetImage(image); err4 == nil {
			if len(img.Names) > 0 {
				imageName = img.Names[0]
			}
		}
	}

	name := options.Container

	builder := &Builder{
		store:       store,
		Type:        containerType,
		FromImage:   imageName,
		FromImageID: image,
		Config:      config,
		Manifest:    manifest,
		Container:   name,
		ContainerID: c.ID,
		Mounts:      []string{},
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
