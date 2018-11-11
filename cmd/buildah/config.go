package main

import (
	"encoding/json"
	"strings"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	configFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "annotation, a",
			Usage: "add `annotation` e.g. annotation=value, for the target image (default [])",
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
			Usage: "set the default `command` to run for containers based on the image",
		},
		cli.StringFlag{
			Name:  "comment",
			Usage: "set a `comment` in the target image",
		},
		cli.StringFlag{
			Name:  "created-by",
			Usage: "set `description` of how the image was created",
		},
		cli.StringFlag{
			Name:  "domainname",
			Usage: "set a domain `name` for containers based on image",
		},
		cli.StringFlag{
			Name:  "entrypoint",
			Usage: "set `entry point` for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "env, e",
			Usage: "add `environment variable` to be set when running containers based on image (default [])",
		},
		cli.StringFlag{
			Name:  "history-comment",
			Usage: "set a `comment` for the history of the target image",
		},
		cli.StringFlag{
			Name:  "hostname",
			Usage: "set a host`name` for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "label, l",
			Usage: "add image configuration `label` e.g. label=value",
		},
		cli.StringSliceFlag{
			Name:  "onbuild",
			Usage: "add onbuild command to be run on images based on this image. Only supported on 'docker' formatted images",
		},
		cli.StringFlag{
			Name:  "os",
			Usage: "set `operating system` of the target image",
		},
		cli.StringSliceFlag{
			Name:  "port, p",
			Usage: "add `port` to expose when running containers based on image (default [])",
		},
		cli.StringFlag{
			Name:  "shell",
			Usage: "add `shell` to run in containers",
		},
		cli.StringFlag{
			Name:  "stop-signal",
			Usage: "set `stop signal` for containers based on image",
		},
		cli.StringFlag{
			Name:  "user, u",
			Usage: "set default `user` to run inside containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "volume, v",
			Usage: "add default `volume` path to be created for containers based on image (default [])",
		},
		cli.StringFlag{
			Name:  "workingdir",
			Usage: "set working `directory` for containers based on image",
		},
	}
	configDescription = "Modifies the configuration values which will be saved to the image"
	configCommand     = cli.Command{
		Name:                   "config",
		Usage:                  "Update image configuration settings",
		Description:            configDescription,
		Flags:                  sortFlags(configFlags),
		Action:                 configCmd,
		ArgsUsage:              "CONTAINER-NAME-OR-ID",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func updateEntrypoint(builder *buildah.Builder, c *cli.Context) {
	if len(strings.TrimSpace(c.String("entrypoint"))) == 0 {
		builder.SetEntrypoint(nil)
		return
	}
	var entrypointJSON []string
	err := json.Unmarshal([]byte(c.String("entrypoint")), &entrypointJSON)

	if err == nil {
		builder.SetEntrypoint(entrypointJSON)
		if len(builder.Cmd()) > 0 {
			logrus.Warnf("cmd %q exists and will be passed to entrypoint as a parameter", strings.Join(builder.Cmd(), " "))
		}
		return
	}

	// it wasn't a valid json array, fall back to string
	entrypointSpec := make([]string, 3)
	entrypointSpec[0] = "/bin/sh"
	entrypointSpec[1] = "-c"
	entrypointSpec[2] = c.String("entrypoint")
	if len(builder.Cmd()) > 0 {
		logrus.Warnf("cmd %q exists but will be ignored because of entrypoint settings", strings.Join(builder.Cmd(), " "))
	}

	builder.SetEntrypoint(entrypointSpec)
}

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
	if c.IsSet("shell") {
		shellSpec, err := shellwords.Parse(c.String("shell"))
		if err != nil {
			logrus.Errorf("error parsing --shell %q: %v", c.String("shell"), err)
		} else {
			builder.SetShell(shellSpec)
		}
	}
	if c.IsSet("stop-signal") {
		builder.SetStopSignal(c.String("stop-signal"))
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
		updateEntrypoint(builder, c)
	}
	// cmd should always run after entrypoint; setting entrypoint clears cmd
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
	if c.IsSet("comment") {
		builder.SetComment(c.String("comment"))
	}
	if c.IsSet("history-comment") {
		builder.SetHistoryComment(c.String("history-comment"))
	}
	if c.IsSet("domainname") {
		builder.SetDomainname(c.String("domainname"))
	}
	if c.IsSet("hostname") {
		builder.SetHostname(c.String("hostname"))
	}
	if c.IsSet("onbuild") {
		for _, onbuild := range c.StringSlice("onbuild") {
			builder.SetOnBuild(onbuild)
		}
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
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	name := args[0]
	if err := parse.ValidateFlags(c, configFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	updateConfig(builder, c)
	return builder.Save()
}
