package buildah

import (
	"fmt"
	"strings"

	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/openshift/imagebuilder"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// BaseImageFakeName is the "name" of a source image which we interpret
	// as "no image".
	BaseImageFakeName = imagebuilder.NoBaseImageSpecifier

	// DefaultTransport is a prefix that we apply to an image name if we
	// can't find one in the local Store, in order to generate a source
	// reference for the image that we can then copy to the local Store.
	DefaultTransport = "docker://"
)

func newBuilder(store storage.Store, options BuilderOptions) (*Builder, error) {
	var err error
	var ref types.ImageReference
	var img *storage.Image
	manifest := []byte{}
	config := []byte{}

	if options.FromImage == BaseImageFakeName {
		options.FromImage = ""
	}
	image := options.FromImage

	if options.Transport == "" {
		options.Transport = DefaultTransport
	}

	systemContext := getSystemContext(options.SignaturePolicyPath)

	imageID := ""
	if image != "" {
		if options.PullPolicy == PullAlways {
			pulledReference, err2 := pullImage(store, options, systemContext)
			if err2 != nil {
				return nil, errors.Wrapf(err2, "error pulling image %q", image)
			}
			ref = pulledReference
		}
		if ref == nil {
			srcRef, err2 := alltransports.ParseImageName(image)
			if err2 != nil {
				srcRef2, err3 := alltransports.ParseImageName(options.Registry + image)
				if err3 != nil {
					srcRef3, err4 := alltransports.ParseImageName(options.Transport + options.Registry + image)
					if err4 != nil {
						return nil, errors.Wrapf(err4, "error parsing image name %q", options.Transport+options.Registry+image)
					}
					srcRef2 = srcRef3
				}
				srcRef = srcRef2
			}

			destImage, err2 := localImageNameForReference(store, srcRef)
			if err2 != nil {
				return nil, errors.Wrapf(err2, "error computing local image name for %q", transports.ImageName(srcRef))
			}
			if destImage == "" {
				return nil, errors.Errorf("error computing local image name for %q", transports.ImageName(srcRef))
			}

			ref, err = is.Transport.ParseStoreReference(store, destImage)
			if err != nil {
				return nil, errors.Wrapf(err, "error parsing reference to image %q", destImage)
			}

			image = destImage
		}
		img, err = is.Transport.GetStoreImage(store, ref)
		if err != nil {
			if errors.Cause(err) == storage.ErrImageUnknown && options.PullPolicy != PullIfMissing {
				return nil, errors.Wrapf(err, "no such image %q", transports.ImageName(ref))
			}
			ref2, err2 := pullImage(store, options, systemContext)
			if err2 != nil {
				return nil, errors.Wrapf(err2, "error pulling image %q", image)
			}
			ref = ref2
			img, err = is.Transport.GetStoreImage(store, ref)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "no such image %q", transports.ImageName(ref))
		}
		imageID = img.ID
		src, err := ref.NewImage(systemContext)
		if err != nil {
			return nil, errors.Wrapf(err, "error instantiating image for %q", transports.ImageName(ref))
		}
		defer src.Close()
		config, err = src.ConfigBlob()
		if err != nil {
			return nil, errors.Wrapf(err, "error reading image configuration for %q", transports.ImageName(ref))
		}
		manifest, _, err = src.Manifest()
		if err != nil {
			return nil, errors.Wrapf(err, "error reading image manifest for %q", transports.ImageName(ref))
		}
	}

	name := "working-container"
	if options.Container != "" {
		name = options.Container
	} else {
		var err2 error
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
		suffix := 1
		tmpName := name
		for errors.Cause(err2) != storage.ErrContainerUnknown {
			_, err2 = store.Container(tmpName)
			if err2 == nil {
				suffix++
				tmpName = fmt.Sprintf("%s-%d", name, suffix)
			}
		}
		name = tmpName
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
