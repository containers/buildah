package umask

import (
	"go.podman.io/common/pkg/umask"
)

func CheckUmask() {
	umask.Check()
}

func SetUmask(value int) int {
	return umask.Set(value)
}
