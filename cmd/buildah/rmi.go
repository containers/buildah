package main

import (
	"fmt"

	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	rmiDescription = "removes one or more locally stored images."
	rmiFlags       = []cli.Flag{
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "force removal of the image",
		},
	}
	rmiCommand = cli.Command{
		Name:        "rmi",
		Usage:       "removes one or more images from local storage",
		Description: rmiDescription,
		Action:      rmiCmd,
		ArgsUsage:   "IMAGE-NAME-OR-ID [...]",
		Flags:       rmiFlags,
	}
)

func rmiCmd(c *cli.Context) error {

	force := false
	if c.IsSet("force") {
		force = c.Bool("force")
	}

	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("image name or ID must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	for _, id := range args {
		image, err := getImage(id, store)
		if err != nil {
			return errors.Wrapf(err, "could not get image %q", id)
		}
		if image != nil {
			ctrIDs, err := runningContainers(image, store)
			if err != nil {
				return errors.Wrapf(err, "error getting running containers for image %q", id)
			}
			if len(ctrIDs) > 0 && len(image.Names) <= 1 {
				if force {
					removeContainers(ctrIDs, store)
				} else {
					for ctrID := range ctrIDs {
						return fmt.Errorf("Could not remove image %q (must force) - container %q is using its reference image", id, ctrID)
					}
				}
			}
			// If the user supplied an ID, we cannot delete the image if it is referred to by multiple tags
			if matchesID(image.ID, id) {
				if len(image.Names) > 1 && !force {
					return fmt.Errorf("unable to delete %s (must force) - image is referred to in multiple tags", image.ID)
				}
				// If it is forced, we have to untag the image so that it can be deleted
				image.Names = image.Names[:0]
			} else {
				name, err2 := untagImage(id, image, store)
				if err2 != nil {
					return err
				}
				fmt.Printf("untagged: %s", name)
			}

			if len(image.Names) > 0 {
				continue
			}
			id, err := removeImage(image, store)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", id)
		}
	}

	return nil
}

func getImage(id string, store storage.Store) (*storage.Image, error) {
	var ref types.ImageReference
	ref, err := properImageRef(id)
	if err != nil {
		//logrus.Debug(err)
	}
	if ref == nil {
		if ref, err = storageImageRef(store, id); err != nil {
			//logrus.Debug(err)
		}
	}
	if ref == nil {
		if ref, err = storageImageID(store, id); err != nil {
			//logrus.Debug(err)
		}
	}
	if ref != nil {
		image, err2 := is.Transport.GetStoreImage(store, ref)
		if err2 != nil {
			return nil, err2
		}
		return image, nil
	}
	return nil, err
}

func untagImage(imgArg string, image *storage.Image, store storage.Store) (string, error) {
	// Remove name from image.Names and set the new name in the ImageStore
	imgStore, err := store.ImageStore()
	if err != nil {
		return "", errors.Wrap(err, "could not untag image")
	}
	newNames := []string{}
	removedName := ""
	for _, name := range image.Names {
		if matchesReference(name, imgArg) {
			removedName = name
			continue
		}
		newNames = append(newNames, name)
	}
	imgStore.SetNames(image.ID, newNames)
	err = imgStore.Save()
	return removedName, err
}

func removeImage(image *storage.Image, store storage.Store) (string, error) {
	imgStore, err := store.ImageStore()
	if err != nil {
		return "", errors.Wrapf(err, "could not open image store")
	}
	err = imgStore.Delete(image.ID)
	if err != nil {
		return "", errors.Wrapf(err, "could not remove image")
	}
	err = imgStore.Save()
	if err != nil {
		return "", errors.Wrapf(err, "could not save image store")
	}
	return image.ID, nil
}

// Returns a list of running containers associated with the given ImageReference
func runningContainers(image *storage.Image, store storage.Store) ([]string, error) {
	ctrIDs := []string{}
	ctrStore, err := store.ContainerStore()
	if err != nil {
		return nil, err
	}

	containers, err := ctrStore.Containers()
	if err != nil {
		return nil, err
	}
	for _, ctr := range containers {
		if ctr.ImageID == image.ID {
			ctrIDs = append(ctrIDs, ctr.ID)
		}
	}
	return ctrIDs, nil
}

func removeContainers(ctrIDs []string, store storage.Store) error {
	ctrStore, err := store.ContainerStore()
	if err != nil {
		return err
	}
	for _, ctrID := range ctrIDs {
		if err = ctrStore.Delete(ctrID); err != nil {
			return errors.Wrapf(err, "could not remove container %q", ctrID)
		}
	}
	return nil
}

// If it's looks like a proper image reference, parse it and check if it
// corresponds to an image that actually exists.
func properImageRef(id string) (types.ImageReference, error) {
	var ref types.ImageReference
	var err error
	if ref, err = alltransports.ParseImageName(id); err == nil {
		if img, err2 := ref.NewImage(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, fmt.Errorf("error confirming presence of image reference %q: %v", transports.ImageName(ref), err)
	}
	return nil, fmt.Errorf("error parsing %q as an image reference: %v", id, err)
}

// If it's looks like an image reference that's relative to our storage, parse
// it and check if it corresponds to an image that actually exists.
func storageImageRef(store storage.Store, id string) (types.ImageReference, error) {
	var ref types.ImageReference
	var err error
	if ref, err = is.Transport.ParseStoreReference(store, id); err == nil {
		if img, err2 := ref.NewImage(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, fmt.Errorf("error confirming presence of storage image reference %q: %v", transports.ImageName(ref), err)
	}
	return nil, fmt.Errorf("error parsing %q as a storage image reference: %v", id, err)
}

// If it might be an ID that's relative to our storage, parse it and check if it
// corresponds to an image that actually exists.  This _should_ be redundant,
// since we already tried deleting the image using the ID directly above, but it
// can't hurt either.
func storageImageID(store storage.Store, id string) (types.ImageReference, error) {
	var ref types.ImageReference
	var err error
	if ref, err = is.Transport.ParseStoreReference(store, "@"+id); err == nil {
		if img, err2 := ref.NewImage(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, fmt.Errorf("error confirming presence of storage image reference %q: %v", transports.ImageName(ref), err)
	}
	return nil, fmt.Errorf("error parsing %q as a storage image reference: %v", "@"+id, err)
}
