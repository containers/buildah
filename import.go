package buildah

import (
	"fmt"

	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
)

func importBuilderDataFromImage(store storage.Store, systemContext *types.SystemContext, imageID, containerName, containerID string) (*Builder, error) {
	manifest := []byte{}
	config := []byte{}
	imageName := ""

	if imageID != "" {
		ref, err := is.Transport.ParseStoreReference(store, "@"+imageID)
		if err != nil {
			return nil, fmt.Errorf("no such image %q: %v", "@"+imageID, err)
		}
		src, err2 := ref.NewImage(systemContext)
		if err2 != nil {
			return nil, fmt.Errorf("error instantiating image: %v", err2)
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
		if img, err3 := store.Image(imageID); err3 == nil {
			if len(img.Names) > 0 {
				imageName = img.Names[0]
			}
		}
	}

	builder := &Builder{
		store:            store,
		Type:             containerType,
		FromImage:        imageName,
		FromImageID:      imageID,
		Config:           config,
		Manifest:         manifest,
		Container:        containerName,
		ContainerID:      containerID,
		ImageAnnotations: map[string]string{},
		ImageCreatedBy:   "",
	}

	builder.initConfig()

	return builder, nil
}

func importBuilder(store storage.Store, options ImportOptions) (*Builder, error) {
	if options.Container == "" {
		return nil, fmt.Errorf("container name must be specified")
	}

	c, err := store.Container(options.Container)
	if err != nil {
		return nil, err
	}

	systemContext := getSystemContext(options.SignaturePolicyPath)

	builder, err := importBuilderDataFromImage(store, systemContext, c.ImageID, options.Container, c.ID)
	if err != nil {
		return nil, err
	}

	err = builder.Save()
	if err != nil {
		return nil, fmt.Errorf("error saving builder state: %v", err)
	}

	return builder, nil
}

func importBuilderFromImage(store storage.Store, options ImportFromImageOptions) (*Builder, error) {
	if options.Image == "" {
		return nil, fmt.Errorf("image name must be specified")
	}

	ref, err := is.Transport.ParseStoreReference(store, options.Image)
	if err != nil {
		return nil, fmt.Errorf("error parsing reference to image %q: %v", options.Image, err)
	}
	img, err := is.Transport.GetStoreImage(store, ref)
	if err != nil {
		return nil, fmt.Errorf("unable to locate image: %v", err)
	}

	systemContext := getSystemContext(options.SignaturePolicyPath)

	builder, err := importBuilderDataFromImage(store, systemContext, img.ID, "", "")
	if err != nil {
		return nil, err
	}

	return builder, nil
}
