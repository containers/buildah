package cli

import (
	"testing"

	"github.com/containers/buildah/define"
	"github.com/stretchr/testify/assert"
)

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
