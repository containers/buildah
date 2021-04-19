// +build !go1.13

package dockerclient

import (
	"io"
)

func errorIsEOF(err error) bool {
	return err == io.EOF
}
