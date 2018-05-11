package types

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

// TestReport is an internal type used for testing.
type TestReport struct {
	Spec *specs.Spec
}
