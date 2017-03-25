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
		if needToShutdownStore {
			store, err := getStore(c)
			if err != nil {
				return err
			}
			store.Shutdown(false)
		}
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:        "from",
			Usage:       "Create a working container based on an image",
			Description: fromDescription,
			Flags:       fromFlags,
			Action:      fromCmd,
			ArgsUsage:   "IMAGE [CONTAINER-NAME]",
		},
		{
			Name:        "list",
			Usage:       "List working containers and their base images",
			Description: listDescription,
			Flags:       listFlags,
			Action:      listCmd,
			ArgsUsage:   " ",
		},
		{
			Name:        "mount",
			Usage:       "Mount a working container's filesystem root",
			Description: mountDescription,
			Action:      mountCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
		{
			Name:        "umount",
			Aliases:     []string{"unmount"},
			Usage:       "Unmount a working container's filesystem root",
			Description: umountDescription,
			Action:      umountCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
		{
			Name:        "add",
			Usage:       "Add content to the container",
			Description: addDescription,
			Action:      addCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...] [DESTINATION]",
		},
		{
			Name:        "copy",
			Usage:       "Copy content into the container",
			Description: copyDescription,
			Action:      copyCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID [[FILE | DIRECTORY | URL] ...] [DESTINATION]",
		},
		{
			Name:        "run",
			Usage:       "Run a command inside of the container",
			Description: runDescription,
			Flags:       runFlags,
			Action:      runCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID COMMAND [ARGS [...]]",
		},
		{
			Name:        "config",
			Usage:       "Update image configuration settings",
			Description: configDescription,
			Flags:       configFlags,
			Action:      configCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID",
		},
		{
			Name:        "commit",
			Usage:       "Create an image from a working container",
			Description: commitDescription,
			Flags:       commitFlags,
			Action:      commitCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID IMAGE",
		},
		{
			Name:        "delete",
			Usage:       "Delete working container(s)",
			Description: deleteDescription,
			Action:      deleteCmd,
			ArgsUsage:   "CONTAINER-NAME-OR-ID [...]",
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		logrus.Errorf("%v", err)
		os.Exit(1)
	}
}
