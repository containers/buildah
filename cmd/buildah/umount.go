package main

import (
	"fmt"
	"os"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	umountCommand := &cobra.Command{
		Use:     "umount",
		Aliases: []string{"unmount"},
		Short:   "Unmount the root file system of the specified working containers",
		Long:    "Unmounts the root file system of the specified working containers.",
		RunE:    umountCmd,
		Example: `buildah umount containerID
  buildah umount containerID1 containerID2 containerID3
  buildah umount --all`,
	}
	umountCommand.SetUsageTemplate(UsageTemplate())

	flags := umountCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolP("all", "a", false, "umount all of the currently mounted containers")

	rootCmd.AddCommand(umountCommand)
}

func umountCmd(c *cobra.Command, args []string) error {
	umountAll := false
	if c.Flag("all").Changed {
		umountAll = true
	}
	umountContainerErrStr := "error unmounting container"
	if len(args) == 0 && !umountAll {
		return errors.Errorf("at least one container ID must be specified")
	}
	if len(args) > 0 && umountAll {
		return errors.Errorf("when using the --all switch, you may not pass any container IDs")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	var lastError error
	if len(args) > 0 {
		for _, name := range args {
			builder, err := openBuilder(getContext(), store, name)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %s", umountContainerErrStr, name)
				continue
			}
			if builder.MountPoint == "" {
				continue
			}

			if err = builder.Unmount(); err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container)
				continue
			}
			fmt.Printf("%s\n", builder.ContainerID)
		}
	} else {
		builders, err := openBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build Containers")
		}
		for _, builder := range builders {
			if builder.MountPoint == "" {
				continue
			}

			if err = builder.Unmount(); err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "%s %q", umountContainerErrStr, builder.Container)
				continue
			}
			fmt.Printf("%s\n", builder.ContainerID)
		}
	}
	return lastError
}
