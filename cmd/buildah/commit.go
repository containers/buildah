package main

import (
	"encoding/json"
	"fmt"

	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports"
	"github.com/containers/storage/pkg/archive"
	"github.com/urfave/cli"
)

var (
	commitFlags = []cli.Flag{
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
		cli.BoolFlag{
			Name:  "do-not-compress",
			Usage: "don't compress layers",
		},
		cli.StringFlag{
			Name:  "output",
			Usage: "image to create",
		},
	}
)

func commitCmd(c *cli.Context) error {
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
	output := ""
	if c.IsSet("output") {
		output = c.String("output")
	}
	compress := archive.Uncompressed
	if !c.IsSet("do-not-compress") || !c.Bool("do-not-compress") {
		compress = archive.Gzip
	}
	if output == "" {
		return fmt.Errorf("the --output flag must be specified")
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

	policy, err := signature.DefaultPolicy(getSystemContext(c))
	if err != nil {
		return err
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}

	destRef, err := transports.ParseImageName(output)
	if err != nil {
		return fmt.Errorf("error parsing output image name %q: %v", output, err)
	}

	config := updateConfig(c, metadata.Config)
	err = copy.Image(policyContext, destRef, makeContainerImageRef(store, container, string(config), compress), getCopyOptions())

	return err
}
