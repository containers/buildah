// +build !linux

package main

import (
	"github.com/spf13/cobra"
)

func init() {
	unshareCommand := cobra.Command{
		Use:    "unshare",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	rootCmd.AddCommand(&unshareCommand)
}
