package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/spf13/cobra"
)

func init() {
	var (
		opts = auth.LogoutOptions{
			Stdout:             os.Stdout,
			AcceptRepositories: true,
		}
		logoutDescription = "Remove the cached username and password for the registry."
	)
	logoutCommand := &cobra.Command{
		Use:   "logout",
		Short: "Logout of a container registry",
		Long:  logoutDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return logoutCmd(cmd, args, &opts)
		},
		Example: `buildah logout quay.io`,
	}
	logoutCommand.SetUsageTemplate(UsageTemplate())

	flags := auth.GetLogoutFlags(&opts)
	flags.SetInterspersed(false)
	logoutCommand.Flags().AddFlagSet(flags)
	rootCmd.AddCommand(logoutCommand)
}

func logoutCmd(c *cobra.Command, args []string, iopts *auth.LogoutOptions) error {
	if len(args) > 1 {
		return errors.New("too many arguments, logout takes at most 1 argument")
	}
	if len(args) == 0 && !iopts.All {
		return errors.New("registry must be given")
	}

	if err := setXDGRuntimeDir(); err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return fmt.Errorf("building system context: %w", err)
	}
	// parse.SystemContextFromOptions may point this field to an auth.json or to a .docker/config.json;
	// that’s fair enough for reads, but incorrect for writes (the two files have incompatible formats),
	// and it interferes with the auth.Logout’s own argument parsing.
	systemContext.AuthFilePath = ""
	return auth.Logout(systemContext, iopts, args)
}
