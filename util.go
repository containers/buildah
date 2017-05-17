package buildah

import (
	"github.com/containers/storage/pkg/reexec"
)

// InitReexec is a wrapper for reexec.Init().  It should be called at
// the start of main(), and if it returns true, main() should return
// immediately.
func InitReexec() bool {
	return reexec.Init()
}

func copyStringStringMap(m map[string]string) map[string]string {
	n := map[string]string{}
	for k, v := range m {
		n[k] = v
	}
	return n
}

func copyStringSlice(s []string) []string {
	t := make([]string, len(s))
	copy(t, s)
	return t
}
