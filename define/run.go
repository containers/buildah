package define

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// Isolation provides a way to specify whether we're supposed to use a proper
// OCI runtime, or some other method for running commands.
type Isolation int

// String converts a Isolation into a string.
func (i Isolation) String() string {
	switch i {
	case IsolationDefault:
		return "IsolationDefault"
	case IsolationOCI:
		return "IsolationOCI"
	case IsolationChroot:
		return "IsolationChroot"
	case IsolationOCIRootless:
		return "IsolationOCIRootless"
	}
	return fmt.Sprintf("unrecognized isolation type %d", i)
}

const (
	// IsolationDefault is whatever we think will work best.
	IsolationDefault Isolation = iota
	// IsolationOCI is a proper OCI runtime.
	IsolationOCI
	// IsolationChroot is a more chroot-like environment: less isolation,
	// but with fewer requirements.
	IsolationChroot
	// IsolationOCIRootless is a proper OCI runtime in rootless mode.
	IsolationOCIRootless
)

// NamespaceOption controls how we set up a namespace when launching processes.
type NamespaceOption struct {
	// Name specifies the type of namespace, typically matching one of the
	// ...Namespace constants defined in
	// github.com/opencontainers/runtime-spec/specs-go.
	Name string
	// Host is used to force our processes to use the host's namespace of
	// this type.
	Host bool
	// Path is the path of the namespace to attach our process to, if Host
	// is not set.  If Host is not set and Path is also empty, a new
	// namespace will be created for the process that we're starting.
	// If Name is specs.NetworkNamespace, if Path doesn't look like an
	// absolute path, it is treated as a comma-separated list of CNI
	// configuration names which will be selected from among all of the CNI
	// network configurations which we find.
	Path string
}

// NamespaceOptions provides some helper methods for a slice of NamespaceOption
// structs.
type NamespaceOptions []NamespaceOption

// Find the configuration for the namespace of the given type.  If there are
// duplicates, find the _last_ one of the type, since we assume it was appended
// more recently.
func (n *NamespaceOptions) Find(namespace string) *NamespaceOption {
	for i := range *n {
		j := len(*n) - 1 - i
		if (*n)[j].Name == namespace {
			return &((*n)[j])
		}
	}
	return nil
}

// AddOrReplace either adds or replaces the configuration for a given namespace.
func (n *NamespaceOptions) AddOrReplace(options ...NamespaceOption) {
nextOption:
	for _, option := range options {
		for i := range *n {
			j := len(*n) - 1 - i
			if (*n)[j].Name == option.Name {
				(*n)[j] = option
				continue nextOption
			}
		}
		*n = append(*n, option)
	}
}

// IDMappingOptions controls how we set up UID/GID mapping when we set up a
// user namespace.
type IDMappingOptions struct {
	HostUIDMapping bool
	HostGIDMapping bool
	UIDMap         []specs.LinuxIDMapping
	GIDMap         []specs.LinuxIDMapping
}
