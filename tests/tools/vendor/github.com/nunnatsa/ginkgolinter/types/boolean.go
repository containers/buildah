package types

import (
	"errors"
	"strings"
)

// Boolean is a bool, implementing the flag.Value interface, to be used as a flag var.
type Boolean bool

func (b *Boolean) Set(value string) error {
	if b == nil {
		return errors.New("trying to set nil parameter")
	}
	switch strings.ToLower(value) {
	case "true":
		*b = true
	case "false":
		*b = false
	default:
		return errors.New(value + " is not a Boolean value")

	}
	return nil
}

func (b Boolean) String() string {
	if b {
		return "true"
	}
	return "false"
}
