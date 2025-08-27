//go:build linux

package chroot

import (
	"fmt"

	"github.com/opencontainers/runtime-spec/specs-go"
	selinux "github.com/opencontainers/selinux/go-selinux"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/sirupsen/logrus"
)

// setSelinuxLabel sets the process label for child processes that we'll start.
func setSelinuxLabel(spec *specs.Spec) error {
	logrus.Debugf("setting selinux label")
	if spec.Process.SelinuxLabel != "" && selinux.GetEnabled() {
		if err := selinux.SetExecLabel(spec.Process.SelinuxLabel); err != nil {
			return fmt.Errorf("setting process label to %q: %w", spec.Process.SelinuxLabel, err)
		}
	}
	return nil
}

// formatMountLabel adds a mount label mount flag to the mount options list
func formatMountLabel(spec *specs.Spec, mountOptions string) string {
	return label.FormatMountLabel(mountOptions, spec.Linux.MountLabel)
}
