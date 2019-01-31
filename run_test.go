package buildah

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opencontainers/runtime-tools/generate"
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
					return fmt.Errorf("expected to receive an error but got nil")
				}
				errMsg := "invalid ulimit argument"
				if !strings.Contains(e.Error(), errMsg) {
					return fmt.Errorf("expected error message to include %#v in %#v", errMsg, e.Error())
				}
				return nil
			},
		},
		{
			name:   "invalid ulimit type",
			ulimit: []string{"bla=hard"},
			test: func(e error, g *generate.Generator) error {
				if e == nil {
					return fmt.Errorf("expected to receive an error but got nil")
				}
				errMsg := "invalid ulimit type"
				if !strings.Contains(e.Error(), errMsg) {
					return fmt.Errorf("expected error message to include %#v in %#v", errMsg, e.Error())
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
							return fmt.Errorf("expected spec to have %#v hard limit set to %v but got %v", rlimit.Type, 4096, rlimit.Hard)
						}
						if rlimit.Soft != 1024 {
							return fmt.Errorf("expected spec to have %#v hard limit set to %v but got %v", rlimit.Type, 1024, rlimit.Soft)
						}
						return nil
					}
				}
				return fmt.Errorf("expected spec to have RLIMIT_FSIZE")
			},
		},
	}

	for _, te := range tt {
		g, _ := generate.New("linux")
		err := addRlimits(te.ulimit, &g)
		if testErr := te.test(err, &g); testErr != nil {
			t.Errorf("test %#v failed: %v", te.name, testErr)
		}
	}
}
