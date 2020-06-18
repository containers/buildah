// +build go1.13

package dockerclient

import (
	"fmt"
	"os"
)

func makeNotExistError(s string) error {
	return fmt.Errorf("%w: %s", os.ErrNotExist, s)
}
