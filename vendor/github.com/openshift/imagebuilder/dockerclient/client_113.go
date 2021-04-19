// +build go1.13

package dockerclient

import (
	"errors"
	"io"
)

func errorIsEOF(err error) bool {
	return errors.Is(err, io.EOF)
}
