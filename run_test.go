package buildah

import (
	"strings"
	"testing"

	"github.com/opencontainers/runtime-tools/generate"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestAddRlimits(t *testing.T) {
	tt := []struct {
		name   string
		ulimit []string
		test   func(error, *generate.Generator) error
	}{
		{
			name:   "empty ulimit",
			ulimit: []string{},
			test: func(e error, g *generate.Generator) error {
				return e
			},
		},
		{
			name:   "invalid ulimit argument",
			ulimit: []string{"bla"},
			test: func(e error, g *generate.Generator) error {
				if e == nil {
					return errors.Errorf("expected to receive an error but got nil")
				}
				errMsg := "invalid ulimit argument"
				if !strings.Contains(e.Error(), errMsg) {
					return errors.Errorf("expected error message to include %#v in %#v", errMsg, e.Error())
				}
				return nil
			},
		},
		{
			name:   "invalid ulimit type",
			ulimit: []string{"bla=hard"},
			test: func(e error, g *generate.Generator) error {
				if e == nil {
					return errors.Errorf("expected to receive an error but got nil")
				}
				errMsg := "invalid ulimit type"
				if !strings.Contains(e.Error(), errMsg) {
					return errors.Errorf("expected error message to include %#v in %#v", errMsg, e.Error())
				}
				return nil
			},
		},
		{
			name:   "valid ulimit",
			ulimit: []string{"fsize=1024:4096"},
			test: func(e error, g *generate.Generator) error {
				if e != nil {
					return e
				}
				rlimits := g.Config.Process.Rlimits
				for _, rlimit := range rlimits {
					if rlimit.Type == "RLIMIT_FSIZE" {
						if rlimit.Hard != 4096 {
							return errors.Errorf("expected spec to have %#v hard limit set to %v but got %v", rlimit.Type, 4096, rlimit.Hard)
						}
						if rlimit.Soft != 1024 {
							return errors.Errorf("expected spec to have %#v hard limit set to %v but got %v", rlimit.Type, 1024, rlimit.Soft)
						}
						return nil
					}
				}
				return errors.Errorf("expected spec to have RLIMIT_FSIZE")
			},
		},
	}

	for _, tst := range tt {
		g, _ := generate.New("linux")
		err := addRlimits(tst.ulimit, &g, []string{})
		if testErr := tst.test(err, &g); testErr != nil {
			t.Errorf("test %#v failed: %v", tst.name, testErr)
		}
	}
}

func TestPathCovers(t *testing.T) {
	testCases := []struct {
		parent, subdirectory string
		covers               bool
	}{
		{"/", "/subdirectory", true},
		{"/run/secrets", "/run/secrets/other", true},
		{"/run/secrets", "/run/tmp", false},
		{"/run/secrets", "/run/secrets2", false},
		{"/run/secrets", "/tmp", false},
		{"/run", "/tmp", false},
		{"/run", "/run", true},
	}
	for i, testCase := range testCases {
		if testCase.covers {
			assert.Truef(t, pathCovers(testCase.parent, testCase.subdirectory), "case %d: parent=%q,subdirectory=%q", i, testCase.parent, testCase.subdirectory)
		} else {
			assert.Falsef(t, pathCovers(testCase.parent, testCase.subdirectory), "case %d: parent=%q,subdirectory=%q", i, testCase.parent, testCase.subdirectory)
		}
	}
}
