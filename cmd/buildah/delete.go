package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var (
	deleteFlags = []cli.Flag{
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

func deleteCmd(c *cli.Context) error {
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
	}
	if name == "" && root == "" && link == "" {
		return fmt.Errorf("either --name or --root or --link, or some combination, must be specified")
	}

	container, err := lookupContainer(store, name, root, link)
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

	err = store.DeleteContainer(container.ID)
	if err != nil {
		return fmt.Errorf("error deleting container: %v", err)
	}

	return nil
}
