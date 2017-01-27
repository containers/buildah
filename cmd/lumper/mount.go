package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

var (
	mountFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "name of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "a previous root directory of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "name of a symlink to create",
		},
	}
)

func mountCmd(c *cli.Context) error {
	store, err := getStore(c)
	if err != nil {
		return err
	}

	name := ""
	if c.IsSet("name") {
		name = c.String("name")
	}
	root := ""
	if c.IsSet("root") {
		root = c.String("root")
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
	if name == "" && root == "" {
		return fmt.Errorf("either --name or --root, or both, must be specified")
	}

	container, err := lookupContainer(store, name, root, "")
	if err != nil {
		return err
	}

	mdata, err := store.GetMetadata(container.ID)
	if err != nil {
		return err
	}
	metadata := ContainerMetadata{}
	err = json.Unmarshal([]byte(mdata), &metadata)
	if err != nil {
		return err
	}

	mountPoint, err := store.Mount(container.ID, "")
	if err != nil {
		return fmt.Errorf("error mounting container: %v", err)
	}

	present := false
	for _, m := range metadata.Mounts {
		if m == mountPoint {
			present = true
			break
		}
	}
	if !present {
		metadata.Mounts = append(append([]string{}, metadata.Mounts...), mountPoint)
	}

	if link != "" {
		err = os.Symlink(mountPoint, link)
		if err != nil {
			return fmt.Errorf("error creating symlink to %q: %v", mountPoint, err)
		}
		metadata.Links = append(append([]string{}, metadata.Links...), link)
	}

	mdata2, err := json.Marshal(&metadata)
	if err != nil {
		return err
	}
	err = store.SetMetadata(container.ID, string(mdata2))
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", mountPoint)

	return nil
}
