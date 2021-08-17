package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoverContainerfile(t *testing.T) {
	_, err := DiscoverContainerfile("./bogus")
	assert.NotNil(t, err)

	_, err = DiscoverContainerfile("./")
	assert.NotNil(t, err)

	name, err := DiscoverContainerfile("test/test1/Dockerfile")
	assert.Nil(t, err)
	assert.Equal(t, name, "test/test1/Dockerfile")

	name, err = DiscoverContainerfile("test/test1/Containerfile")
	assert.Nil(t, err)
	assert.Equal(t, name, "test/test1/Containerfile")

	name, err = DiscoverContainerfile("test/test1")
	assert.Nil(t, err)
	assert.Equal(t, name, "test/test1/Containerfile")

	name, err = DiscoverContainerfile("test/test2")
	assert.Nil(t, err)
	assert.Equal(t, name, "test/test2/Dockerfile")

}
