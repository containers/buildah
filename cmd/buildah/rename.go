package main

import (
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	renameDescription = "Rename a local container"
	renameCommand     = cli.Command{
		Name:                   "rename",
		Usage:                  "Rename a container",
		Description:            renameDescription,
		Action:                 renameCmd,
		ArgsUsage:              "CONTAINER-NAME-OR-ID CONTAINER-NAME",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func renameCmd(c *cli.Context) error {
	var builder *buildah.Builder

	args := c.Args()
	if len(args) != 2 {
		return errors.Errorf("container and it's new name must be specified'")
	}

	name := args[0]
	newName := args[1]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err = openBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	oldName := builder.Container
	if oldName == newName {
		return errors.Errorf("renaming a container with the same name as its current name")
	}

	if build, err := openBuilder(getContext(), store, newName); err == nil {
		return errors.Errorf("The container name %q is already in use by container %q", newName, build.ContainerID)
	}

	err = store.SetNames(builder.ContainerID, []string{newName})
	if err != nil {
		return errors.Wrapf(err, "error renaming container %q to the name %q", oldName, newName)
	}
	builder.Container = newName
	return builder.Save()
}
