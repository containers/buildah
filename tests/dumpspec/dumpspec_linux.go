package main

import (
	"os"
	"slices"
	"syscall"

	"github.com/containers/storage/pkg/unshare"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

func getStarter(containerDir, consoleSocket, pidFile string, spec rspec.Spec, extraFile *os.File) interface{ Start() error } {
	cmd := unshare.Command(subprocName, containerDir, consoleSocket, pidFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if spec.Linux != nil {
		for _, ns := range spec.Linux.Namespaces {
			switch ns.Type {
			case rspec.UserNamespace:
				cmd.UnshareFlags |= syscall.CLONE_NEWUSER
			case rspec.NetworkNamespace: // caller is expecting to configure networking for this process's network namespace
				cmd.UnshareFlags |= syscall.CLONE_NEWNET
			case rspec.MountNamespace:
				cmd.UnshareFlags |= syscall.CLONE_NEWNS
			case rspec.IPCNamespace:
				cmd.UnshareFlags |= syscall.CLONE_NEWIPC
			case rspec.UTSNamespace:
				cmd.UnshareFlags |= syscall.CLONE_NEWUTS
			case rspec.CgroupNamespace:
				cmd.UnshareFlags |= syscall.CLONE_NEWCGROUP
			}
		}
		cmd.UidMappings = slices.Clone(spec.Linux.UIDMappings)
		cmd.GidMappings = slices.Clone(spec.Linux.GIDMappings)
	}
	if extraFile != nil {
		cmd.ExtraFiles = append([]*os.File{extraFile}, cmd.ExtraFiles...)
	}
	return cmd
}
