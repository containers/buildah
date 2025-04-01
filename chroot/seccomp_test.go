//go:build linux && seccomp

package chroot

import (
	"fmt"
	"os"

	"github.com/containers/common/pkg/seccomp"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const seccompAvailable = true

func setupSeccomp(spec *specs.Spec, seccompProfilePath string) error {
	switch seccompProfilePath {
	case "unconfined":
		spec.Linux.Seccomp = nil
	case "":
		seccompConfig, err := seccomp.GetDefaultProfile(spec)
		if err != nil {
			return fmt.Errorf("loading default seccomp profile failed: %w", err)
		}
		spec.Linux.Seccomp = seccompConfig
	default:
		seccompProfile, err := os.ReadFile(seccompProfilePath)
		if err != nil {
			return fmt.Errorf("opening seccomp profile failed: %w", err)
		}
		seccompConfig, err := seccomp.LoadProfile(string(seccompProfile), spec)
		if err != nil {
			return fmt.Errorf("loading seccomp profile (%s) failed: %w", seccompProfilePath, err)
		}
		spec.Linux.Seccomp = seccompConfig
	}
	return nil
}
