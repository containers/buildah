package main

import (
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	// DefaultCreatedBy is the default description of how an image layer
	// was created that we use when adding to an image's history.
	DefaultCreatedBy = "manual edits"
)

var (
	configFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "annotation, a",
			Usage: "add `annotation` e.g. annotation=value, for the target image",
		},
		cli.StringFlag{
			Name:  "arch",
			Usage: "set `architecture` of the target image",
		},
		cli.StringFlag{
			Name:  "author",
			Usage: "set image author contact `information`",
		},
		cli.StringFlag{
			Name:  "cmd",
			Usage: "sets the default `command` to run for containers based on the image",
		},
		cli.StringFlag{
			Name:  "created-by",
			Usage: "add `description` of how the image was created",
			Value: DefaultCreatedBy,
		},
		cli.StringFlag{
			Name:  "entrypoint",
			Usage: "set `entry point` for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "add `environment variable` to be set when running containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "label, l",
			Usage: "add image configuration `label` e.g. label=value",
		},
		cli.StringFlag{
			Name:  "os",
			Usage: "set `operating system` of the target image",
		},
		cli.StringSliceFlag{
			Name:  "port, p",
			Usage: "add `port` to expose when running containers based on image",
		},
		cli.StringFlag{
			Name:  "user, u",
			Usage: "set default `user` to run inside containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "volume, v",
			Usage: "add default `volume` path to be created for containers based on image",
		},
		cli.StringFlag{
			Name:  "workingdir",
			Usage: "set working `directory` for containers based on image",
		},
	}
	configDescription = "Modifies the configuration values which will be saved to the image"
	configCommand     = cli.Command{
		Name:        "config",
		Usage:       "Update image configuration settings",
		Description: configDescription,
		Flags:       configFlags,
		Action:      configCmd,
		ArgsUsage:   "CONTAINER-NAME-OR-ID",
	}
)

func updateConfig(builder *buildah.Builder, c *cli.Context) {
	if c.IsSet("author") {
		builder.SetMaintainer(c.String("author"))
	}
	if c.IsSet("created-by") {
		builder.SetCreatedBy(c.String("created-by"))
	}
	if c.IsSet("arch") {
		builder.SetArchitecture(c.String("arch"))
	}
	if c.IsSet("os") {
		builder.SetOS(c.String("os"))
	}
	if c.IsSet("user") {
		builder.SetUser(c.String("user"))
	}
	if c.IsSet("port") || c.IsSet("p") {
		for _, portSpec := range c.StringSlice("port") {
			builder.SetPort(portSpec)
		}
	}
	if c.IsSet("env") || c.IsSet("e") {
		for _, envSpec := range c.StringSlice("env") {
			env := strings.SplitN(envSpec, "=", 2)
			if len(env) > 1 {
				builder.SetEnv(env[0], env[1])
			} else {
				builder.UnsetEnv(env[0])
			}
		}
	}
	if c.IsSet("entrypoint") {
		entrypointSpec, err := shellwords.Parse(c.String("entrypoint"))
		if err != nil {
			logrus.Errorf("error parsing --entrypoint %q: %v", c.String("entrypoint"), err)
		} else {
			builder.SetEntrypoint(entrypointSpec)
		}
	}
	if c.IsSet("cmd") {
		cmdSpec, err := shellwords.Parse(c.String("cmd"))
		if err != nil {
			logrus.Errorf("error parsing --cmd %q: %v", c.String("cmd"), err)
		} else {
			builder.SetCmd(cmdSpec)
		}
	}
	if c.IsSet("volume") {
		if volSpec := c.StringSlice("volume"); len(volSpec) > 0 {
			for _, spec := range volSpec {
				builder.AddVolume(spec)
			}
		}
	}
	if c.IsSet("label") || c.IsSet("l") {
		for _, labelSpec := range c.StringSlice("label") {
			label := strings.SplitN(labelSpec, "=", 2)
			if len(label) > 1 {
				builder.SetLabel(label[0], label[1])
			} else {
				builder.UnsetLabel(label[0])
			}
		}
	}
	if c.IsSet("workingdir") {
		builder.SetWorkDir(c.String("workingdir"))
	}
	if c.IsSet("annotation") || c.IsSet("a") {
		for _, annotationSpec := range c.StringSlice("annotation") {
			annotation := strings.SplitN(annotationSpec, "=", 2)
			if len(annotation) > 1 {
				builder.SetAnnotation(annotation[0], annotation[1])
			} else {
				builder.UnsetAnnotation(annotation[0])
			}
		}
	}
}

func configCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	name := args[0]
	if err := validateFlags(c, configFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	updateConfig(builder, c)
	return builder.Save()
}
