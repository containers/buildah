package main

import (
	"fmt"

	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/image/pkg/docker/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type logoutReply struct {
	authfile string
	all      bool
}

func init() {
	var (
		opts              logoutReply
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

	flags := logoutCommand.Flags()
	flags.SetInterspersed(false)
	flags.StringVar(&opts.authfile, "authfile", buildahcli.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.BoolVarP(&opts.all, "all", "a", false, "Remove the cached credentials for all registries in the auth file")
	rootCmd.AddCommand(logoutCommand)
}

func logoutCmd(c *cobra.Command, args []string, iopts *logoutReply) error {
	if len(args) > 1 {
		return errors.Errorf("too many arguments, logout takes at most 1 argument")
	}
	if len(args) == 0 && !iopts.all {
		return errors.Errorf("registry must be given")
	}
	var server string
	if len(args) == 1 {
		server = parse.ScrubServer(args[0])
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	if iopts.all {
		if err := config.RemoveAllAuthentication(systemContext); err != nil {
			return err
		}
		fmt.Println("Removed login credentials for all registries")
		return nil
	}

	err = config.RemoveAuthentication(systemContext, server)
	switch err {
	case nil:
		fmt.Printf("Removed login credentials for %s\n", server)
		return nil
	case config.ErrNotLoggedIn:
		return errors.Errorf("Not logged into %s\n", server)
	default:
		return errors.Wrapf(err, "error logging out of %q", server)
	}
}
