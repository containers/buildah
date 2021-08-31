package main

import (
	"context"
	"os"

	"github.com/containers/buildah"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	imageStorage "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var storeOptions storage.StoreOptions
	var systemContext types.SystemContext
	var logLevel string
	var maxParallelDownloads uint

	if buildah.InitReexec() {
		return
	}

	unshare.MaybeReexecUsingUserNamespace(false)

	rootCmd := &cobra.Command{
		Use:  "copy [flags] source destination",
		Long: "copies an image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return err
			}

			level, err := logrus.ParseLevel(logLevel)
			if err != nil {
				return err
			}
			logrus.SetLevel(level)

			store, err := storage.GetStore(storeOptions)
			if err != nil {
				return err
			}
			imageStorage.Transport.SetStore(store)

			if len(args) < 1 {
				return errors.Wrapf(err, "no source name provided")
			}
			src, err := alltransports.ParseImageName(args[0])
			if err != nil {
				return errors.Wrapf(err, "error parsing source name")
			}
			if len(args) < 1 {
				return errors.Wrapf(err, "no destination name provided")
			}
			dest, err := alltransports.ParseImageName(args[1])
			if err != nil {
				return errors.Wrapf(err, "error parsing destination name")
			}

			policy, err := signature.DefaultPolicy(&systemContext)
			if err != nil {
				return errors.Wrapf(err, "error reading signature policy")
			}
			policyContext, err := signature.NewPolicyContext(policy)
			if err != nil {
				return errors.Wrapf(err, "error creating new signature policy context")
			}
			defer func() {
				if err := policyContext.Destroy(); err != nil {
					logrus.Error(errors.Wrapf(err, "error destroying signature policy context"))
				}
			}()

			options := cp.Options{
				ReportWriter:         os.Stdout,
				SourceCtx:            &systemContext,
				DestinationCtx:       &systemContext,
				MaxParallelDownloads: maxParallelDownloads,
			}
			if _, err = cp.Image(context.TODO(), policyContext, dest, src, &options); err != nil {
				return err
			}

			defer func() {
				_, err := store.Shutdown(false)
				if err != nil {
					logrus.Error(err)
				}
			}()
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&storeOptions.GraphRoot, "root", "", "storage root")
	rootCmd.PersistentFlags().StringVar(&storeOptions.RunRoot, "runroot", "", "runtime root")
	rootCmd.PersistentFlags().StringVar(&storeOptions.GraphDriverName, "storage-driver", "", "storage driver")
	rootCmd.PersistentFlags().StringSliceVar(&storeOptions.GraphDriverOptions, "storage-opt", nil, "storage option")
	rootCmd.PersistentFlags().StringVar(&systemContext.SystemRegistriesConfPath, "registries-conf", "", "location of registries.conf")
	rootCmd.PersistentFlags().StringVar(&systemContext.SystemRegistriesConfDirPath, "registries-conf-dir", "", "location of registries.d")
	rootCmd.PersistentFlags().StringVar(&systemContext.SignaturePolicyPath, "signature-policy", "", "`pathname` of signature policy file")
	rootCmd.PersistentFlags().StringVar(&systemContext.UserShortNameAliasConfPath, "short-name-alias-conf", "", "`pathname` of short name alias cache file (not usually used)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "logging level")
	rootCmd.PersistentFlags().UintVar(&maxParallelDownloads, "max-parallel-downloads", 0, "maximum `number` of blobs to copy at once")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
