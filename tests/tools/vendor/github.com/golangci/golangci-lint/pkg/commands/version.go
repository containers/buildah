package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/golangci/golangci-lint/pkg/config"
)

type versionInfo struct {
	Info      BuildInfo
	BuildInfo *debug.BuildInfo
}

func (e *Executor) initVersionConfiguration(cmd *cobra.Command) {
	fs := cmd.Flags()
	fs.SortFlags = false // sort them as they are defined here
	initVersionFlagSet(fs, e.cfg)
}

func initVersionFlagSet(fs *pflag.FlagSet, cfg *config.Config) {
	// Version config
	vc := &cfg.Version
	fs.StringVar(&vc.Format, "format", "", wh("The version's format can be: 'short', 'json'"))
	fs.BoolVar(&vc.Debug, "debug", false, wh("Add build information"))
}

func (e *Executor) initVersion() {
	versionCmd := &cobra.Command{
		Use:               "version",
		Short:             "Version",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if e.cfg.Version.Debug {
				info, ok := debug.ReadBuildInfo()
				if !ok {
					return nil
				}

				switch strings.ToLower(e.cfg.Version.Format) {
				case "json":
					return json.NewEncoder(os.Stdout).Encode(versionInfo{
						Info:      e.buildInfo,
						BuildInfo: info,
					})

				default:
					fmt.Println(info.String())
					return printVersion(os.Stdout, e.buildInfo)
				}
			}

			switch strings.ToLower(e.cfg.Version.Format) {
			case "short":
				fmt.Println(e.buildInfo.Version)
				return nil

			case "json":
				return json.NewEncoder(os.Stdout).Encode(e.buildInfo)

			default:
				return printVersion(os.Stdout, e.buildInfo)
			}
		},
	}

	e.rootCmd.AddCommand(versionCmd)
	e.initVersionConfiguration(versionCmd)
}

func printVersion(w io.Writer, buildInfo BuildInfo) error {
	_, err := fmt.Fprintf(w, "golangci-lint has version %s built with %s from %s on %s\n",
		buildInfo.Version, buildInfo.GoVersion, buildInfo.Commit, buildInfo.Date)
	return err
}
