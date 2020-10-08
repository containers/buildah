// +build !linux,!darwin

package buildah

import (
	"github.com/containers/buildah/define"
	"github.com/pkg/errors"
)

// ContainerDevices is currently not implemented.
type ContainerDevices = []struct{}

func setChildProcess() error {
	return errors.New("function not supported on non-linux systems")
}

func runUsingRuntimeMain() {}

func (b *Builder) Run(command []string, options RunOptions) error {
	return errors.New("function not supported on non-linux systems")
}
func DefaultNamespaceOptions() (NamespaceOptions, error) {
	return define.NamespaceOptions{}, errors.New("function not supported on non-linux systems")
}
