package main

import (
	"fmt"
	"os"

	"github.com/containers/buildah"
	"github.com/containers/libpod/pkg/util"
	"github.com/containers/storage"
	ispecs "github.com/opencontainers/image-spec/specs-go"
	rspecs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	debug := false

	var defaultStoreDriverOptions *cli.StringSlice
	if buildah.InitReexec() {
		return
	}
	storageOptions, err := util.GetDefaultStoreOptions()
	if err != nil {
		logrus.Errorf(err.Error())
		cli.OsExiter(1)
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
		cli.BoolFlag{
			Name:  "debug",
			Usage: "print debugging information",
		},
		cli.StringFlag{
			Name:   "registries-conf",
			Usage:  "path to registries.conf file (not usually used)",
			EnvVar: "REGISTRIES_CONFIG_PATH",
		},
		cli.StringFlag{
			Name:  "registries-conf-dir",
			Usage: "path to registries.conf.d directory (not usually used)",
		},
		cli.StringFlag{
			Name:  "root",
			Usage: "storage root dir",
			Value: storageOptions.GraphRoot,
		},
		cli.StringFlag{
			Name:  "runroot",
			Usage: "storage state dir",
			Value: storageOptions.RunRoot,
		},
		cli.StringFlag{
			Name:  "storage-driver",
			Usage: "storage driver",
			Value: storageOptions.GraphDriverName,
		},
		cli.StringSliceFlag{
			Name:  "storage-opt",
			Usage: "storage driver option",
			Value: defaultStoreDriverOptions,
		},
		cli.StringSliceFlag{
			Name:  "userns-uid-map",
			Usage: "default `ctrID:hostID:length` UID mapping to use",
		},
		cli.StringSliceFlag{
			Name:  "userns-gid-map",
			Usage: "default `ctrID:hostID:length` GID mapping to use",
		},
		cli.StringFlag{
			Name:   "default-mounts-file",
			Usage:  "path to default mounts file",
			Hidden: true,
		},
	}
	app.Before = func(c *cli.Context) error {
		logrus.SetLevel(logrus.ErrorLevel + logrus.Level(c.Int("loglevel")))
		if c.GlobalBool("debug") {
			debug = true
			logrus.SetLevel(logrus.DebugLevel)
		}
		maybeReexecUsingUserNamespace(c, false)
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
		fromCommand,
		imagesCommand,
		infoCommand,
		inspectCommand,
		mountCommand,
		pullCommand,
		pushCommand,
		renameCommand,
		rmCommand,
		rmiCommand,
		runCommand,
		tagCommand,
		umountCommand,
		unshareCommand,
		versionCommand,
	}
	if err := app.Run(os.Args); err != nil {
		if debug {
			logrus.Errorf(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		cli.OsExiter(1)
	}
}
