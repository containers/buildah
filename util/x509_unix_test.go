//go:build !windows

package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveRootCACertFileWithNoEnvSetUnix(t *testing.T) {
	// calling t.SetEnv has the side-effect of restoring it to the previous
	// value after the test (or leaving it unset).
	// We call this before unsetting it so that we benefit from that cleanup
	t.Setenv("SSL_CERT_FILE", "bogusval")

	// now unset it (t.Unsetenv doesn't exist or we'd use that.)
	assert.Nil(t, os.Unsetenv("SSL_CERT_FILE"))
	path, err := ResolveRootCACertFile()
	assert.Nil(t, err)

	// if our test env doesn't have any of the default locations set,
	// then we need to fix our lists
	assert.NotEmpty(t, path)
}
