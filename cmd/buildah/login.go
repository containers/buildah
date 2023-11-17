package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/spf13/cobra"
)

type loginReply struct {
	loginOpts auth.LoginOptions
	getLogin  bool
	tlsVerify bool
}

func init() {
	var (
		opts = loginReply{
			loginOpts: auth.LoginOptions{
				Stdin:              os.Stdin,
				Stdout:             os.Stdout,
				AcceptRepositories: true,
			},
		}
		loginDescription = "Login to a container registry on a specified server."
	)
	loginCommand := &cobra.Command{
		Use:   "login",
		Short: "Login to a container registry",
		Long:  loginDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return loginCmd(cmd, args, &opts)
		},
		Example: `buildah login quay.io`,
	}
	loginCommand.SetUsageTemplate(UsageTemplate())

	flags := loginCommand.Flags()
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	flags.BoolVar(&opts.getLogin, "get-login", true, "return the current login user for the registry")
	flags.AddFlagSet(auth.GetLoginFlags(&opts.loginOpts))
	opts.loginOpts.Stdin = os.Stdin
	opts.loginOpts.Stdout = os.Stdout
	rootCmd.AddCommand(loginCommand)
}

func loginCmd(c *cobra.Command, args []string, iopts *loginReply) error {
	if len(args) > 1 {
		return errors.New("too many arguments, login takes only 1 argument")
	}
	if len(args) == 0 {
		return errors.New("please specify a registry to login to")
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
	// and it interferes with the auth.Login’s own argument parsing.
	systemContext.AuthFilePath = ""
	ctx := getContext()
	iopts.loginOpts.GetLoginSet = c.Flag("get-login").Changed
	return auth.Login(ctx, systemContext, &iopts.loginOpts, args)
}
