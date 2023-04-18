package util

import (
	"testing"

	"github.com/containers/buildah/define"
	"github.com/stretchr/testify/assert"
)

func TestDecryptConfig(t *testing.T) {
	// Just a smoke test for the default path.
	res, err := DecryptConfig(nil)
	assert.NoError(t, err)
	assert.Nil(t, res)
}

func TestEncryptConfig(t *testing.T) {
	// Just a smoke test for the default path.
	cfg, layers, err := EncryptConfig(nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
	assert.Nil(t, layers)
}
func TestGetFormat(t *testing.T) {
	_, err := GetFormat("bogus")
	assert.NotNil(t, err)

	format, err := GetFormat("oci")
	assert.Nil(t, err)
	assert.Equalf(t, define.OCIv1ImageManifest, format, "expected oci format but got %v.", format)
	format, err = GetFormat("docker")
	assert.Nil(t, err)
	assert.Equalf(t, define.Dockerv2ImageManifest, format, "expected docker format but got %v.", format)
}
