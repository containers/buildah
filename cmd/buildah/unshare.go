// +build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"syscall"

	"github.com/containers/buildah/unshare"
	"github.com/containers/buildah/util"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/syndtr/gocapability/capability"
	"github.com/urfave/cli"
)

const (
	// startedInUserNS is an environment variable that, if set, means that we shouldn't try
	// to create and enter a new user namespace and then re-exec ourselves.
	startedInUserNS = "_BUILDAH_STARTED_IN_USERNS"
)

var (
	unshareDescription = "Runs a command in a modified user namespace"
	unshareCommand     = cli.Command{
		Name:           "unshare",
		Usage:          "Run a command in a modified user namespace",
		Description:    unshareDescription,
		Action:         unshareCmd,
		ArgsUsage:      "[COMMAND [ARGS [...]]]",
		SkipArgReorder: true,
	}
)

type runnable interface {
	Run() error
}

func bailOnError(err error, format string, a ...interface{}) {
	if err != nil {
		if format != "" {
			logrus.Errorf("%s: %v", fmt.Sprintf(format, a...), err)
		} else {
			logrus.Errorf("%v", err)
		}
		cli.OsExiter(1)
	}
}

func maybeReexecUsingUserNamespace(c *cli.Context, evenForRoot bool) {
	// If we've already been through this once, no need to try again.
	if os.Getenv(startedInUserNS) != "" {
		return
	}

	// If this is one of the commands that doesn't need this indirection, skip it.
	if c.NArg() == 0 {
		return
	}
	switch c.Args()[0] {
	case "help", "version":
		return
	}

	var uidNum, gidNum uint64
	// Figure out who we are.
	me, err := user.Current()
	if !os.IsNotExist(err) {
		bailOnError(err, "error determining current user")
		uidNum, err = strconv.ParseUint(me.Uid, 10, 32)
		bailOnError(err, "error parsing current UID %s", me.Uid)
		gidNum, err = strconv.ParseUint(me.Gid, 10, 32)
		bailOnError(err, "error parsing current GID %s", me.Gid)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// ID mappings to use to reexec ourselves.
	var uidmap, gidmap []specs.LinuxIDMapping
	if uidNum != 0 || evenForRoot {
		// Read the set of ID mappings that we're allowed to use.  Each
		// range in /etc/subuid and /etc/subgid file is a starting host
		// ID and a range size.
		uidmap, gidmap, err = util.GetSubIDMappings(me.Username, me.Username)
		bailOnError(err, "error reading allowed ID mappings")
		if len(uidmap) == 0 {
			logrus.Warnf("Found no UID ranges set aside for user %q in /etc/subuid.", me.Username)
		}
		if len(gidmap) == 0 {
			logrus.Warnf("Found no GID ranges set aside for user %q in /etc/subgid.", me.Username)
		}
		// Map our UID and GID, then the subuid and subgid ranges,
		// consecutively, starting at 0, to get the mappings to use for
		// a copy of ourselves.
		uidmap = append([]specs.LinuxIDMapping{{HostID: uint32(uidNum), ContainerID: 0, Size: 1}}, uidmap...)
		gidmap = append([]specs.LinuxIDMapping{{HostID: uint32(gidNum), ContainerID: 0, Size: 1}}, gidmap...)
		var rangeStart uint32
		for i := range uidmap {
			uidmap[i].ContainerID = rangeStart
			rangeStart += uidmap[i].Size
		}
		rangeStart = 0
		for i := range gidmap {
			gidmap[i].ContainerID = rangeStart
			rangeStart += gidmap[i].Size
		}
	} else {
		// If we have CAP_SYS_ADMIN, then we don't need to create a new namespace in order to be able
		// to use unshare(), so don't bother creating a new user namespace at this point.
		capabilities, err := capability.NewPid(0)
		bailOnError(err, "error reading the current capabilities sets")
		if capabilities.Get(capability.EFFECTIVE, capability.CAP_SYS_ADMIN) {
			return
		}
		// Read the set of ID mappings that we're currently using.
		uidmap, gidmap, err = util.GetHostIDMappings("")
		bailOnError(err, "error reading current ID mappings")
		// Just reuse them.
		for i := range uidmap {
			uidmap[i].HostID = uidmap[i].ContainerID
		}
		for i := range gidmap {
			gidmap[i].HostID = gidmap[i].ContainerID
		}
	}

	// Unlike most uses of reexec or unshare, we're using a name that
	// _won't_ be recognized as a registered reexec handler, since we
	// _want_ to fall through reexec.Init() to the normal main().
	cmd := unshare.Command(append([]string{"buildah-in-a-user-namespace"}, os.Args[1:]...)...)

	// If, somehow, we don't become UID 0 in our child, indicate that the child shouldn't try again.
	err = os.Setenv(startedInUserNS, "1")
	bailOnError(err, "error setting %s=1 in environment", startedInUserNS)

	// Set the default isolation type to use the "rootless" method.
	if _, present := os.LookupEnv("BUILDAH_ISOLATION"); !present {
		if err = os.Setenv("BUILDAH_ISOLATION", "rootless"); err != nil {
			logrus.Errorf("error setting BUILDAH_ISOLATION=rootless in environment: %v", err)
			os.Exit(1)
		}
	}

	// Reuse our stdio.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up a new user namespace with the ID mapping.
	cmd.UnshareFlags = syscall.CLONE_NEWUSER | syscall.CLONE_NEWNS
	cmd.UseNewuidmap = uidNum != 0
	cmd.UidMappings = uidmap
	cmd.UseNewgidmap = uidNum != 0
	cmd.GidMappings = gidmap
	cmd.GidMappingsEnableSetgroups = true

	// Finish up.
	logrus.Debugf("running %+v with environment %+v, UID map %+v, and GID map %+v", cmd.Cmd.Args, os.Environ(), cmd.UidMappings, cmd.GidMappings)
	execRunnable(cmd)
}

// execRunnable runs the specified unshare command, captures its exit status,
// and exits with the same status.
func execRunnable(cmd runnable) {
	if err := cmd.Run(); err != nil {
		if exitError, ok := errors.Cause(err).(*exec.ExitError); ok {
			if exitError.ProcessState.Exited() {
				if waitStatus, ok := exitError.ProcessState.Sys().(syscall.WaitStatus); ok {
					if waitStatus.Exited() {
						logrus.Errorf("%v", exitError)
						os.Exit(waitStatus.ExitStatus())
					}
					if waitStatus.Signaled() {
						logrus.Errorf("%v", exitError)
						os.Exit(int(waitStatus.Signal()) + 128)
					}
				}
			}
		}
		logrus.Errorf("%v", err)
		logrus.Errorf("(unable to determine exit status)")
		os.Exit(1)
	}
	os.Exit(0)
}

// unshareCmd execs whatever using the ID mappings that we want to use for ourselves
func unshareCmd(c *cli.Context) error {
	// force reexec using the configured ID mappings
	maybeReexecUsingUserNamespace(c, true)
	// exec the specified command, if there is one
	args := c.Args()
	if len(args) < 1 {
		// try to exec the shell, if one's set
		shell, shellSet := os.LookupEnv("SHELL")
		if !shellSet {
			logrus.Errorf("no command specified")
			os.Exit(1)
		}
		args = []string{shell}
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "USER=root", "USERNAME=root", "GROUP=root", "LOGNAME=root", "UID=0", "GID=0")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	execRunnable(cmd)
	os.Exit(1)
	return nil
}
