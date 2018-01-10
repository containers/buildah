package main

import (
	"fmt"
	"os"

	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	rmiDescription = "removes one or more locally stored images."
	rmiFlags       = []cli.Flag{
		cli.BoolFlag{
			Name:  "all, a",
			Usage: "remove all images",
		},
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
	force := c.Bool("force")
	removeAll := c.Bool("all")

	args := c.Args()
	if len(args) == 0 && !removeAll {
		return errors.Errorf("image name or ID must be specified")
	}
	if len(args) > 0 && removeAll {
		return errors.Errorf("when using the --all switch, you may not pass any images names or IDs")
	}

	if err := validateFlags(c, rmiFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	imagesToDelete := args[:]
	var lastError error

	if removeAll {
		images, err := store.Images()
		if err != nil {
			return errors.Wrapf(err, "error reading images")
		}
		for _, image := range images {
			imagesToDelete = append(imagesToDelete, image.ID)
		}
	}

	for _, id := range imagesToDelete {
		image, err := getImage(id, store)
		if err != nil || image == nil {
			if lastError != nil {
				fmt.Fprintln(os.Stderr, lastError)
			}
			if err == nil {
				err = storage.ErrNotAnImage
			}
			lastError = errors.Wrapf(err, "could not get image %q", id)
			continue
		}
		if image != nil {
			ctrIDs, err := runningContainers(image, store)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error getting running containers for image %q", id)
				continue
			}
			if len(ctrIDs) > 0 && len(image.Names) <= 1 {
				if force {
					err = removeContainers(ctrIDs, store)
					if err != nil {
						if lastError != nil {
							fmt.Fprintln(os.Stderr, lastError)
						}
						lastError = errors.Wrapf(err, "error removing containers %v for image %q", ctrIDs, id)
						continue
					}
				} else {
					for _, ctrID := range ctrIDs {
						if lastError != nil {
							fmt.Fprintln(os.Stderr, lastError)
						}
						lastError = errors.Wrapf(storage.ErrImageUsedByContainer, "Could not remove image %q (must force) - container %q is using its reference image", id, ctrID)
					}
					continue
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
					if lastError != nil {
						fmt.Fprintln(os.Stderr, lastError)
					}
					lastError = errors.Wrapf(err2, "error removing tag %q from image %q", id, image.ID)
					continue
				}
				fmt.Printf("untagged: %s\n", name)
			}

			if len(image.Names) > 0 {
				continue
			}
			id, err := removeImage(image, store)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error removing image %q", image.ID)
				continue
			}
			fmt.Printf("%s\n", id)
		}
	}

	if lastError != nil {
		fmt.Fprintln(os.Stderr, lastError)
	}
	return nil
}

func getImage(id string, store storage.Store) (*storage.Image, error) {
	var ref types.ImageReference
	ref, err := properImageRef(id)
	if err != nil {
		logrus.Debug(err)
	}
	if ref == nil {
		if ref, err = storageImageRef(store, id); err != nil {
			logrus.Debug(err)
		}
	}
	if ref == nil {
		if ref, err = storageImageID(store, id); err != nil {
			logrus.Debug(err)
		}
	}
	if ref != nil {
		image, err2 := is.Transport.GetStoreImage(store, ref)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "error reading image using reference %q", transports.ImageName(ref))
		}
		return image, nil
	}
	return nil, err
}

func untagImage(imgArg string, image *storage.Image, store storage.Store) (string, error) {
	newNames := []string{}
	removedName := ""
	for _, name := range image.Names {
		if matchesReference(name, imgArg) {
			removedName = name
			continue
		}
		newNames = append(newNames, name)
	}
	if removedName != "" {
		if err := store.SetNames(image.ID, newNames); err != nil {
			return "", errors.Wrapf(err, "error removing name %q from image %q", removedName, image.ID)
		}
	}
	return removedName, nil
}

func removeImage(image *storage.Image, store storage.Store) (string, error) {
	if _, err := store.DeleteImage(image.ID, true); err != nil {
		return "", errors.Wrapf(err, "could not remove image %q", image.ID)
	}
	return image.ID, nil
}

// Returns a list of running containers associated with the given ImageReference
func runningContainers(image *storage.Image, store storage.Store) ([]string, error) {
	ctrIDs := []string{}
	containers, err := store.Containers()
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
	for _, ctrID := range ctrIDs {
		if err := store.DeleteContainer(ctrID); err != nil {
			return errors.Wrapf(err, "could not remove container %q", ctrID)
		}
	}
	return nil
}

// If it's looks like a proper image reference, parse it and check if it
// corresponds to an image that actually exists.
func properImageRef(id string) (types.ImageReference, error) {
	var err error
	if ref, err := alltransports.ParseImageName(id); err == nil {
		if img, err2 := ref.NewImageSource(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, errors.Wrapf(err, "error confirming presence of image reference %q", transports.ImageName(ref))
	}
	return nil, errors.Wrapf(err, "error parsing %q as an image reference", id)
}

// If it's looks like an image reference that's relative to our storage, parse
// it and check if it corresponds to an image that actually exists.
func storageImageRef(store storage.Store, id string) (types.ImageReference, error) {
	var err error
	if ref, err := is.Transport.ParseStoreReference(store, id); err == nil {
		if img, err2 := ref.NewImageSource(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, errors.Wrapf(err, "error confirming presence of storage image reference %q", transports.ImageName(ref))
	}
	return nil, errors.Wrapf(err, "error parsing %q as a storage image reference", id)
}

// If it might be an ID that's relative to our storage, truncated or not, so
// parse it and check if it corresponds to an image that we have stored
// locally.
func storageImageID(store storage.Store, id string) (types.ImageReference, error) {
	var err error
	imageID := id
	if img, err := store.Image(id); err == nil && img != nil {
		imageID = img.ID
	}
	if ref, err := is.Transport.ParseStoreReference(store, imageID); err == nil {
		if img, err2 := ref.NewImageSource(nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, errors.Wrapf(err, "error confirming presence of storage image reference %q", transports.ImageName(ref))
	}
	return nil, errors.Wrapf(err, "error parsing %q as a storage image reference: %v", id)
}
