package main

import (
	"context"

	"github.com/containers/buildah/internal/source"
	"github.com/spf13/cobra"
	"go.podman.io/image/v5/pkg/cli/basetls/tlsdetails"
)

type sourcePullOptions struct {
	source.PullOptions
	tlsDetails string
}

type sourcePushOptions struct {
	source.PushOptions
	tlsDetails string
}

var (
	// buildah source
	sourceDescription = `  Create, push, pull and manage source images and associated source artifacts.  A source image contains all source artifacts an ordinary OCI image has been built with.  Those artifacts can be any kind of source artifact, such as source RPMs, an entire source tree or text files.

  Note that the buildah-source command and all its subcommands are experimental and may be subject to future changes.
`
	sourceCommand = &cobra.Command{
		Use:   "source",
		Short: "Manage source containers",
		Long:  sourceDescription,
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	// buildah source create
	sourceCreateDescription = `  Create and initialize a source image.  A source image is an OCI artifact; an OCI image with a custom config media type.

  Note that the buildah-source command and all its subcommands are experimental and may be subject to future changes.
`
	sourceCreateOptions = source.CreateOptions{}
	sourceCreateCommand = &cobra.Command{
		Args:    cobra.ExactArgs(1),
		Use:     "create",
		Short:   "Create a source image",
		Long:    sourceCreateDescription,
		Example: "buildah source create /tmp/fedora:latest-source",
		RunE: func(_ *cobra.Command, args []string) error {
			return source.Create(context.Background(), args[0], sourceCreateOptions)
		},
	}

	// buildah source add
	sourceAddOptions     = source.AddOptions{}
	sourceAddDescription = `  Add add a source artifact to a source image.  The artifact will be added as a gzip-compressed tar ball.  Add attempts to auto-tar and auto-compress only if necessary.

  Note that the buildah-source command and all its subcommands are experimental and may be subject to future changes.
`
	sourceAddCommand = &cobra.Command{
		Args:    cobra.ExactArgs(2),
		Use:     "add",
		Short:   "Add a source artifact to a source image",
		Long:    sourceAddDescription,
		Example: "buildah source add /tmp/fedora sources.tar.gz",
		RunE: func(_ *cobra.Command, args []string) error {
			return source.Add(context.Background(), args[0], args[1], sourceAddOptions)
		},
	}

	// buildah source pull
	sourcePullOpts        = sourcePullOptions{}
	sourcePullDescription = `  Pull a source image from a registry to a specified path.  The pull operation will fail if the image does not comply with a source-image OCI artifact.

  Note that the buildah-source command and all its subcommands are experimental and may be subject to future changes.
`
	sourcePullCommand = &cobra.Command{
		Args:    cobra.ExactArgs(2),
		Use:     "pull",
		Short:   "Pull a source image from a registry to a specified path",
		Long:    sourcePullDescription,
		Example: "buildah source pull quay.io/sourceimage/example:latest /tmp/sourceimage:latest",
		RunE: func(c *cobra.Command, args []string) error {
			return sourcePullCmd(c, args, sourcePullOpts)
		},
	}

	// buildah source push
	sourcePushOpts        = sourcePushOptions{}
	sourcePushDescription = `  Push a source image from a specified path to a registry.

  Note that the buildah-source command and all its subcommands are experimental and may be subject to future changes.
`
	sourcePushCommand = &cobra.Command{
		Args:    cobra.ExactArgs(2),
		Use:     "push",
		Short:   "Push a source image from a specified path to a registry",
		Long:    sourcePushDescription,
		Example: "buildah source push /tmp/sourceimage:latest quay.io/sourceimage/example:latest",
		RunE: func(c *cobra.Command, args []string) error {
			return sourcePushCmd(c, args, sourcePushOpts)
		},
	}
)

func sourcePullCmd(_ *cobra.Command, args []string, opts sourcePullOptions) error {
	baseTLSConfig, err := tlsdetails.BaseTLSFromOptionalFile(opts.tlsDetails)
	if err != nil {
		return err
	}
	opts.PullOptions.BaseTLSConfig = baseTLSConfig.TLSConfig()
	return source.Pull(context.Background(), args[0], args[1], opts.PullOptions)
}

func sourcePushCmd(_ *cobra.Command, args []string, opts sourcePushOptions) error {
	baseTLSConfig, err := tlsdetails.BaseTLSFromOptionalFile(opts.tlsDetails)
	if err != nil {
		return err
	}
	opts.PushOptions.BaseTLSConfig = baseTLSConfig.TLSConfig()
	return source.Push(context.Background(), args[0], args[1], opts.PushOptions)
}

func init() {
	// buildah source
	sourceCommand.SetUsageTemplate(UsageTemplate())
	rootCmd.AddCommand(sourceCommand)

	// buildah source create
	sourceCreateCommand.SetUsageTemplate(UsageTemplate())
	sourceCommand.AddCommand(sourceCreateCommand)
	sourceCreateFlags := sourceCreateCommand.Flags()
	sourceCreateFlags.StringVar(&sourceCreateOptions.Author, "author", "", "set the author")
	sourceCreateFlags.BoolVar(&sourceCreateOptions.TimeStamp, "time-stamp", true, "set the \"created\" time stamp")

	// buildah source add
	sourceAddCommand.SetUsageTemplate(UsageTemplate())
	sourceCommand.AddCommand(sourceAddCommand)
	sourceAddFlags := sourceAddCommand.Flags()
	sourceAddFlags.StringArrayVar(&sourceAddOptions.Annotations, "annotation", []string{}, "add an annotation (format: key=value)")

	// buildah source pull
	sourcePullCommand.SetUsageTemplate(UsageTemplate())
	sourceCommand.AddCommand(sourcePullCommand)
	sourcePullFlags := sourcePullCommand.Flags()
	sourcePullFlags.StringVar(&sourcePullOpts.PullOptions.Credentials, "creds", "", "use `[username[:password]]` for accessing the registry")
	sourcePullFlags.StringVar(&sourcePullOpts.tlsDetails, "tls-details", "", "path to a containers-tls-details.yaml file")
	sourcePullFlags.BoolVar(&sourcePullOpts.PullOptions.TLSVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry")
	sourcePullFlags.BoolVarP(&sourcePullOpts.PullOptions.Quiet, "quiet", "q", false, "don't output pull progress information")

	// buildah source push
	sourcePushCommand.SetUsageTemplate(UsageTemplate())
	sourceCommand.AddCommand(sourcePushCommand)
	sourcePushFlags := sourcePushCommand.Flags()
	sourcePushFlags.StringVar(&sourcePushOpts.PushOptions.Credentials, "creds", "", "use `[username[:password]]` for accessing the registry")
	sourcePushFlags.StringVar(&sourcePushOpts.PushOptions.DigestFile, "digestfile", "", "after copying the artifact, write the digest of the resulting image to the file")
	sourcePushFlags.StringVar(&sourcePushOpts.tlsDetails, "tls-details", "", "path to a containers-tls-details.yaml file")
	sourcePushFlags.BoolVar(&sourcePushOpts.PushOptions.TLSVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry")
	sourcePushFlags.BoolVarP(&sourcePushOpts.PushOptions.Quiet, "quiet", "q", false, "don't output push progress information")
}
