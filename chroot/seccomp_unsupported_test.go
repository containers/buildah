//go:build (!linux && !freebsd) || !seccomp

package chroot

import (
	"github.com/opencontainers/runtime-spec/specs-go"
)

const seccompAvailable = false

func setupSeccomp(spec *specs.Spec, _ string) error {
	if spec.Linux != nil {
		// runtime-tools may have supplied us with a default filter
		spec.Linux.Seccomp = nil
	}
	return nil
}
