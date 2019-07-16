package main

import (
	"fmt"
	"os"

	"github.com/containers/buildah"
	"github.com/containers/buildah/pkg/unshare"
	"github.com/containers/storage"
	ispecs "github.com/opencontainers/image-spec/specs-go"
	rspecs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type globalFlags struct {
	Debug             bool
	Root              string
	RunRoot           string
	StorageDriver     string
	RegistriesConf    string
	RegistriesConfDir string
	DefaultMountsFile string
	StorageOpts       []string
	UserNSUID         []string
	UserNSGID         []string
}

var rootCmd = &cobra.Command{
	Use:  "buildah",
	Long: "A tool that facilitates building OCI images",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return before(cmd)
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return after(cmd)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

var (
	globalFlagResults globalFlags
)

func init() {

	var (
		defaultStoreDriverOptions []string
	)
	storageOptions, err := storage.DefaultStoreOptions(false, 0)
	if err != nil {
		logrus.Errorf(err.Error())
		os.Exit(1)

	}

	if len(storageOptions.GraphDriverOptions) > 0 {
		optionSlice := storageOptions.GraphDriverOptions[:]
		defaultStoreDriverOptions = optionSlice
	}

	cobra.OnInitialize(initConfig)
	//rootCmd.TraverseChildren = true
	rootCmd.Version = fmt.Sprintf("%s (image-spec %s, runtime-spec %s)", buildah.Version, ispecs.Version, rspecs.Version)
	rootCmd.PersistentFlags().BoolVar(&globalFlagResults.Debug, "debug", false, "print debugging information")
	// TODO Need to allow for environment variable
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.RegistriesConf, "registries-conf", "", "path to registries.conf file (not usually used)")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.RegistriesConfDir, "registries-conf-dir", "", "path to registries.conf.d directory (not usually used)")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.Root, "root", storageOptions.GraphRoot, "storage root dir")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.RunRoot, "runroot", storageOptions.RunRoot, "storage state dir")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.StorageDriver, "storage-driver", storageOptions.GraphDriverName, "storage-driver")
	rootCmd.PersistentFlags().StringSliceVar(&globalFlagResults.StorageOpts, "storage-opt", defaultStoreDriverOptions, "storage driver option")
	rootCmd.PersistentFlags().StringSliceVar(&globalFlagResults.UserNSUID, "userns-uid-map", []string{}, "default `ctrID:hostID:length` UID mapping to use")
	rootCmd.PersistentFlags().StringSliceVar(&globalFlagResults.UserNSGID, "userns-gid-map", []string{}, "default `ctrID:hostID:length` GID mapping to use")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.DefaultMountsFile, "default-mounts-file", "", "path to default mounts file")

	if err := rootCmd.PersistentFlags().MarkHidden("default-mounts-file"); err != nil {
		fmt.Println("unable to setup menu")
	}
}

func initConfig() {
	// TODO Cobra allows us to do extra stuff here at init
	// time if we ever want to take advantage.
}

func before(cmd *cobra.Command) error {
	llFlag := cmd.Flags().Lookup("log-level")
	llFlagNum := 0
	if llFlag != nil {
		llFlagNum, _ = cmd.Flags().GetInt("log-level")
	}
	logrus.SetLevel(logrus.ErrorLevel + logrus.Level(llFlagNum))
	if globalFlagResults.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	switch cmd.Use {
	case "", "help", "version", "mount":
		return nil
	}
	unshare.MaybeReexecUsingUserNamespace(false)
	return nil
}

func after(cmd *cobra.Command) error {
	if needToShutdownStore {
		store, err := getStore(cmd)
		if err != nil {
			return err
		}
		_, _ = store.Shutdown(false)
	}
	return nil
}

func main() {
	if buildah.InitReexec() {
		return
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
