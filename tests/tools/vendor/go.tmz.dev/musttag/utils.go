package musttag

import (
	"fmt"
	"os/exec"
	"strings"
)

// mainModule returns the directory and the set of packages of the main module.
func mainModule() (dir string, packages map[string]struct{}, _ error) {
	// https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns
	// > When using modules, "all" expands to all packages in the main module
	// > and their dependencies, including dependencies needed by tests of any of those.

	// NOTE: the command may run out of file descriptors if go version <= 1.18,
	// especially on macOS, which has the default soft limit set to 256 (ulimit -nS).
	// Since go1.19 the limit is automatically increased to the maximum allowed value;
	// see https://github.com/golang/go/issues/46279 for details.
	cmd := [...]string{"go", "list", "-f={{if and (not .Standard) .Module.Main}}{{.ImportPath}}{{end}}", "all"}

	out, err := exec.Command(cmd[0], cmd[1:]...).Output()
	if err != nil {
		return "", nil, fmt.Errorf("running `go list all`: %w", err)
	}

	list := strings.Split(strings.TrimSpace(string(out)), "\n")
	packages = make(map[string]struct{}, len(list)*2)

	for _, pkg := range list {
		packages[pkg] = struct{}{}
		packages[pkg+"_test"] = struct{}{} // `*_test` packages belong to the main module, see issue #24.
	}

	out, err = exec.Command("go", "list", "-m", "-f={{.Dir}}").Output()
	if err != nil {
		return "", nil, fmt.Errorf("running `go list -m`: %w", err)
	}

	dir = strings.TrimSpace(string(out))
	return dir, packages, nil
}
