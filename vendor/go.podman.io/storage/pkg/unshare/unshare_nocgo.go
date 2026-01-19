//go:build linux && !cgo

package unshare

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
)

func unshareWithNoCGO(c *Cmd) {
	if c.Cmd.SysProcAttr == nil {
		c.Cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	attr := c.Cmd.SysProcAttr
	if c.UnshareFlags&syscall.CLONE_NEWUSER != 0 {
		attr.Cloneflags = uintptr(c.UnshareFlags)
		attr.GidMappingsEnableSetgroups = c.GidMappingsEnableSetgroups
		if c.Ctty != nil {
			index := len(c.Cmd.ExtraFiles)
			c.Cmd.ExtraFiles = append(c.Cmd.ExtraFiles, c.Ctty)
			attr.Ctty = index
		}
	}
}

func parseIntContainersUnshareEnv(name string) int {
	env := os.Getenv(name)
	if env == "" {
		return -1
	}
	_ = os.Unsetenv(name)
	v, err := strconv.Atoi(env)
	if err != nil {
		return -1
	}
	return v
}

func init() {
	flags := parseIntContainersUnshareEnv("_Containers-unshare")
	if flags == -1 {
		return
	}

	if pidFD := parseIntContainersUnshareEnv("_Containers-pid-pipe"); pidFD > -1 {
		pidFile := os.NewFile(uintptr(pidFD), "")
		_, err := fmt.Fprintf(pidFile, "%d", os.Getpid())
		if err != nil {
			bailOnError(err, "Write pid failed")
		}
		_ = pidFile.Close()
	}
	if continueFD := parseIntContainersUnshareEnv("_Containers-continue-pipe"); continueFD > -1 {
		continueFile := os.NewFile(uintptr(continueFD), "")
		buf := make([]byte, 2048)
		n, err := continueFile.Read(buf)
		if err != nil && err != io.EOF {
			bailOnError(err, "Read containers continue pipe")
		}
		if n > 0 {
			bailOnError(fmt.Errorf(string(buf)), "Unexpected containers continue pipe read")
		}
	}

	if setSid := parseIntContainersUnshareEnv("_Containers-setsid"); setSid == 1 {
		_, err := syscall.Setsid()
		if err != nil {
			bailOnError(err, "Error during setsid")
		}
	}

	if flags&syscall.CLONE_NEWUSER != 0 {
		if err := syscall.Setresuid(0, 0, 0); err != nil {
			bailOnError(err, "Setresuid failed")
		}
		if err := syscall.Setresgid(0, 0, 0); err != nil {
			bailOnError(err, "Setresgid failed")
		}
	}

	// Re-invoke the execve system call to obtain capabilities.
	err := syscall.Exec("/proc/self/exe", os.Args, os.Environ())
	if err != nil {
		bailOnError(err, "syscall.Exec %s", "/proc/self/exe")
	}
}
