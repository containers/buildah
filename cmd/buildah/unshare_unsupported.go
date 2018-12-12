// +build !linux

package main

import (
	"github.com/urfave/cli"
)

const (
	startedInUserNS = "_BUILDAH_STARTED_IN_USERNS"
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
