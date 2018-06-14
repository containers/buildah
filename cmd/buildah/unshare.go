// +build linux

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/unshare"
	"github.com/projectatomic/buildah/util"
	"github.com/sirupsen/logrus"
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

func maybeReexecUsingUserNamespace(c *cli.Context, evenForRoot bool) {
	// If we've already been through this once, no need to try again.
	if os.Getenv(startedInUserNS) != "" {
		return
	}

	// Figure out if we're already root, or "root", which is close enough,
	// unless we've been explicitly told to do this even for root.
	me, err := user.Current()
	if err != nil {
		logrus.Errorf("error determining current user: %v", err)
		cli.OsExiter(1)
	}
	if me.Uid == "0" && !evenForRoot {
		return
	}
	uidNum, err := strconv.ParseUint(me.Uid, 10, 32)
	if err != nil {
		logrus.Errorf("error parsing current UID %s: %v", me.Uid, err)
		cli.OsExiter(1)
	}
	gidNum, err := strconv.ParseUint(me.Gid, 10, 32)
	if err != nil {
		logrus.Errorf("error parsing current GID %s: %v", me.Gid, err)
		cli.OsExiter(1)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Read the set of ID mappings that we're allowed to use.  Each range
	// in /etc/subuid and /etc/subgid file is a starting ID and a range size.
	uidmap, gidmap, err := util.GetSubIDMappings(me.Username, me.Username)
	if err != nil {
		logrus.Errorf("error reading allowed ID mappings: %v", err)
		cli.OsExiter(1)
	}
	if len(uidmap) == 0 {
		logrus.Warnf("Found no UID ranges set aside for user %q in /etc/subuid.", me.Username)
	}
	if len(gidmap) == 0 {
		logrus.Warnf("Found no GID ranges set aside for user %q in /etc/subgid.", me.Username)
	}

	// Build modified maps that map us to uid/gid 0, and maps every other
	// range to itself.  In a namespace that uses this map, the invoking
	// user will appear to be root.  This should let us create storage
	// directories and access credentials under the invoking user's home
	// directory.
	uidmap2 := append([]specs.LinuxIDMapping{{HostID: uint32(uidNum), ContainerID: 0, Size: 1}}, uidmap...)
	for i := range uidmap2[1:] {
		uidmap2[i+1].ContainerID = uidmap2[i+1].HostID
	}
	gidmap2 := append([]specs.LinuxIDMapping{{HostID: uint32(gidNum), ContainerID: 0, Size: 1}}, gidmap...)
	for i := range gidmap2[1:] {
		gidmap2[i+1].ContainerID = gidmap2[i+1].HostID
	}

	// Map the uidmap and gidmap ranges, consecutively, starting at 0.
	// When used to created a namespace inside of a namespace that uses the
	// maps we've created above, they'll produce mappings which don't map
	// in the invoking user.  This is more suitable for running commands in
	// containers, so we'll want to use it as a default for any containers
	// that we create.
	umap := new(bytes.Buffer)
	for i := range uidmap {
		if i > 0 {
			fmt.Fprintf(umap, ",")
		}
		fmt.Fprintf(umap, "%d:%d:%d", uidmap[i].ContainerID, uidmap[i].HostID, uidmap[i].Size)
	}
	gmap := new(bytes.Buffer)
	for i := range gidmap {
		if i > 0 {
			fmt.Fprintf(gmap, ",")
		}
		fmt.Fprintf(gmap, "%d:%d:%d", gidmap[i].ContainerID, gidmap[i].HostID, gidmap[i].Size)
	}

	// Add args to change the global defaults.
	defaultStorageDriver := "vfs"
	defaultRoot, err := util.UnsharedRootPath(me.HomeDir)
	if err != nil {
		logrus.Errorf("%v", err)
		cli.OsExiter(1)
	}
	defaultRunroot, err := util.UnsharedRunrootPath(me.Uid)
	if err != nil {
		logrus.Errorf("%v", err)
		cli.OsExiter(1)
	}
	var moreArgs []string
	if !c.GlobalIsSet("storage-driver") || !c.GlobalIsSet("root") || !c.GlobalIsSet("runroot") || (!c.GlobalIsSet("userns-uid-map") && !c.GlobalIsSet("userns-gid-map")) {
		logrus.Infof("Running without privileges, assuming arguments:")
		if !c.GlobalIsSet("storage-driver") {
			logrus.Infof(" --storage-driver %q", defaultStorageDriver)
			moreArgs = append(moreArgs, "--storage-driver", defaultStorageDriver)
		}
		if !c.GlobalIsSet("root") {
			logrus.Infof(" --root %q", defaultRoot)
			moreArgs = append(moreArgs, "--root", defaultRoot)
		}
		if !c.GlobalIsSet("runroot") {
			logrus.Infof(" --runroot %q", defaultRunroot)
			moreArgs = append(moreArgs, "--runroot", defaultRunroot)
		}
		if !c.GlobalIsSet("userns-uid-map") && !c.GlobalIsSet("userns-gid-map") && umap.Len() > 0 && gmap.Len() > 0 {
			logrus.Infof(" --userns-uid-map %q --userns-gid-map %q", umap.String(), gmap.String())
			moreArgs = append(moreArgs, "--userns-uid-map", umap.String(), "--userns-gid-map", gmap.String())
		}
	}

	// Unlike most uses of reexec or unshare, we're using a name that
	// _won't_ be recognized as a registered reexec handler, since we
	// _want_ to fall through reexec.Init() to the normal main().
	cmd := unshare.Command(append(append([]string{"buildah-unprivileged"}, moreArgs...), os.Args[1:]...)...)

	// If, somehow, we don't become UID 0 in our child, indicate that the child shouldn't try again.
	if err = os.Setenv(startedInUserNS, "1"); err != nil {
		logrus.Errorf("error setting %s=1 in environment: %v", startedInUserNS, err)
		os.Exit(1)
	}

	// Reuse our stdio.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up a new user namespace with the ID mapping.
	cmd.UnshareFlags = syscall.CLONE_NEWUSER
	cmd.UseNewuidmap = true
	cmd.UidMappings = uidmap2
	cmd.UseNewgidmap = true
	cmd.GidMappings = gidmap2
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
