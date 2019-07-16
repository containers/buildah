package main

import (
	"context"
	"fmt"
	"os"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type rmiResults struct {
	all   bool
	prune bool
	force bool
}

func init() {
	var (
		rmiDescription = "\n  Removes one or more locally stored images."
		opts           rmiResults
	)
	rmiCommand := &cobra.Command{
		Use:   "rmi",
		Short: "Remove one or more images from local storage",
		Long:  rmiDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rmiCmd(cmd, args, opts)
		},
		Example: `buildah rmi imageID
  buildah rmi --all --force
  buildah rmi imageID1 imageID2 imageID3`,
	}
	rmiCommand.SetUsageTemplate(UsageTemplate())

	flags := rmiCommand.Flags()
	flags.SetInterspersed(false)

	flags.BoolVarP(&opts.all, "all", "a", false, "remove all images")
	flags.BoolVarP(&opts.prune, "prune", "p", false, "prune dangling images")
	flags.BoolVarP(&opts.force, "force", "f", false, "force removal of the image and any containers using the image")

	rootCmd.AddCommand(rmiCommand)
}

func rmiCmd(c *cobra.Command, args []string, iopts rmiResults) error {
	if len(args) == 0 && !iopts.all && !iopts.prune {
		return errors.Errorf("image name or ID must be specified")
	}
	if len(args) > 0 && iopts.all {
		return errors.Errorf("when using the --all switch, you may not pass any images names or IDs")
	}
	if iopts.all && iopts.prune {
		return errors.Errorf("when using the --all switch, you may not use --prune switch")
	}
	if len(args) > 0 && iopts.prune {
		return errors.Errorf("when using the --prune switch, you may not pass any images names or IDs")
	}

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	imagesToDelete := args[:]

	if iopts.all {
		imagesToDelete, err = findAllImages(store)
		if err != nil {
			return err
		}
	}

	if iopts.prune {
		imagesToDelete, err = findDanglingImages(store)
		if err != nil {
			return err
		}
	}

	ctx := getContext()
	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	return deleteImages(ctx, systemContext, store, imagesToDelete, iopts.all, iopts.force)
}

func deleteImages(ctx context.Context, systemContext *types.SystemContext, store storage.Store, imagesToDelete []string, removeAll, force bool) error {
	var lastError error
	for _, id := range imagesToDelete {
		image, err := getImage(ctx, systemContext, store, id)
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

		ctrIDs, err := runningContainers(store, image)
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
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Errorf("unable to delete %s (must force) - image is referred to in multiple tags", image.ID)
				continue
			}
			// If it is forced, we have to untag the image so that it can be deleted
			image.Names = image.Names[:0]
		} else {
			name, err2 := untagImage(id, store, image)
			if err2 != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err2, "error removing tag %q from image %q", id, image.ID)
				continue
			}
			fmt.Printf("untagged: %s\n", name)

			// Need to fetch the image state again after making changes to it i.e untag
			// because only a copy of the image state is returned
			image, err = getImage(ctx, systemContext, store, image.ID)
			if err != nil || image == nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error getting image after untag %q", image.ID)
			}
		}

		isParent, err := imageIsParent(ctx, systemContext, store, image)
		if err != nil {
			if lastError != nil {
				fmt.Fprintln(os.Stderr, lastError)
			}
			lastError = errors.Wrapf(err, "error determining if the image %q is a parent", image.ID)
			continue
		}
		// If the --all flag is not set and the image has named references or is
		// a parent, do not delete image.
		if len(image.Names) > 0 && !removeAll {
			continue
		}

		if isParent && len(image.Names) == 0 && !removeAll {
			if lastError != nil {
				fmt.Fprintln(os.Stderr, lastError)
			}
			lastError = errors.Errorf("unable to delete %q (cannot be forced) - image has dependent child images", image.ID)
			continue
		}
		id, err := removeImage(ctx, systemContext, store, image)
		if err != nil {
			if lastError != nil {
				fmt.Fprintln(os.Stderr, lastError)
			}
			lastError = errors.Wrapf(err, "error removing image %q", image.ID)
			continue
		}
		fmt.Printf("%s\n", id)
	}

	return lastError
}

