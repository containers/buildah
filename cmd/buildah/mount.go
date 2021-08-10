package main

import (
	"encoding/json"
	"fmt"
	"os"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type jsonMount struct {
	Container  string `json:"container,omitempty"`
	MountPoint string `json:"mountPoint"`
}

type mountOptions struct {
	json bool
}

func init() {
	var (
		mountDescription = `buildah mount
  mounts a working container's root filesystem for manipulation.

  Note:  In rootless mode you need to first execute buildah unshare, to put you
  into the usernamespace. Afterwards you can buildah mount the container and
  view/modify the content in the containers root file system.
`
		opts       mountOptions
		noTruncate bool
	)
	mountCommand := &cobra.Command{
		Use:   "mount",
		Short: "Mount a working container's root filesystem",
		Long:  mountDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mountCmd(cmd, args, opts)
		},
		Example: `buildah mount
  buildah mount containerID
  buildah mount containerID1 containerID2

  In rootless mode you must use buildah unshare first.
  buildah unshare
  buildah mount containerID
`,
	}
	mountCommand.SetUsageTemplate(UsageTemplate())

	flags := mountCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolVar(&opts.json, "json", false, "output in JSON format")
	flags.BoolVar(&noTruncate, "notruncate", false, "do not truncate output")
	rootCmd.AddCommand(mountCommand)
	if err := flags.MarkHidden("notruncate"); err != nil {
		logrus.Fatalf("error marking notruncate as hidden: %v", err)
	}
}

func mountCmd(c *cobra.Command, args []string, opts mountOptions) error {
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}
	var jsonMounts []jsonMount
	var lastError error
	if len(args) > 0 {
		// Do not allow to mount a graphdriver that is not vfs if we are creating the userns as part
		// of the mount command.
		// Differently, allow the mount if we are already in a userns, as the mount point will still
		// be accessible once "buildah mount" exits.
		if os.Geteuid() != 0 && store.GraphDriverName() != "vfs" {
			return errors.Errorf("cannot mount using driver %s in rootless mode. You need to run it in a `buildah unshare` session", store.GraphDriverName())
		}

		for _, name := range args {
			builder, err := openBuilder(getContext(), store, name)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error reading build container %q", name)
				continue
			}
			mountPoint, err := builder.Mount(builder.MountLabel)
			if err != nil {
				if lastError != nil {
					fmt.Fprintln(os.Stderr, lastError)
				}
				lastError = errors.Wrapf(err, "error mounting %q container %q", name, builder.Container)
				continue
			}
			if len(args) > 1 {
				if opts.json {
					jsonMounts = append(jsonMounts, jsonMount{Container: name, MountPoint: mountPoint})
					continue
				}
				fmt.Printf("%s %s\n", name, mountPoint)
			} else {
				if opts.json {
					jsonMounts = append(jsonMounts, jsonMount{MountPoint: mountPoint})
					continue
				}
				fmt.Printf("%s\n", mountPoint)
			}
		}
	} else {
		builders, err := openBuilders(store)
		if err != nil {
			return errors.Wrapf(err, "error reading build containers")
		}

		for _, builder := range builders {
			mounted, err := builder.Mounted()
			if err != nil {
				return err
			}
			if mounted {
				if opts.json {
					jsonMounts = append(jsonMounts, jsonMount{Container: builder.Container, MountPoint: builder.MountPoint})
					continue
				}
				fmt.Printf("%s %s\n", builder.Container, builder.MountPoint)
			}
		}
	}

	if opts.json {
		data, err := json.MarshalIndent(jsonMounts, "", "    ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", data)
	}

	return lastError
}
