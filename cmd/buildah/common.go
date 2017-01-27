package main

import (
	"encoding/json"
	"fmt"

	"github.com/containers/image/copy"
	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage/storage"
	"github.com/urfave/cli"
)

const (
	Package       = "buildah"
	ContainerType = Package + " 0.0.0"
)

type ContainerMetadata struct {
	Type     string
	Config   []byte
	Manifest []byte
	Links    []string
	Mounts   []string
}

func getStore(c *cli.Context) (storage.Store, error) {
	options := storage.DefaultStoreOptions
	if c.GlobalIsSet("root") || c.GlobalIsSet("runroot") {
		options.GraphRoot = c.GlobalString("root")
		options.RunRoot = c.GlobalString("runroot")
	}
	if c.GlobalIsSet("storage-driver") {
		options.GraphDriverName = c.GlobalString("storage-driver")
	}
	if c.GlobalIsSet("storage-options") {
		opts := c.GlobalStringSlice("storage-options")
		if len(opts) > 0 {
			options.GraphDriverOptions = opts
		}
	}
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	return store, err
}

func getSystemContext(c *cli.Context) *types.SystemContext {
	sc := &types.SystemContext{}
	if c.GlobalIsSet("signature-policy") {
		sc.SignaturePolicyPath = c.GlobalString("signature-policy")
	}
	return sc
}

func getCopyOptions() *copy.Options {
	return &copy.Options{}
}

func lookupContainer(store storage.Store, name, root, link string) (*storage.Container, error) {
	containers, err := store.Containers()
	if err != nil {
		return nil, fmt.Errorf("error listing containers: %v", err)
	}
	for _, c := range containers {
		if name != "" {
			matches := false
			for _, n := range c.Names {
				if name == n {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}
		metadata := ContainerMetadata{}
		if root != "" || link != "" {
			mdata, err := store.GetMetadata(c.ID)
			if err != nil || mdata == "" {
				// probably not one of ours
				continue
			}
			err = json.Unmarshal([]byte(mdata), &metadata)
			if err != nil {
				// probably not one of ours
				continue
			}
		}
		if root != "" {
			matches := false
			for _, m := range metadata.Mounts {
				if m == root {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}
		if link != "" {
			matches := false
			for _, l := range metadata.Links {
				if l == link {
					matches = true
					break
				}
			}
			if !matches {
				continue
			}
		}
		return &c, nil
	}
	return nil, fmt.Errorf("no matching container found")
}
