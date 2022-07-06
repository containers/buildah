package util

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/containers/common/pkg/config"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func TestMergeEnv(t *testing.T) {
	tests := [][3][]string{
		{
			[]string{"A=B", "B=C", "C=D"},
			nil,
			[]string{"A=B", "B=C", "C=D"},
		},
		{
			nil,
			[]string{"A=B", "B=C", "C=D"},
			[]string{"A=B", "B=C", "C=D"},
		},
		{
			[]string{"A=B", "B=C", "C=D", "E=F"},
			[]string{"B=O", "F=G"},
			[]string{"A=B", "B=O", "C=D", "E=F", "F=G"},
		},
	}
	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			result := MergeEnv(test[0], test[1])
			if len(result) != len(test[2]) {
				t.Fatalf("expected %v, got %v", test[2], result)
			}
			for i := range result {
				if result[i] != test[2][i] {
					t.Fatalf("expected %v, got %v", test[2], result)
				}
			}
		})
	}
}

func TestRuntime(t *testing.T) {
	os.Setenv("CONTAINERS_CONF", "/dev/null")
	conf, _ := config.Default()
	defaultRuntime := conf.Engine.OCIRuntime
	runtime := Runtime()
	if runtime != defaultRuntime {
		t.Fatalf("expected %v, got %v", runtime, defaultRuntime)
	}
	defaultRuntime = "myoci"
	os.Setenv("BUILDAH_RUNTIME", defaultRuntime)
	runtime = Runtime()
	if runtime != defaultRuntime {
		t.Fatalf("expected %v, got %v", runtime, defaultRuntime)
	}
}

func TestMountsSort(t *testing.T) {
	mounts1a := []specs.Mount{
		{
			Source:      "/a/bb/c",
			Destination: "/a/bb/c",
		},
		{
			Source:      "/a/b/c",
			Destination: "/a/b/c",
		},
		{
			Source:      "/a",
			Destination: "/a",
		},
		{
			Source:      "/a/b",
			Destination: "/a/b",
		},
		{
			Source:      "/d/e",
			Destination: "/a/c",
		},
		{
			Source:      "/b",
			Destination: "/b",
		},
		{
			Source:      "/",
			Destination: "/",
		},
		{
			Source:      "/a/b/c",
			Destination: "/aa/b/c",
		},
	}
	mounts1b := []specs.Mount{
		{
			Source:      "/xyz",
			Destination: "/",
		},
		{
			Source:      "/a",
			Destination: "/a",
		},
		{
			Source:      "/b",
			Destination: "/b",
		},
		{
			Source:      "/a/b",
			Destination: "/a/b",
		},
		{
			Source:      "/d/e",
			Destination: "/a/c",
		},
		{
			Source:      "/a/b/c",
			Destination: "/a/b/c",
		},
		{
			Source:      "/a/bb/c",
			Destination: "/a/bb/c",
		},
		{
			Source:      "/a/b/c",
			Destination: "/aa/b/c",
		},
	}
	sorted := SortMounts(mounts1a)
	for i := range sorted {
		if sorted[i].Destination != mounts1b[i].Destination {
			t.Fatalf("failed sort \n%+v\n%+v", mounts1b, sorted)
		}
	}

}

func TestCause(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		err         func() error
		expectedErr error
	}{
		{
			name:        "nil error",
			err:         func() error { return nil },
			expectedErr: nil,
		},
		{
			name:        "equal errors",
			err:         func() error { return errors.New("foo") },
			expectedErr: errors.New("foo"),
		},
		{
			name:        "wrapped error",
			err:         func() error { return fmt.Errorf("baz: %w", fmt.Errorf("bar: %w", errors.New("foo"))) },
			expectedErr: errors.New("foo"),
		},
		{
			name: "max depth reached",
			err: func() error {
				err := errors.New("error")
				for i := 0; i <= 101; i++ {
					err = fmt.Errorf("%d: %w", i, err)
				}
				return err
			},
			expectedErr: fmt.Errorf("0: %w", errors.New("error")),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := Cause(tc.err())
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}
