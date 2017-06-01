package util

import (
	"github.com/containers/image/docker/reference"
	is "github.com/containers/image/storage"
	"github.com/containers/storage"
	"github.com/pkg/errors"
)

// ExpandTags takes unqualified names, parses them as image names, and returns
// the fully expanded result, including a tag.
func ExpandTags(tags []string) ([]string, error) {
	expanded := []string{}
	for _, tag := range tags {
		name, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing tag %q", tag)
		}
		name = reference.TagNameOnly(name)
		tag = ""
		if tagged, ok := name.(reference.NamedTagged); ok {
			tag = ":" + tagged.Tag()
		}
		expanded = append(expanded, name.Name()+tag)
	}
	return expanded, nil
}

// FindImage locates the locally-stored image which corresponds to a given name.
func FindImage(store storage.Store, image string) (*storage.Image, error) {
	ref, err := is.Transport.ParseStoreReference(store, image)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing reference to image %q", image)
	}
	img, err := is.Transport.GetStoreImage(store, ref)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to locate image")
	}
	return img, nil
}

// AddImageNames adds the specified names to the specified image.
func AddImageNames(store storage.Store, image *storage.Image, addNames []string) error {
	names, err := ExpandTags(addNames)
	if err != nil {
		return err
	}
	err = store.SetNames(image.ID, append(image.Names, names...))
	if err != nil {
		return errors.Wrapf(err, "error adding names (%v) to image %q", names, image.ID)
	}
	return nil
}
