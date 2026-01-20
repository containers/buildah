//go:build !linux

package main

import (
	"os"
	"os/exec"

	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

func getStarter(containerDir, consoleSocket, pidFile string, _ rspec.Spec, extraFile *os.File) interface{ Start() error } {
	cmd := exec.Command(subprocName, containerDir, consoleSocket, pidFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if extraFile != nil {
		cmd.ExtraFiles = append([]*os.File{extraFile}, cmd.ExtraFiles...)
	}
	return cmd
}
