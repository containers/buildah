//go:build go113
// +build go113

package copier

import (
	"github.com/pkg/errors"
)

func unwrapError(err error) error {
	e := errors.Cause(err)
	for e != nil {
		err = e
		e = errors.Unwrap(err)
	}
	return err
}
