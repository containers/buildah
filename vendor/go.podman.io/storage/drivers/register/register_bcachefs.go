//go:build !exclude_graphdriver_bcachefs && linux

package register

import (
	_ "go.podman.io/storage/drivers/bcachefs"
)
