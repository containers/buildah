package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveRootCACertFileWithEnvSet(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", "/path/to/file")

	path, err := ResolveRootCACertFile()
	assert.Nil(t, err)

	assert.Equal(t, "/path/to/file", path)
}
