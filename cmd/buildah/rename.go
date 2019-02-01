package main

import (
	"github.com/containers/buildah"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	renameDescription = "\n  Renames a local container."
	renameCommand     = &cobra.Command{
		Use:   "rename",
		Short: "Rename a container",
		Long:  renameDescription,
		RunE:  renameCmd,
		Example: `  buildah rename containerName NewName
  buildah rename containerID NewName`,
		Args: cobra.ExactArgs(2),
	}
)

func init() {
	rootCmd.AddCommand(renameCommand)
}

func renameCmd(c *cobra.Command, args []string) error {
	var builder *buildah.Builder

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
