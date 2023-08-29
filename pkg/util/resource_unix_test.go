package util

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUlimit(t *testing.T) {
	_, err := ParseUlimit("bogus")
	assert.NotNil(t, err)

	ul, err := ParseUlimit("memlock=100:200")
	assert.Nil(t, err)
	assert.Equal(t, ul.Soft, int64(100))
	assert.Equal(t, ul.Hard, int64(200))

	var limit syscall.Rlimit
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit)
	assert.Nil(t, err)

	ul, err = ParseUlimit("nofile=-1:-1")
	assert.Nil(t, err)
	assert.Equal(t, ul.Soft, int64(limit.Cur))
	assert.Equal(t, ul.Hard, int64(limit.Max))

	ul, err = ParseUlimit("nofile=100:-1")
	assert.Nil(t, err)
	assert.Equal(t, ul.Soft, int64(100))
	assert.Equal(t, ul.Hard, int64(limit.Max))
}
