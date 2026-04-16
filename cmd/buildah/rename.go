package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.podman.io/buildah"
)

var (
	renameDescription = "\n  Renames a working container."
	renameCommand     = &cobra.Command{
		Use:   "rename",
		Short: "Rename a working container",
		Long:  renameDescription,
		RunE:  renameCmd,
		Example: `buildah rename containerName NewName
  buildah rename containerID NewName`,
		Args:    cobra.ExactArgs(2),
		GroupID: groupContainers,
	}
)

func renameInit() {
	renameCommand.SetUsageTemplate(UsageTemplate())
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
		return fmt.Errorf("reading build container %q: %w", name, err)
	}

	oldName := builder.Container
	if oldName == newName {
		return fmt.Errorf("renaming a container with the same name as its current name")
	}

	if build, err := openBuilder(getContext(), store, newName); err == nil {
		return fmt.Errorf("the container name %q is already in use by container %q", newName, build.ContainerID)
	}

	err = store.SetNames(builder.ContainerID, []string{newName})
	if err != nil {
		return fmt.Errorf("renaming container %q to the name %q: %w", oldName, newName, err)
	}
	builder.Container = newName
	return builder.Save()
}
