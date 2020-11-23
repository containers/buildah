package umask

import (
	"github.com/containers/common/pkg/umask"
)

func CheckUmask() {
	umask.Check()
}

func SetUmask(value int) int {
	return umask.Set(value)
}
