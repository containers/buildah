package buildah

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapContainerNameToHostname(t *testing.T) {
	cases := [][2]string{
		{"trivial", "trivial"},
		{"Nottrivial", "Nottrivial"},
		{"0Nottrivial", "0Nottrivial"},
		{"0Nottrivi-al", "0Nottrivi-al"},
		{"-0Nottrivi-al", "0Nottrivi-al"},
		{".-0Nottrivi-.al", "0Nottrivi-.al"},
		{".-0Nottrivi-.al0123456789", "0Nottrivi-.al0123456789"},
		{".-0Nottrivi-.al0123456789+0123456789", "0Nottrivi-.al01234567890123456789"},
		{".-0Nottrivi-.al0123456789+0123456789/0123456789", "0Nottrivi-.al012345678901234567890123456789"},
		{".-0Nottrivi-.al0123456789+0123456789/0123456789%0123456789", "0Nottrivi-.al0123456789012345678901234567890123456789"},
		{".-0Nottrivi-.al0123456789+0123456789/0123456789%0123456789_0123456789", "0Nottrivi-.al01234567890123456789012345678901234567890123456789"},
		{".-0Nottrivi-.al0123456789+0123456789/0123456789%0123456789_0123456789:0123456", "0Nottrivi-.al012345678901234567890123456789012345678901234567890"},
		{".-0Nottrivi-.al0123456789+0123456789/0123456789%0123456789_0123456789:0123456789", "0Nottrivi-.al012345678901234567890123456789012345678901234567890"},
	}
	for i := range cases {
		t.Run(cases[i][0], func(t *testing.T) {
			sanitized := mapContainerNameToHostname(cases[i][0])
			assert.Equalf(t, cases[i][1], sanitized, "mapping container name %q to a valid hostname", cases[i][0])
		})
	}
}
