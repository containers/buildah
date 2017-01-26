package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/types"
	"github.com/containers/storage/storage"
	"github.com/urfave/cli"
)

var (
	fromFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "set a name for the working container",
		},
		cli.StringFlag{
			Name:  "image",
			Usage: "name of the starting image",
		},
		cli.BoolFlag{
			Name:  "pull",
			Usage: "pull the image if not present",
		},
		cli.StringFlag{
			Name:  "registry",
			Usage: "prefix to prepend to the image name in order to pull the image",
		},
		cli.BoolFlag{
			Name:  "mount",
			Usage: "mount the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "name of a symlink to create to the root directory of the container",
		},
	}
)

func pullImage(c *cli.Context, store storage.Store, sc *types.SystemContext, name string) error {
	spec := name
	if c.IsSet("registry") {
		spec = c.String("registry") + name
	}

	srcRef, err := transports.ParseImageName(spec)
	if err != nil {
		return fmt.Errorf("error parsing image name %q: %v", spec, err)
	}

	if ref := srcRef.DockerReference(); ref != nil {
		name = ref.FullName()
	}

	destRef, err := is.Transport.ParseStoreReference(store, name)
	if err != nil {
		return fmt.Errorf("error parsing full image name %q: %v", spec, err)
	}

	policy, err := signature.DefaultPolicy(getSystemContext(c))
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}

	logrus.Debugf("copying %q to %q", spec, name)

	err = copy.Image(policyContext, destRef, srcRef, getCopyOptions())
	if err != nil {
		return err
	}

	// Go find the image, and attach the requested name to it, so that we
	// can more easily find it later, even if the destination reference
	// looks different.
	destImage, err := is.Transport.GetStoreImage(store, destRef)
	if err != nil {
		return err
	}

	names := append(destImage.Names, spec, name)
	err = store.SetNames(destImage.ID, names)
	if err != nil {
		return err
	}

	return nil
}

func fromCmd(c *cli.Context) error {
	var img *storage.Image
	manifest := []byte{}
	config := []byte{}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	image := ""
	if c.IsSet("image") {
		image = c.String("image")
	}

	pull := false
	if c.IsSet("pull") {
		pull = c.Bool("pull")
	}

	name := "working-container"
	if c.IsSet("name") {
		name = c.String("name")
	} else {
		if image != "" {
			name = image + "-working-container"
		}
	}
	if name != "" {
		suffix := 1
		tmpName := name
		err = nil
		for err != storage.ErrContainerUnknown {
			_, err = store.GetContainer(tmpName)
			if err == nil {
				suffix++
				tmpName = fmt.Sprintf("%s-%d", name, suffix)
			}
		}
		name = tmpName
	}

	mount := false
	if c.IsSet("mount") {
		mount = c.Bool("mount")
	}

	link := ""
	if c.IsSet("link") {
		link = c.String("link")
		if link == "" {
			return fmt.Errorf("link location can not be empty")
		}
		abs, err := filepath.Abs(link)
		if err != nil {
			return fmt.Errorf("error converting link path %q to absolute path: %v", link, err)
		}
		link = abs
	}

	systemContext := getSystemContext(c)

	if image != "" {
		ref, err := is.Transport.ParseStoreReference(store, image)
		if err != nil {
			return fmt.Errorf("error parsing reference to image %q: %v", image, err)
		}
		img, err = is.Transport.GetStoreImage(store, ref)
		if err != nil {
			if err != storage.ErrImageUnknown || !pull {
				return fmt.Errorf("no such image %q: %v", image, err)
			}
			err = pullImage(c, store, systemContext, image)
			if err != nil {
				return fmt.Errorf("error pulling image %q: %v", image, err)
			}
			ref, err = is.Transport.ParseStoreReference(store, image)
			if err != nil {
				return fmt.Errorf("error parsing reference to image %q: %v", image, err)
			}
			img, err = is.Transport.GetStoreImage(store, ref)
		}
		if err != nil {
			return fmt.Errorf("no such image %q: %v", image, err)
		}
		image = img.ID
		src, err := ref.NewImage(systemContext)
		if err != nil {
			return fmt.Errorf("error instantiating image: %v", err)
		}
		defer src.Close()
		config, err = src.ConfigBlob()
		if err != nil {
			return fmt.Errorf("error reading image configuration: %v", err)
		}
		manifest, _, err = src.Manifest()
		if err != nil {
			return fmt.Errorf("error reading image manifest: %v", err)
		}
	}

	metadata := &ContainerMetadata{
		Type:     ContainerType,
		Config:   config,
		Manifest: manifest,
	}

	options := storage.ContainerOptions{}
	container, err := store.CreateContainer("", []string{name}, image, "", "", &options)
	if err != nil {
		return fmt.Errorf("error creating container: %v", err)
	}

	if mount {
		mountPoint, err := store.Mount(container.ID, "")
		if err != nil {
			return fmt.Errorf("error mounting container: %v", err)
		}
		metadata.Mounts = []string{mountPoint}
		if link != "" {
			err = os.Symlink(mountPoint, link)
			if err != nil {
				return fmt.Errorf("error creating symlink to %q: %v", mountPoint, err)
			}
			metadata.Links = []string{link}
		}
	}
	fmt.Printf("%s\n", name)

	mdata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("error encoding container metadata: %v", err)
	}
	err = store.SetMetadata(container.ID, string(mdata))
	if err != nil {
		return fmt.Errorf("error saving container metadata: %v", err)
	}

	return nil
}
