//go:build !windows

package main

import (
	"fmt"
	"net"
	"os"

	"github.com/containers/buildah/internal/pty"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func sendConsoleDescriptor(consoleSocket string) (*os.File, error) {
	closePty := true
	control, pty, err := pty.GetPtyDescriptors()
	if err != nil {
		return nil, fmt.Errorf("allocating pseudo-terminal: %w", err)
	}
	defer unix.Close(control)
	defer func() {
		if closePty {
			if err := unix.Close(pty); err != nil {
				logrus.Errorf("closing pty descriptor %d: %v", pty, err)
			}
		}
	}()
	socketReceiver, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: consoleSocket, Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("allocating pseudo-terminal: %w", err)
	}
	defer socketReceiver.Close()
	rights := unix.UnixRights(control)
	_, _, err = socketReceiver.WriteMsgUnix(nil, rights, nil)
	if err != nil {
		return nil, fmt.Errorf("sending terminal control fd to parent process: %w", err)
	}
	closePty = false
	return os.NewFile(uintptr(pty), "controlling terminal"), nil
}
