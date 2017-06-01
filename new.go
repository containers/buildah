package buildah

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	is "github.com/containers/image/storage"
	"github.com/containers/storage"
	"github.com/openshift/imagebuilder"
	"github.com/pkg/errors"
)

const (
	// BaseImageFakeName is the "name" of a source image which we interpret
	// as "no image".
	BaseImageFakeName = imagebuilder.NoBaseImageSpecifier
)

func newBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	var img *storage.Image
	manifest := []byte{}
	config := []byte{}

	name := "working-container"
	if options.FromImage == BaseImageFakeName {
		options.FromImage = ""
	}
	image := options.FromImage
	if options.Container != "" {
		name = options.Container
	} else {
		if image != "" {
			prefix := image
			s := strings.Split(prefix, "/")
			if len(s) > 0 {
				prefix = s[len(s)-1]
			}
			s = strings.Split(prefix, ":")
			if len(s) > 0 {
				prefix = s[0]
			}
			s = strings.Split(prefix, "@")
			if len(s) > 0 {
				prefix = s[0]
			}
			name = prefix + "-" + name
		}
	}
	if name != "" {
		var err error
		suffix := 1
		tmpName := name
		for err != storage.ErrContainerUnknown {
			_, err = store.Container(tmpName)
			if err == nil {
				suffix++
				tmpName = fmt.Sprintf("%s-%d", name, suffix)
			}
		}
		name = tmpName
	}

	systemContext := getSystemContext(options.SignaturePolicyPath)

	imageID := ""
	if image != "" {
		if options.PullPolicy == PullAlways {
			err := pullImage(store, options, systemContext)
			if err != nil {
				return nil, errors.Wrapf(err, "error pulling image %q", image)
			}
		}
		ref, err := is.Transport.ParseStoreReference(store, image)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing reference to image %q", image)
		}
		img, err = is.Transport.GetStoreImage(store, ref)
		if err != nil {
			if err == storage.ErrImageUnknown && options.PullPolicy != PullIfMissing {
				return nil, errors.Wrapf(err, "no such image %q", image)
			}
			err = pullImage(store, options, systemContext)
			if err != nil {
				return nil, errors.Wrapf(err, "error pulling image %q", image)
			}
			ref, err = is.Transport.ParseStoreReference(store, image)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing reference to image %q", image)
			}
			img, err = is.Transport.GetStoreImage(store, ref)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "no such image %q", image)
		}
		imageID = img.ID
		src, err := ref.NewImage(systemContext)
		if err != nil {
			return nil, errors.Wrapf(err, "error instantiating image")
		}
		defer src.Close()
		config, err = src.ConfigBlob()
		if err != nil {
			return nil, errors.Wrapf(err, "error reading image configuration")
		}
		manifest, _, err = src.Manifest()
		if err != nil {
			return nil, errors.Wrapf(err, "error reading image manifest")
		}
	}

	coptions := storage.ContainerOptions{}
	container, err := store.CreateContainer("", []string{name}, imageID, "", "", &coptions)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating container")
	}

	defer func() {
		if err != nil {
			if err2 := store.DeleteContainer(container.ID); err != nil {
				logrus.Errorf("error deleting container %q: %v", container.ID, err2)
			}
		}
	}()

	builder := &Builder{
		store:            store,
		Type:             containerType,
		FromImage:        image,
		FromImageID:      imageID,
		Config:           config,
		Manifest:         manifest,
		Container:        name,
		ContainerID:      container.ID,
		ImageAnnotations: map[string]string{},
		ImageCreatedBy:   "",
	}

	if options.Mount {
		_, err = builder.Mount("")
		if err != nil {
			return nil, errors.Wrapf(err, "error mounting build container")
		}
	}

	builder.initConfig()
	err = builder.Save()
	if err != nil {
		return nil, errors.Wrapf(err, "error saving builder state")
	}

	return builder, nil
}
