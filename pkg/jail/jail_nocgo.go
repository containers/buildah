//go:build freebsd && !cgo
// +build freebsd,!cgo

package jail

import (
	"errors"
)

type NS int32

const (
	DISABLED NS = 0
	NEW      NS = 1
	INHERIT  NS = 2
)

type config struct {
	params map[string]interface{}
}

func NewConfig() *config {
	return nil
}

func (c *config) Set(key string, value interface{}) {
}

type jail struct {
}

func Create(jconf *config) (*jail, error) {
	return nil, errors.New("not supported in nocgo")
}

func CreateAndAttach(jconf *config) (*jail, error) {
	return nil, errors.New("not supported in nocgo")
}

func FindByName(name string) (*jail, error) {
	return nil, errors.New("not supported in nocgo")
}

func (j *jail) Set(jconf *config) error {
	return errors.New("not supported in nocgo")
}
