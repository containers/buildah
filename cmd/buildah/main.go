package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/config"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	ispecs "github.com/opencontainers/image-spec/specs-go"
	rspecs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type globalFlags struct {
	Debug                      bool
	LogLevel                   string
	Root                       string
	RunRoot                    string
	StorageDriver              string
	RegistriesConf             string
	RegistriesConfDir          string
	DefaultMountsFile          string
	StorageOpts                []string
	UserNSUID                  []string
	UserNSGID                  []string
	CPUProfile                 string
	cpuProfileFile             *os.File
	MemoryProfile              string
	UserShortNameAliasConfPath string
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
	rootCmd.Version = fmt.Sprintf("%s (image-spec %s, runtime-spec %s)", define.Version, ispecs.Version, rspecs.Version)
	rootCmd.PersistentFlags().BoolVar(&globalFlagResults.Debug, "debug", false, "print debugging information")
	// TODO Need to allow for environment variable
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.RegistriesConf, "registries-conf", "", "path to registries.conf file (not usually used)")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.RegistriesConfDir, "registries-conf-dir", "", "path to registries.conf.d directory (not usually used)")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.UserShortNameAliasConfPath, "short-name-alias-conf", "", "path to short name alias cache file (not usually used)")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.Root, "root", storageOptions.GraphRoot, "storage root dir")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.RunRoot, "runroot", storageOptions.RunRoot, "storage state dir")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.StorageDriver, "storage-driver", storageOptions.GraphDriverName, "storage-driver")
	rootCmd.PersistentFlags().StringSliceVar(&globalFlagResults.StorageOpts, "storage-opt", defaultStoreDriverOptions, "storage driver option")
	rootCmd.PersistentFlags().StringSliceVar(&globalFlagResults.UserNSUID, "userns-uid-map", []string{}, "default `ctrID:hostID:length` UID mapping to use")
	rootCmd.PersistentFlags().StringSliceVar(&globalFlagResults.UserNSGID, "userns-gid-map", []string{}, "default `ctrID:hostID:length` GID mapping to use")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.DefaultMountsFile, "default-mounts-file", "", "path to default mounts file")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.LogLevel, logLevel, "warn", `The log level to be used. Either "trace", "debug", "info", "warn", "error", "fatal", or "panic".`)
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.CPUProfile, "cpu-profile", "", "`file` to write CPU profile")
	rootCmd.PersistentFlags().StringVar(&globalFlagResults.MemoryProfile, "memory-profile", "", "`file` to write memory profile")

	if err := rootCmd.PersistentFlags().MarkHidden("cpu-profile"); err != nil {
		logrus.Fatalf("unable to mark cpu-profile flag as hidden: %v", err)
	}
	if err := rootCmd.PersistentFlags().MarkHidden("debug"); err != nil {
		logrus.Fatalf("unable to mark debug flag as hidden: %v", err)
	}
	if err := rootCmd.PersistentFlags().MarkHidden("default-mounts-file"); err != nil {
		logrus.Fatalf("unable to mark default-mounts-file flag as hidden: %v", err)
	}
	if err := rootCmd.PersistentFlags().MarkHidden("memory-profile"); err != nil {
		logrus.Fatalf("unable to mark memory-profile flag as hidden: %v", err)
	}
}

func initConfig() {
	// TODO Cobra allows us to do extra stuff here at init
	// time if we ever want to take advantage.
}

const logLevel = "log-level"

func before(cmd *cobra.Command) error {
	strLvl, err := cmd.Flags().GetString(logLevel)
	if err != nil {
		return err
	}
	logrusLvl, err := logrus.ParseLevel(strLvl)
	if err != nil {
		return errors.Wrapf(err, "unable to parse log level")
	}
	logrus.SetLevel(logrusLvl)
	if globalFlagResults.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	switch cmd.Use {
	case "", "help", "version", "mount":
		return nil
	}
	unshare.MaybeReexecUsingUserNamespace(false)
	if globalFlagResults.CPUProfile != "" {
		globalFlagResults.cpuProfileFile, err = os.Create(globalFlagResults.CPUProfile)
		if err != nil {
			logrus.Fatalf("could not create CPU profile %s: %v", globalFlagResults.CPUProfile, err)
		}
		if err = pprof.StartCPUProfile(globalFlagResults.cpuProfileFile); err != nil {
			logrus.Fatalf("error starting CPU profiling: %v", err)
		}
	}

	defaultContainerConfig, err := config.Default()
	if err != nil {
		return err
	}

	for _, env := range defaultContainerConfig.Engine.Env {
		splitEnv := strings.SplitN(env, "=", 2)
		if len(splitEnv) != 2 {
			return fmt.Errorf("invalid environment variable %q from containers.conf, valid configuration is KEY=value pair", env)
		}
		// skip if the env is already defined
		if _, ok := os.LookupEnv(splitEnv[0]); ok {
			logrus.Debugf("environment variable %q is already defined, skip the settings from containers.conf", splitEnv[0])
			continue
		}
		if err := os.Setenv(splitEnv[0], splitEnv[1]); err != nil {
			return err
		}
	}

	return nil
}

func shutdownStore(cmd *cobra.Command) error {
	if needToShutdownStore {
		store, err := getStore(cmd)
		if err != nil {
			return err
		}
		logrus.Debugf("shutting down the store")
		needToShutdownStore = false
		if _, err = store.Shutdown(false); err != nil {
			if errors.Cause(err) == storage.ErrLayerUsedByContainer {
				logrus.Infof("failed to shutdown storage: %q", err)
			} else {
				logrus.Warnf("failed to shutdown storage: %q", err)
			}
		}
	}
	return nil
}

func after(cmd *cobra.Command) error {
	if err := shutdownStore(cmd); err != nil {
		return err
	}

	if globalFlagResults.CPUProfile != "" {
		pprof.StopCPUProfile()
		globalFlagResults.cpuProfileFile.Close()
	}
	if globalFlagResults.MemoryProfile != "" {
		memoryProfileFile, err := os.Create(globalFlagResults.MemoryProfile)
		if err != nil {
			logrus.Fatalf("could not create memory profile %s: %v", globalFlagResults.MemoryProfile, err)
		}
		defer memoryProfileFile.Close()
		runtime.GC()
		if err := pprof.Lookup("heap").WriteTo(memoryProfileFile, 1); err != nil {
			logrus.Fatalf("could not write memory profile %s: %v", globalFlagResults.MemoryProfile, err)
		}
	}
	return nil
}

func main() {
	if buildah.InitReexec() {
		return
	}

	// Hard code TMPDIR functions to use $TMPDIR or /var/tmp
	os.Setenv("TMPDIR", parse.GetTempDir())

	if err := rootCmd.Execute(); err != nil {
		if logrus.IsLevelEnabled(logrus.TraceLevel) {
			fmt.Fprintf(os.Stderr, "%+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		exitCode := cli.ExecErrorCodeGeneric
		if ee, ok := (errors.Cause(err)).(*exec.ExitError); ok {
			if w, ok := ee.Sys().(syscall.WaitStatus); ok {
				exitCode = w.ExitStatus()
			}
		}
		if err := shutdownStore(rootCmd); err != nil {
			logrus.Warnf("failed to shutdown storage: %q", err)
		}
		os.Exit(exitCode)
	}
}
