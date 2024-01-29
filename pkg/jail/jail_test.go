//go:build freebsd
// +build freebsd

package jail

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	tt := []struct {
		version                  string
		shouldFail               bool
		kind                     string
		major, minor, patchlevel int
	}{
		{"14.0-RELEASE", false, "RELEASE", 14, 0, 0},
		{"14.0-RELEASE-p3", false, "RELEASE", 14, 0, 3},
		{"13.2-STABLE", false, "STABLE", 13, 2, 0},
		{"14.0-STABLE", false, "STABLE", 14, 0, 0},
		{"15.0-CURRENT", false, "CURRENT", 15, 0, 0},

		{"14-RELEASE", true, "", -1, -1, -1},
		{"14.1-STABLE-p1", true, "", -1, -1, -1},
		{"14-RELEASE-p3", true, "", -1, -1, -1},
	}
	for _, tc := range tt {
		kind, major, minor, patchlevel, err := parseVersion(tc.version)
		if tc.shouldFail {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, kind, tc.kind)
			assert.Equal(t, major, tc.major)
			assert.Equal(t, minor, tc.minor)
			assert.Equal(t, patchlevel, tc.patchlevel)
		}

	}
}
