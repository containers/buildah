// +build !linux

package main

import (
	"github.com/spf13/cobra"
)

const (
	startedInUserNS = "_BUILDAH_STARTED_IN_USERNS"
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

func maybeReexecUsingUserNamespace(cmd string, evenForRoot bool) {
	return
}
