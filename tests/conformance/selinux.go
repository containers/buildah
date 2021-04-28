// +build linux

package conformance

import (
	selinux "github.com/opencontainers/selinux/go-selinux"
)

func selinuxMountFlag() string {
	if selinux.GetEnabled() {
		return ":Z"
	}
	return ""
}
