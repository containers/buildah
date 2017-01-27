package main

import (
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli"
)

var (
	configFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "author",
			Usage: "image author contact information",
		},
		cli.StringFlag{
			Name:  "arch",
			Usage: "image target architecture",
		},
		cli.StringFlag{
			Name:  "os",
			Usage: "image target operating system",
		},
		cli.StringFlag{
			Name:  "user",
			Usage: "user name to run containers as",
		},
		cli.StringSliceFlag{
			Name:  "port",
			Usage: "ports to expose when running containers",
		},
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "environment variable to set when running containers",
		},
		cli.StringSliceFlag{
			Name:  "entrypoint",
			Usage: "container entry point",
		},
		cli.StringSliceFlag{
			Name:  "cmd",
			Usage: "container command",
		},
		cli.StringSliceFlag{
			Name:  "volume",
			Usage: "container volume",
		},
		cli.StringFlag{
			Name:  "workingdir",
			Usage: "container working directory",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "container label",
		},
	}
)

func updateConfig(c *cli.Context, config []byte) []byte {
	image := v1.Image{}
	if err := json.Unmarshal(config, &image); err != nil {
		if len(config) > 0 {
			logrus.Errorf("error importing image configuration, using original configuration")
			return config
		}
	}
	createdBytes, err := time.Now().MarshalText()
	if err != nil {
		logrus.Errorf("error setting image creation time: %v", err)
	} else {
		image.Created = string(createdBytes)
	}
	if image.Architecture == "" {
		image.Architecture = runtime.GOARCH
	}
	if image.OS == "" {
		image.OS = runtime.GOOS
	}
	if c.IsSet("author") {
		image.Author = c.String("author")
	}
	if c.IsSet("arch") {
		image.Architecture = c.String("arch")
	}
	if c.IsSet("os") {
		image.OS = c.String("os")
	}
	if c.IsSet("env") {
		for _, envSpec := range c.StringSlice("env") {
			image.Config.Env = append(append([]string{}, image.Config.Env...), envSpec)
		}
	}
	if c.IsSet("label") {
		for _, labelSpec := range c.StringSlice("label") {
			label := strings.SplitN(labelSpec, "=", 2)
			if image.Config.Labels == nil {
				image.Config.Labels = make(map[string]string)
			}
			if len(label) > 1 {
				image.Config.Labels[label[0]] = label[1]
			} else {
				delete(image.Config.Labels, label[0])
			}
		}
	}
	if c.IsSet("workingdir") {
		image.Config.WorkingDir = c.String("workingdir")
	}
	updatedImage, err := json.Marshal(&image)
	if err != nil {
		logrus.Errorf("error exporting updated image configuration, using original configuration")
		return config
	}
	return updatedImage
}
