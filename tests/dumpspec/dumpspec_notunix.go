//go:build windows

package main

import (
	"errors"
	"os"
)

func sendConsoleDescriptor(consoleSocket string) (*os.File, error) {
	return nil, errors.New("unable to transport pseudoterminal descriptors")
}