func getImage(ctx context.Context, systemContext *types.SystemContext, store storage.Store, id string) (*storage.Image, error) {
	var ref types.ImageReference
	ref, err := properImageRef(ctx, id)
	if err != nil {
		logrus.Debug(err)
	}
	if ref == nil {
		if ref, err = storageImageRef(systemContext, store, id); err != nil {
			logrus.Debug(err)
		}
	}
	if ref == nil {
		if ref, err = storageImageID(ctx, store, id); err != nil {
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

func untagImage(imgArg string, store storage.Store, image *storage.Image) (string, error) {
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

func removeImage(ctx context.Context, systemContext *types.SystemContext, store storage.Store, image *storage.Image) (string, error) {
	parent, err := getParent(ctx, systemContext, store, image)
	if err != nil {
		return "", err
	}
	if _, err := store.DeleteImage(image.ID, true); err != nil {
		return "", errors.Wrapf(err, "could not remove image %q", image.ID)
	}
	for parent != nil {
		nextParent, err := getParent(ctx, systemContext, store, parent)
		if err != nil {
			return image.ID, errors.Wrapf(err, "unable to get parent from image %q", image.ID)
		}
		isParent, err := imageIsParent(ctx, systemContext, store, parent)
		if err != nil {
			return image.ID, errors.Wrapf(err, "unable to get check if image %q is a parent", image.ID)
		}
		// Do not remove if image is a base image and is not untagged, or if
		// the image has more children.
		if len(parent.Names) > 0 || isParent {
			return image.ID, nil
		}
		id := parent.ID
		if _, err := store.DeleteImage(id, true); err != nil {
			logrus.Debugf("unable to remove intermediate image %q: %v", id, err)
		} else {
			fmt.Println(id)
		}
		parent = nextParent
	}
	return image.ID, nil
}

// Returns a list of running containers associated with the given ImageReference
func runningContainers(store storage.Store, image *storage.Image) ([]string, error) {
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
func properImageRef(ctx context.Context, id string) (types.ImageReference, error) {
	var err error
	if ref, err := alltransports.ParseImageName(id); err == nil {
		if img, err2 := ref.NewImageSource(ctx, nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, errors.Wrapf(err, "error confirming presence of image reference %q", transports.ImageName(ref))
	}
	return nil, errors.Wrapf(err, "error parsing %q as an image reference", id)
}

// If it's looks like an image reference that's relative to our storage, parse
// it and check if it corresponds to an image that actually exists.
func storageImageRef(systemContext *types.SystemContext, store storage.Store, id string) (types.ImageReference, error) {
	ref, _, err := util.FindImage(store, "", systemContext, id)
	if err != nil {
		if ref != nil {
			return nil, errors.Wrapf(err, "error confirming presence of storage image reference %q", transports.ImageName(ref))
		}
		return nil, errors.Wrapf(err, "error confirming presence of storage image name %q", id)
	}
	return ref, err
}

// If it might be an ID that's relative to our storage, truncated or not, so
// parse it and check if it corresponds to an image that we have stored
// locally.
func storageImageID(ctx context.Context, store storage.Store, id string) (types.ImageReference, error) {
	var err error
	imageID := id
	if img, err := store.Image(id); err == nil && img != nil {
		imageID = img.ID
	}
	if ref, err := is.Transport.ParseStoreReference(store, imageID); err == nil {
		if img, err2 := ref.NewImageSource(ctx, nil); err2 == nil {
			img.Close()
			return ref, nil
		}
		return nil, errors.Wrapf(err, "error confirming presence of storage image reference %q", transports.ImageName(ref))
	}
	return nil, errors.Wrapf(err, "error parsing %q as a storage image reference", id)
}

// Returns a list of all existing images
func findAllImages(store storage.Store) ([]string, error) {
	imagesToDelete := []string{}

	images, err := store.Images()
	if err != nil {
		return nil, errors.Wrapf(err, "error reading images")
	}
	for _, image := range images {
		imagesToDelete = append(imagesToDelete, image.ID)
	}

	return imagesToDelete, nil
}

// Returns a list of all dangling images
func findDanglingImages(store storage.Store) ([]string, error) {
	imagesToDelete := []string{}

	images, err := store.Images()
	if err != nil {
		return nil, errors.Wrapf(err, "error reading images")
	}
	for _, image := range images {
		if len(image.Names) == 0 {
			imagesToDelete = append(imagesToDelete, image.ID)
		}
	}

	return imagesToDelete, nil
}
