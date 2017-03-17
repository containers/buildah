package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/storage"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

func main() {
	var defaultStoreDriverOptions *cli.StringSlice
	if reexec.Init() {
		return
	}

	app := cli.NewApp()
	app.Name = buildah.Package
	app.Usage = "an image builder"
	if len(storage.DefaultStoreOptions.GraphDriverOptions) > 0 {
		var optionSlice cli.StringSlice = storage.DefaultStoreOptions.GraphDriverOptions[:]
		defaultStoreDriverOptions = &optionSlice
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "root",
			Usage: "storage root dir",
			Value: storage.DefaultStoreOptions.GraphRoot,
		},
		cli.StringFlag{
			Name:  "runroot",
			Usage: "storage state dir",
			Value: storage.DefaultStoreOptions.RunRoot,
		},
		cli.StringFlag{
			Name:  "storage-driver",
			Usage: "storage driver",
			Value: storage.DefaultStoreOptions.GraphDriverName,
		},
		cli.StringSliceFlag{
			Name:  "storage-option",
			Usage: "storage driver option",
			Value: defaultStoreDriverOptions,
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "print debugging information",
		},
	}
	app.Before = func(c *cli.Context) error {
		logrus.SetLevel(logrus.ErrorLevel)
		if c.GlobalIsSet("debug") {
			if c.GlobalBool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
			}
		}
		return nil
	}
	app.After = func(c *cli.Context) error {
		store, err := getStore(c)
		if err != nil {
			return err
		}
		store.Shutdown(false)
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:        "from",
			Usage:       "create a working container based on an image",
			Description: "creates a working container based on an image",
			Flags:       fromFlags,
			Action:      fromCmd,
			ArgsUsage:   "IMAGE [CONTAINER-NAME]",
		},
		{
			Name:        "list",
			Usage:       "list working containers and their base images",
			Description: "lists working containers and their base images",
			Flags:       listFlags,
			Action:      listCmd,
			ArgsUsage:   " ",
		},
		{
			Name:        "mount",
			Usage:       "mount a working container's filesystem root",
			Description: "mounts a working container's filesystem root",
			Flags:       mountFlags,
			Action:      mountCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
		{
			Name:        "umount",
			Aliases:     []string{"unmount"},
			Usage:       "unmount a working container's filesystem root",
			Description: "unmounts a working container's filesystem root",
			Flags:       umountFlags,
			Action:      umountCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
		{
			Name:        "add",
			Usage:       "add content to the container",
			Description: "add content to the container's filesystem",
			Flags:       addFlags,
			Action:      addCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...]",
		},
		{
			Name:        "copy",
			Usage:       "copy content into the container",
			Description: "copy content into the container's filesystem",
			Flags:       copyFlags,
			Action:      copyCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...]",
		},
		{
			Name:        "run",
			Usage:       "run a command inside of the container",
			Description: "runs a command using the container's root filesystem",
			Flags:       runFlags,
			Action:      runCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID COMMAND [ARGS [...]]",
		},
		{
			Name:        "config",
			Usage:       "update image configuration settings",
			Description: "updates a working container's image configuration settings",
			Flags:       configFlags,
			Action:      configCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
		{
			Name:        "commit",
			Usage:       "create an image from a working container",
			Description: "creates an image from a working container",
			Flags:       commitFlags,
			Action:      commitCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID IMAGE",
		},
		{
			Name:        "delete",
			Usage:       "delete a working container",
			Description: "deletes a working container",
			Flags:       deleteFlags,
			Action:      deleteCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
	}
	app.Run(os.Args)
}
