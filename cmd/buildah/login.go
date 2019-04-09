package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/image/docker"
	"github.com/containers/image/pkg/docker/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

type loginReply struct {
	authfile      string
	certDir       string
	password      string
	username      string
	tlsVerify     bool
	stdinPassword bool
	getLogin      bool
}

func init() {
	var (
		opts             loginReply
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
	flags.StringVar(&opts.authfile, "authfile", "", "path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVarP(&opts.password, "password", "p", "", "Password for registry")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry")
	flags.StringVarP(&opts.username, "username", "u", "", "Username for registry")
	flags.BoolVar(&opts.stdinPassword, "password-stdin", false, "Take the password from stdin")
	flags.BoolVar(&opts.getLogin, "get-login", true, "Return the current login user for the registry")
	rootCmd.AddCommand(loginCommand)
}

func loginCmd(c *cobra.Command, args []string, iopts *loginReply) error {
	if len(args) > 1 {
		return errors.Errorf("too many arguments, login takes only 1 argument")
	}
	if len(args) == 0 {
		return errors.Errorf("please specify a registry to login to")
	}
	server := parse.RegistryFromFullName(parse.ScrubServer(args[0]))
	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}
	if !iopts.getLogin {
		user, err := config.GetUserLoggedIn(systemContext, server)
		if err != nil {
			return errors.Wrapf(err, "unable to check for login user")
		}

		if user == "" {
			return errors.Errorf("not logged into %s", server)
		}
		fmt.Printf("%s\n", user)
		return nil
	}

	// username of user logged in to server (if one exists)
	userFromAuthFile, passFromAuthFile, err := config.GetAuthentication(systemContext, server)
	if err != nil {
		return errors.Wrapf(err, "error reading auth file")
	}

	ctx := getContext()
	password := iopts.password

	if iopts.stdinPassword {
		var stdinPasswordStrBuilder strings.Builder
		if iopts.password != "" {
			return errors.Errorf("Can't specify both --password-stdin and --password")
		}
		if iopts.username == "" {
			return errors.Errorf("Must provide --username with --password-stdin")
		}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Fprint(&stdinPasswordStrBuilder, scanner.Text())
		}
		password = stdinPasswordStrBuilder.String()
	}

	// If no username and no password is specified, try to use existing ones.
	if iopts.username == "" && password == "" && userFromAuthFile != "" && passFromAuthFile != "" {
		fmt.Println("Authenticating with existing credentials...")
		if err := docker.CheckAuth(ctx, systemContext, userFromAuthFile, passFromAuthFile, server); err == nil {
			fmt.Println("Existing credentials are valid. Already logged in to", server)
			return nil
		}
		fmt.Println("Existing credentials are invalid, please enter valid username and password")
	}

	username, password, err := GetUserAndPass(iopts.username, password, userFromAuthFile)
	if err != nil {
		return errors.Wrapf(err, "error getting username and password")
	}

	if err = docker.CheckAuth(ctx, systemContext, username, password, server); err == nil {
		// Write the new credentials to the authfile
		if err = config.SetAuthentication(systemContext, server, username, password); err != nil {
			return err
		}
	}
	switch err {
	case nil:
		fmt.Println("Login Succeeded!")
		return nil
	case docker.ErrUnauthorizedForCredentials:
		return errors.Errorf("error logging into %q: invalid username/password", server)
	default:
		return errors.Wrapf(err, "error authenticating creds for %q", server)
	}
}

// GetUserAndPass gets the username and password from STDIN if not given
// using the -u and -p flags.  If the username prompt is left empty, the
// displayed userFromAuthFile will be used instead.
func GetUserAndPass(username, password, userFromAuthFile string) (string, string, error) {
	var err error
	reader := bufio.NewReader(os.Stdin)
	if username == "" {
		if userFromAuthFile != "" {
			fmt.Printf("Username (%s): ", userFromAuthFile)
		} else {
			fmt.Print("Username: ")
		}
		username, err = reader.ReadString('\n')
		if err != nil {
			return "", "", errors.Wrapf(err, "error reading username")
		}
		// If the user just hit enter, use the displayed user from the
		// the authentication file.  This allows to do a lazy
		// `$ buildah login -p $NEW_PASSWORD` without specifying the
		// user.
		if strings.TrimSpace(username) == "" {
			username = userFromAuthFile
		}
	}
	if password == "" {
		fmt.Print("Password: ")
		pass, err := terminal.ReadPassword(0)
		if err != nil {
			return "", "", errors.Wrapf(err, "error reading password")
		}
		password = string(pass)
		fmt.Println()
	}
	return strings.TrimSpace(username), password, err
}
