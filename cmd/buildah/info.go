package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"text/template"

	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	infoDescription = "The information displayed pertains to the host and current storage statistics which is useful when reporting issues."
	infoFlags       = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, D",
			Usage: "display additional debug information",
		},
		cli.StringFlag{
			Name:  "format",
			Usage: "use `format` as a Go template to format the output",
		},
	}
	infoCommand = cli.Command{
		Name:                   "info",
		Usage:                  "Display Buildah system information",
		Description:            infoDescription,
		Action:                 infoCmd,
		Flags:                  sortFlags(infoFlags),
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

func infoCmd(c *cli.Context) error {
	if len(c.Args()) > 0 {
		return errors.New("'buildah info' does not accept arguments")
	}

	if err := parse.ValidateFlags(c, infoFlags); err != nil {
		return err
	}
	info := map[string]interface{}{}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	infoArr, err := buildah.Info(store)
	if err != nil {
		return errors.Wrapf(err, "error getting info")
	}

	if c.Bool("debug") {
		debugInfo := debugInfo(c)
		infoArr = append(infoArr, buildah.InfoData{Type: "debug", Data: debugInfo})
	}

	for _, currInfo := range infoArr {
		info[currInfo.Type] = currInfo.Data
	}

	if c.IsSet("format") {
		format := c.String("format")
		if matched, err := regexp.MatchString("{{.*}}", format); err != nil {
			return errors.Wrapf(err, "error validating format provided: %s", format)
		} else if !matched {
			return errors.Errorf("error invalid format provided: %s", format)
		}
		t, err := template.New("format").Parse(format)
		if err != nil {
			return errors.Wrapf(err, "Template parsing error")
		}
		if err = t.Execute(os.Stdout, info); err != nil {
			return err
		}
		if terminal.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Println()
		}
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "    ")
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		enc.SetEscapeHTML(false)
	}
	return enc.Encode(info)
}

// top-level "debug" info
func debugInfo(c *cli.Context) map[string]interface{} {
	info := map[string]interface{}{}
	info["compiler"] = runtime.Compiler
	info["go version"] = runtime.Version()
	info["buildah version"] = buildah.Version
	info["git commit"] = GitCommit
	return info
}
