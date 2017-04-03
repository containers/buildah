package main

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/urfave/cli"
)

var (
	rmiDescription = "Removes one or more locally stored images."
	rmiCommand     = cli.Command{
		Name:        "rmi",
		Usage:       "Removes one or more images from local storage",
		Description: rmiDescription,
		Action:      rmiCmd,
		ArgsUsage:   "IMAGE-NAME-OR-ID [...]",
	}
)

func rmiCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("image name or ID must be specified")
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	for _, id := range args {
		// If it's an exact name or ID match with the underlying
		// storage library's information about the image, then it's
		// enough.
		_, err = store.DeleteImage(id, true)
		if err != nil {
			var ref types.ImageReference
			// If it's looks like a proper image reference, parse
			// it and check if it corresponds to an image that
			// actually exists.
			if ref2, err2 := alltransports.ParseImageName(id); err2 == nil {
				if img, err3 := ref2.NewImage(nil); err3 == nil {
					img.Close()
					ref = ref2
				} else {
					logrus.Debugf("error confirming presence of image %q: %v", transports.ImageName(ref2), err3)
				}
			} else {
				logrus.Debugf("error parsing %q as an image reference: %v", id, err2)
			}
			if ref == nil {
				// If it's looks like an image reference that's
				// relative to our storage, parse it and check
				// if it corresponds to an image that actually
				// exists.
				if ref2, err2 := storage.Transport.ParseStoreReference(store, id); err2 == nil {
					if img, err3 := ref2.NewImage(nil); err3 == nil {
						img.Close()
						ref = ref2
					} else {
						logrus.Debugf("error confirming presence of image %q: %v", transports.ImageName(ref2), err3)
					}
				} else {
					logrus.Debugf("error parsing %q as a store reference: %v", id, err2)
				}
			}
			if ref == nil {
				// If it might be an ID that's relative to our
				// storage, parse it and check if it
				// corresponds to an image that actually
				// exists.  This _should_ be redundant, since
				// we already tried deleting the image using
				// the ID directly above, but it can't hurt,
				// either.
				if ref2, err2 := storage.Transport.ParseStoreReference(store, "@"+id); err2 == nil {
					if img, err3 := ref2.NewImage(nil); err3 == nil {
						img.Close()
						ref = ref2
					} else {
						logrus.Debugf("error confirming presence of image %q: %v", transports.ImageName(ref2), err3)
					}
				} else {
					logrus.Debugf("error parsing %q as an image reference: %v", "@"+id, err2)
				}
			}
			if ref != nil {
				err = ref.DeleteImage(nil)
			}
		}
		if err != nil {
			return fmt.Errorf("error removing image %q: %v", id, err)
		}
		fmt.Printf("%s\n", id)
	}

	return nil
}
