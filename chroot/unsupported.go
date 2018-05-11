// +build !linux

package chroot

import (
	"github.com/pkg/errors"
)

// RunUsingChroot is not supported.
func RunUsingChroot(spec *specs.Spec, bundlePath string, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	return errors.Errorf("--isolation chroot is not supported on this platform")
}
