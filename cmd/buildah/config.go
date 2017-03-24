package main

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/mattn/go-shellwords"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

const (
	// DefaultCreatedBy is the default description of how an image layer
	// was created that we use when adding to an image's history.
	DefaultCreatedBy = "manual edits"
)

var (
	configFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "author",
			Usage: "image author contact `information`",
		},
		cli.StringFlag{
			Name:  "created-by",
			Usage: "`description` of how the image was created",
			Value: DefaultCreatedBy,
		},
		cli.StringFlag{
			Name:  "arch",
			Usage: "`architecture` of the target image",
		},
		cli.StringFlag{
			Name:  "os",
			Usage: "`operating system` of the target image",
		},
		cli.StringFlag{
			Name:  "user",
			Usage: "`user` to run containers based on image as",
		},
		cli.StringSliceFlag{
			Name:  "port",
			Usage: "`port` to expose when running containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "`environment variable` to set when running containers based on image",
		},
		cli.StringFlag{
			Name:  "entrypoint",
			Usage: "`entry point` for containers based on image",
		},
		cli.StringFlag{
			Name:  "cmd",
			Usage: "`command` for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "volume",
			Usage: "`volume` to create for containers based on image",
		},
		cli.StringFlag{
			Name:  "workingdir",
			Usage: "working `directory` for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "image configuration `label` e.g. label=value",
		},
		cli.StringSliceFlag{
			Name:  "annotation",
			Usage: "`annotation` e.g. annotation=value, for the target image",
		},
	}
	configDescription = "Modifies the configuration values which will be saved to the image"
)

func updateConfig(builder *buildah.Builder, c *cli.Context) {
	if c.IsSet("author") {
		builder.Maintainer = c.String("author")
	}
	if c.IsSet("created-by") {
		builder.CreatedBy = c.String("created-by")
	}
	if c.IsSet("arch") {
		builder.Architecture = c.String("arch")
	}
	if c.IsSet("os") {
		builder.OS = c.String("os")
	}
	if c.IsSet("user") {
		builder.User = c.String("user")
	}
	if c.IsSet("port") {
		if builder.Expose == nil {
			builder.Expose = make(map[string]interface{})
		}
		for _, portSpec := range c.StringSlice("port") {
			builder.Expose[portSpec] = struct{}{}
		}
	}
	if c.IsSet("env") {
		for _, envSpec := range c.StringSlice("env") {
			builder.Env = append(builder.Env, envSpec)
		}
	}
	if c.IsSet("entrypoint") {
		entrypointSpec, err := shellwords.Parse(c.String("entrypoint"))
		if err != nil {
			logrus.Errorf("error parsing --entrypoint %q: %v", c.String("entrypoint"), err)
		} else {
			builder.Entrypoint = entrypointSpec
		}
	}
	if c.IsSet("cmd") {
		cmdSpec, err := shellwords.Parse(c.String("cmd"))
		if err != nil {
			logrus.Errorf("error parsing --cmd %q: %v", c.String("cmd"), err)
		} else {
			builder.Cmd = cmdSpec
		}
	}
	if c.IsSet("volume") {
		for _, volSpec := range c.StringSlice("volume") {
			builder.Volumes = append(builder.Volumes, volSpec)
		}
	}
	if c.IsSet("label") {
		if builder.Labels == nil {
			builder.Labels = make(map[string]string)
		}
		for _, labelSpec := range c.StringSlice("label") {
			label := strings.SplitN(labelSpec, "=", 2)
			if len(label) > 1 {
				builder.Labels[label[0]] = label[1]
			} else {
				delete(builder.Labels, label[0])
			}
		}
	}
	if c.IsSet("workingdir") {
		builder.Workdir = c.String("workingdir")
	}
	if c.IsSet("annotation") {
		if builder.Annotations == nil {
			builder.Annotations = make(map[string]string)
		}
		for _, annotationSpec := range c.StringSlice("annotation") {
			annotation := strings.SplitN(annotationSpec, "=", 2)
			if len(annotation) > 1 {
				builder.Annotations[annotation[0]] = annotation[1]
			} else {
				delete(builder.Annotations, annotation[0])
			}
		}
	}
}

func configCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return fmt.Errorf("container ID must be specified")
	}
	name := args[0]
	args = args.Tail()

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(store, name)
	if err != nil {
		return fmt.Errorf("error reading build container %q: %v", name, err)
	}

	updateConfig(builder, c)
	return builder.Save()
}
