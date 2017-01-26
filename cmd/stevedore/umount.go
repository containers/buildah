package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

var (
	umountFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "name",
			Usage: "name of the working container",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "root directory of the working container",
		},
		cli.StringFlag{
			Name:  "link",
			Usage: "symlink to the root directory of the working container",
		},
	}
)

func umountCmd(c *cli.Context) error {
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
	if name == "" && root == "" && link == "" {
		return fmt.Errorf("either --name or --root or --link, or some combination, must be specified")
	}

	container, err := lookupContainer(store, name, root, link)
	if err != nil {
		return err
	}

	err = store.Unmount(container.ID)
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

	for _, link := range metadata.Links {
		err = os.Remove(link)
		if err != nil {
			return fmt.Errorf("error removing symlink %q: %v", link, err)
		}
	}
	metadata.Links = nil

	mdata2, err := json.Marshal(&metadata)
	if err != nil {
		return err
	}
	err = store.SetMetadata(container.ID, string(mdata2))
	if err != nil {
		return err
	}

	return nil
}
