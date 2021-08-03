package main

import (
	"os"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/pkg/errors"
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
	flags.SetInterspersed(false)
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")
	flags.BoolVar(&opts.getLogin, "get-login", true, "Return the current login user for the registry")
	flags.AddFlagSet(auth.GetLoginFlags(&opts.loginOpts))
	rootCmd.AddCommand(loginCommand)
}

func loginCmd(c *cobra.Command, args []string, iopts *loginReply) error {
	if len(args) > 1 {
		return errors.Errorf("too many arguments, login takes only 1 argument")
	}
	if len(args) == 0 {
		return errors.Errorf("please specify a registry to login to")
	}

	if err := setXDGRuntimeDir(); err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}
	ctx := getContext()
	iopts.loginOpts.GetLoginSet = c.Flag("get-login").Changed
	return auth.Login(ctx, systemContext, &iopts.loginOpts, args)
}
