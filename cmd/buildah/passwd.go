package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

var (
	passwdDescription = `Generate a password hash using golang.org/x/crypto/bcrypt.`
	passwdCommand     = &cobra.Command{
		Use:     "passwd",
		Short:   "Generate a password hash",
		Long:    passwdDescription,
		RunE:    passwdCmd,
		Example: `buildah passwd testpassword`,
		Args:    cobra.ExactArgs(1),
		Hidden:  true,
	}
)

func passwdCmd(c *cobra.Command, args []string) error {
	passwd, err := bcrypt.GenerateFromPassword([]byte(args[0]), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	fmt.Println(string(passwd))
	return nil
}

func init() {
	rootCmd.AddCommand(passwdCommand)
}
