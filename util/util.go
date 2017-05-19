package util

import (
	"fmt"

	"github.com/containers/image/docker/reference"
	is "github.com/containers/image/storage"
	"github.com/containers/storage"
)

// ExpandTags takes unqualified names, parses them as image names, and returns
// the fully expanded result, including a tag.
func ExpandTags(tags []string) ([]string, error) {
	expanded := []string{}
	for _, tag := range tags {
		name, err := reference.ParseNormalizedNamed(tag)
		if err != nil {
			return nil, fmt.Errorf("error parsing tag %q: %v", tag, err)
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
		return nil, fmt.Errorf("error parsing reference to image %q: %v", image, err)
	}
	img, err := is.Transport.GetStoreImage(store, ref)
	if err != nil {
		return nil, fmt.Errorf("unable to locate image: %v", err)
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
		return fmt.Errorf("error adding names (%v) to image %q: %v", names, image.ID, err)
	}
	return nil
}
