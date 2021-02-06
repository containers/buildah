package define

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	// Package is the name of this package, used in help output and to
	// identify working containers.
	Package = "buildah"
	// Version for the Package.  Bump version in contrib/rpm/buildah.spec
	// too.
	Version = "1.20.0-dev"
)

// IDMappingOptions controls how we set up UID/GID mapping when we set up a
// user namespace.
type IDMappingOptions struct {
	HostUIDMapping bool
	HostGIDMapping bool
	UIDMap         []specs.LinuxIDMapping
	GIDMap         []specs.LinuxIDMapping
}
