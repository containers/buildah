package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"text/template"

	"github.com/containers/buildah"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

type infoResults struct {
	debug  bool
	format string
}

func init() {
	var (
		infoDescription = "\n  Display information about the host and current storage statistics which are useful when reporting issues."
		opts            infoResults
	)
	infoCommand := &cobra.Command{
		Use:   "info",
		Short: "Display Buildah system information",
		Long:  infoDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return infoCmd(cmd, opts)
		},
		Args:    cobra.NoArgs,
		Example: `buildah info`,
	}
	infoCommand.SetUsageTemplate(UsageTemplate())

	flags := infoCommand.Flags()
	flags.BoolVarP(&opts.debug, "debug", "d", false, "display additional debug information")
	flags.StringVar(&opts.format, "format", "", "use `format` as a Go template to format the output")
	rootCmd.AddCommand(infoCommand)
}

func infoCmd(c *cobra.Command, iopts infoResults) error {
	info := map[string]interface{}{}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	infoArr, err := buildah.Info(store)
	if err != nil {
		return errors.Wrapf(err, "error getting info")
	}

	if iopts.debug {
		debugInfo := debugInfo()
		infoArr = append(infoArr, buildah.InfoData{Type: "debug", Data: debugInfo})
	}

	for _, currInfo := range infoArr {
		info[currInfo.Type] = currInfo.Data
	}

	if iopts.format != "" {
		format := iopts.format
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
func debugInfo() map[string]interface{} {
	info := map[string]interface{}{}
	info["compiler"] = runtime.Compiler
	info["go version"] = runtime.Version()
	info["buildah version"] = buildah.Version
	info["git commit"] = GitCommit
	return info
}
