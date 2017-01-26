package main

import (
	"bytes"

	"github.com/Sirupsen/logrus"
	"github.com/opencontainers/runtime-tools/generate"
)

var (
	configFlags = []cli.Flag{}
)

func updateConfig(c *cli.Context, config []byte) []byte {
	buffer := bytes.Buffer{}
	g, err := generate.NewFromTemplate(bytes.NewReader(config))
	if err != nil {
		logrus.Errorf("error importing template configuration, using original configuration")
		return config
	}
	options := generate.ExportOptions{}
	err = g.Save(buffer, options)
	if err != nil {
		logrus.Errorf("error exporting updated configuration, using original configuration")
		return config
	}
	return buffer.Bytes()
}
