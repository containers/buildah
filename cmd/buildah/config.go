package main

import (
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	docker "github.com/docker/docker/image"
	"github.com/mattn/go-shellwords"
	ociv1 "github.com/opencontainers/image-spec/specs-go/v1"
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
			Usage: "user to run containers based on image as",
		},
		cli.StringSliceFlag{
			Name:  "port",
			Usage: "port to expose when running containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "environment variable to set when running containers based on image",
		},
		cli.StringFlag{
			Name:  "entrypoint",
			Usage: "entry point for containers based on image",
		},
		cli.StringFlag{
			Name:  "cmd",
			Usage: "command for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "volume",
			Usage: "volume to create for containers based on image",
		},
		cli.StringFlag{
			Name:  "workingdir",
			Usage: "initial working directory for containers based on image",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "image label e.g. label=value",
		},
	}
)

func copyDockerImageConfig(dimage *docker.Image) (ociv1.Image, error) {
	created, err := dimage.Created.UTC().MarshalText()
	if err != nil {
		created = []byte{}
	}
	image := ociv1.Image{
		Created:      string(created),
		Author:       dimage.Author,
		Architecture: dimage.Architecture,
		OS:           dimage.OS,
		Config: ociv1.ImageConfig{
			User:         dimage.Config.User,
			ExposedPorts: map[string]struct{}{},
			Env:          dimage.Config.Env,
			Entrypoint:   dimage.Config.Entrypoint,
			Cmd:          dimage.Config.Cmd,
			Volumes:      dimage.Config.Volumes,
			WorkingDir:   dimage.Config.WorkingDir,
			Labels:       dimage.Config.Labels,
		},
		RootFS: ociv1.RootFS{
			Type:    "",
			DiffIDs: []string{},
		},
		History: []ociv1.History{},
	}
	for port, what := range dimage.Config.ExposedPorts {
		image.Config.ExposedPorts[string(port)] = what
	}
	RootFS := docker.RootFS{}
	if dimage.RootFS != nil {
		RootFS = *dimage.RootFS
	}
	if RootFS.Type == docker.TypeLayers {
		image.RootFS.Type = docker.TypeLayers
		for _, id := range RootFS.DiffIDs {
			image.RootFS.DiffIDs = append(image.RootFS.DiffIDs, id.String())
		}
	}
	for _, history := range dimage.History {
		created, err := history.Created.UTC().MarshalText()
		if err != nil {
			created = []byte{}
		}
		ohistory := ociv1.History{
			Created:    string(created),
			CreatedBy:  history.CreatedBy,
			Author:     history.Author,
			Comment:    history.Comment,
			EmptyLayer: history.EmptyLayer,
		}
		image.History = append(image.History, ohistory)
	}
	return image, nil
}

func updateConfig(c *cli.Context, config []byte) []byte {
	image := ociv1.Image{}
	dimage := docker.Image{}
	if err := json.Unmarshal(config, &dimage); err == nil && dimage.DockerVersion != "" {
		logrus.Debugf("attempting to read image configuration as a docker image configuration")
		if image, err = copyDockerImageConfig(&dimage); err != nil {
			logrus.Errorf("error importing docker image configuration, using original configuration")
			return config
		}
	} else {
		logrus.Debugf("attempting to parse image configuration as an OCI image configuration")
		if err := json.Unmarshal(config, &image); err != nil {
			if len(config) > 0 {
				logrus.Errorf("error importing image configuration, using original configuration")
				return config
			}
		}
	}
	createdBytes, err := time.Now().UTC().MarshalText()
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
	if c.IsSet("user") {
		image.Config.User = c.String("user")
	}
	if c.IsSet("port") {
		if image.Config.ExposedPorts == nil {
			image.Config.ExposedPorts = make(map[string]struct{})
		}
		for _, portSpec := range c.StringSlice("port") {
			image.Config.ExposedPorts[portSpec] = struct{}{}
		}
	}
	if c.IsSet("env") {
		for _, envSpec := range c.StringSlice("env") {
			image.Config.Env = append(append([]string{}, image.Config.Env...), envSpec)
		}
	}
	if c.IsSet("entrypoint") {
		entrypointSpec, err := shellwords.Parse(c.String("entrypoint"))
		if err != nil {
			logrus.Errorf("error parsing --entrypoint %q: %v", c.String("entrypoint"), err)
		} else {
			image.Config.Entrypoint = entrypointSpec
		}
	}
	if c.IsSet("cmd") {
		cmdSpec, err := shellwords.Parse(c.String("cmd"))
		if err != nil {
			logrus.Errorf("error parsing --cmd %q: %v", c.String("cmd"), err)
		} else {
			image.Config.Cmd = cmdSpec
		}
	}
	if c.IsSet("volume") {
		if image.Config.Volumes == nil {
			image.Config.Volumes = make(map[string]struct{})
		}
		for _, volSpec := range c.StringSlice("volume") {
			image.Config.Volumes[volSpec] = struct{}{}
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
