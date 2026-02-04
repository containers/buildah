package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/containerd/platforms"
	"github.com/containers/buildah/define"
	ispecs "github.com/opencontainers/image-spec/specs-go"
	rspecs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
	iversion "go.podman.io/image/v5/version"
)

// Overwritten at build time
var (
	GitCommit string
	buildInfo string
)

type versionInfo struct {
	Version       string `json:"version"`
	GoVersion     string `json:"goVersion"`
	ImageSpec     string `json:"imageSpec"`
	RuntimeSpec   string `json:"runtimeSpec"`
	ImageVersion  string `json:"imageVersion"`
	GitCommit     string `json:"gitCommit"`
	Built         string `json:"built"`
	OsArch        string `json:"osArch"`
	BuildPlatform string `json:"buildPlatform"`
}

type versionOptions struct {
	json bool
}

func init() {
	var opts versionOptions

	// cli command to print out the version info of buildah
	versionCommand := &cobra.Command{
		Use:   "version",
		Short: "Display the Buildah version information",
		Long:  "Displays Buildah version information.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return versionCmd(opts)
		},
		Args:    cobra.NoArgs,
		Example: `buildah version`,
	}
	versionCommand.SetUsageTemplate(UsageTemplate())

	flags := versionCommand.Flags()
	flags.BoolVar(&opts.json, "json", false, "output in JSON format")

	rootCmd.AddCommand(versionCommand)
}

func versionCmd(opts versionOptions) error {
	var err error
	buildTime := int64(0)
	if buildInfo != "" {
		// converting unix time from string to int64
		buildTime, err = strconv.ParseInt(buildInfo, 10, 64)
		if err != nil {
			return err
		}
	}

	version := versionInfo{
		Version:       define.Version,
		GoVersion:     runtime.Version(),
		ImageSpec:     ispecs.Version,
		RuntimeSpec:   rspecs.Version,
		ImageVersion:  iversion.Version,
		GitCommit:     GitCommit,
		Built:         time.Unix(buildTime, 0).Format(time.ANSIC),
		OsArch:        runtime.GOOS + "/" + runtime.GOARCH,
		BuildPlatform: platforms.DefaultString(),
	}

	if opts.json {
		data, err := json.MarshalIndent(version, "", "    ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", data)
		return nil
	}

	fmt.Println("Version:        ", version.Version)
	fmt.Println("Go Version:     ", version.GoVersion)
	fmt.Println("Image Spec:     ", version.ImageSpec)
	fmt.Println("Runtime Spec:   ", version.RuntimeSpec)
	fmt.Println("image Version:  ", version.ImageVersion)
	fmt.Println("Git Commit:     ", version.GitCommit)

	// Prints out the build time in readable format
	fmt.Println("Built:          ", version.Built)
	fmt.Println("OS/Arch:        ", version.OsArch)
	fmt.Println("BuildPlatform:  ", version.BuildPlatform)

	return nil
}
