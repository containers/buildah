// +build !go1.13

package dockerclient

import (
	"os"
)

func makeNotExistError(s string) error {
	return os.ErrNotExist
}
