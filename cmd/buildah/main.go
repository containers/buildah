package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage"
	ispecs "github.com/opencontainers/image-spec/specs-go"
	rspecs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

func main() {
	var defaultStoreDriverOptions *cli.StringSlice
	if buildah.InitReexec() {
		return
	}

	app := cli.NewApp()
	app.Name = buildah.Package
	app.Version = fmt.Sprintf("%s (image-spec %s, runtime-spec %s)", buildah.Version, ispecs.Version, rspecs.Version)
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
			Name:  "storage-opt",
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
			_, _ = store.Shutdown(false)
		}
		return nil
	}
	app.Commands = []cli.Command{
		addCommand,
		budCommand,
		commitCommand,
		configCommand,
		containersCommand,
		copyCommand,
		exportCommand,
		fromCommand,
		imagesCommand,
		importCommand,
		inspectCommand,
		mountCommand,
		pushCommand,
		rmCommand,
		rmiCommand,
		runCommand,
		tagCommand,
		umountCommand,
		versionCommand,
	}
	err := app.Run(os.Args)
	if err != nil {
		logrus.Errorf("%v", err)
		os.Exit(1)
	}
}
