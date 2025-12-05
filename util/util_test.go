package util

import (
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
	mounts1b := []string{
		"/",
		"/a",
		"/b",
		"/a/b",
		"/a/c",
		"/a/b/c",
		"/a/bb/c",
		"/aa/b/c",
	}
	sorted := SortMounts(mounts1a)
	sortedDests := make([]string, len(mounts1a))
	for i := range sorted {
		sortedDests[i] = sorted[i].Destination
	}
	assert.Equalf(t, mounts1b, sortedDests, "sort returned results in unexpected by-destination order")
}
