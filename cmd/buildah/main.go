package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/reexec"
	"github.com/nalind/buildah"
	"github.com/urfave/cli"
)

func main() {
	if reexec.Init() {
		return
	}

	app := cli.NewApp()
	app.Name = buildah.Package
	app.Usage = "an image builder"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "root",
			Usage: "storage root dir",
		},
		cli.StringFlag{
			Name:  "runroot",
			Usage: "storage state dir",
		},
		cli.BoolFlag{
			Name:  "storage-driver",
			Usage: "storage driver",
		},
		cli.StringSliceFlag{
			Name:  "storage-option",
			Usage: "storage driver option",
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
			Aliases:     []string{"f"},
			Usage:       "create a working container based on an image",
			Description: "creates a working container based on an image",
			Flags:       fromFlags,
			Action:      fromCmd,
		},
		{
			Name:        "mount",
			Aliases:     []string{"m"},
			Usage:       "mount and create a symbolic link to a working container's filesystem root",
			Description: "mounts and creates a symbolic link to a working container's filesystem root",
			Flags:       mountFlags,
			Action:      mountCmd,
		},
		{
			Name:        "umount",
			Aliases:     []string{"u", "unmount"},
			Usage:       "unmount and remove a symbolic link to a working container's filesystem root",
			Description: "unmounts and removes a symbolic link to a working container's filesystem root",
			Flags:       umountFlags,
			Action:      umountCmd,
		},
		{
			Name:        "run",
			Usage:       "run a command inside of the container",
			Description: "runs a command using the container's root filesystem",
			Flags:       append(runFlags, runConfigurationFlags...),
			Action:      runCmd,
		},
		{
			Name:        "config",
			Usage:       "update image configuration settings",
			Description: "updates a working container's image configuration settings",
			Flags:       append(configFlags, configurationFlags...),
			Action:      configCmd,
		},
		{
			Name:        "commit",
			Aliases:     []string{"c"},
			Usage:       "create an image from a working container",
			Description: "creates an image from a working container",
			Flags:       append(commitFlags, configurationFlags...),
			Action:      commitCmd,
		},
		{
			Name:        "delete",
			Aliases:     []string{"d"},
			Usage:       "delete a working container",
			Description: "deletes a working container",
			Flags:       deleteFlags,
			Action:      deleteCmd,
		},
	}
	app.Run(os.Args)
}
