// +build !linux

package main

import (
	"github.com/urfave/cli"
)

var (
	unshareCommand = cli.Command{
		Name:           "unshare",
		Hidden:         true,
		Action:         func(c *cli.Context) error { return nil },
		SkipArgReorder: true,
	}
)

func maybeReexecUsingUserNamespace(c *cli.Context, evenForRoot bool) {
	return
}
