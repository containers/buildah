package internalutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetHas(t *testing.T) {
	m := map[string]string{
		"key1": "ignored",
	}
	assert.True(t, SetHas(m, "key1"))
	assert.False(t, SetHas(m, "key2"))
}
