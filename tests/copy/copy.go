package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/common/libnetwork/network"
	"github.com/containers/common/pkg/config"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/containers/image/v5/signature"
	imageStorage "github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/unshare"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	var storeOptions storage.StoreOptions
	var systemContext types.SystemContext
	var logLevel string
	var maxParallelDownloads uint
	var compressionFormat string
	var manifestFormat string
	compressionLevel := -1

	if buildah.InitReexec() {
		return
	}

	unshare.MaybeReexecUsingUserNamespace(false)

	storeOptions, err := storage.DefaultStoreOptions()
	if err != nil {
		storeOptions = storage.StoreOptions{}
	}

	rootCmd := &cobra.Command{
		Use:  "copy [flags] source destination",
		Long: "copies an image",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return err
			}
			if compressionLevel != -1 {
				systemContext.CompressionLevel = &compressionLevel
			}
			if compressionFormat != "" {
				alg, err := compression.AlgorithmByName(compressionFormat)
				if err != nil {
					return err
				}
				systemContext.CompressionFormat = &alg
			}
			switch strings.ToLower(manifestFormat) {
			case "oci":
				manifestFormat = v1.MediaTypeImageManifest
			case "docker", "dockerv2s2":
				manifestFormat = manifest.DockerV2Schema2MediaType
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

			conf, err := config.Default()
			if err != nil {
				return err
			}
			_, _, err = network.NetworkBackend(store, conf, false)
			if err != nil {
				return err
			}

			if len(args) < 1 {
				return errors.New("no source name provided")
			}
			src, err := alltransports.ParseImageName(args[0])
			if err != nil {
				return fmt.Errorf("parsing source name: %w", err)
			}
			if len(args) < 1 {
				return errors.New("no destination name provided")
			}
			dest, err := alltransports.ParseImageName(args[1])
			if err != nil {
				return fmt.Errorf("parsing destination name: %w", err)
			}

			policy, err := signature.DefaultPolicy(&systemContext)
			if err != nil {
				return fmt.Errorf("reading signature policy: %w", err)
			}
			policyContext, err := signature.NewPolicyContext(policy)
			if err != nil {
				return fmt.Errorf("creating new signature policy context: %w", err)
			}
			defer func() {
				if err := policyContext.Destroy(); err != nil {
					logrus.Error(fmt.Errorf("destroying signature policy context: %w", err))
				}
			}()

			options := cp.Options{
				ReportWriter:          os.Stdout,
				SourceCtx:             &systemContext,
				DestinationCtx:        &systemContext,
				MaxParallelDownloads:  maxParallelDownloads,
				ForceManifestMIMEType: manifestFormat,
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
	rootCmd.PersistentFlags().StringVar(&manifestFormat, "format", "", "image manifest type")
	rootCmd.PersistentFlags().BoolVar(&systemContext.DirForceCompress, "dest-compress", false, "force compression of layers for dir: destinations")
	rootCmd.PersistentFlags().BoolVar(&systemContext.DirForceDecompress, "dest-decompress", false, "force decompression of layers for dir: destinations")
	rootCmd.PersistentFlags().StringVar(&compressionFormat, "dest-compress-format", "", "compression type")
	rootCmd.PersistentFlags().IntVar(&compressionLevel, "dest-compress-level", 0, "compression level")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
