package define

import (
	"fmt"
)

type Isolation int

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
