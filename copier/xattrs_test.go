package copier

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	// exercise the ERANGE-handling logic
	initialXattrListSize = 1
	initialXattrValueSize = 1
}

func TestXattrs(t *testing.T) {
	if !xattrsSupported {
		t.Skipf("xattrs are not supported on this platform, skipping")
	}
	testValues := map[string]string{
		"user.a": "attribute value a",
		"user.b": "attribute value b",
	}
	tmp := t.TempDir()
	for attribute, value := range testValues {
		t.Run(fmt.Sprintf("attribute=%s", attribute), func(t *testing.T) {
			f, err := os.CreateTemp(tmp, "copier-xattr-test-")
			if !assert.Nil(t, err, "error creating test file: %v", err) {
				t.FailNow()
			}
			defer os.Remove(f.Name())

			err = Lsetxattrs(f.Name(), map[string]string{attribute: value})
			if errors.Is(err, syscall.ENOTSUP) {
				t.Skipf("extended attributes not supported on %q, skipping", tmp)
			}
			if !assert.Nil(t, err, "error setting attribute on file: %v", err) {
				t.FailNow()
			}

			xattrs, err := Lgetxattrs(f.Name())
			if !assert.Nil(t, err, "error reading attributes of file: %v", err) {
				t.FailNow()
			}
			xvalue, ok := xattrs[attribute]
			if !assert.True(t, ok, "did not read back attribute %q for file", attribute) {
				t.FailNow()
			}
			if !assert.Equal(t, value, xvalue, "read back different value for attribute %q", attribute) {
				t.FailNow()
			}
		})
	}
}
