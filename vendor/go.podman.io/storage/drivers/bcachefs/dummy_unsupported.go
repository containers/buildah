//go:build !linux

package bcachefs

import graphdriver "go.podman.io/storage/drivers"

// Init returns an error on non-Linux platforms as bcachefs is Linux-only.
func Init(home string, options graphdriver.Options) (graphdriver.Driver, error) {
	return nil, graphdriver.ErrNotSupported
}
