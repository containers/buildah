package analysisutil

import (
	"go/ast"
	"strconv"
)

// Imports tells if the file imports at least one of the packages.
// If no packages provided then function returns false.
func Imports(file *ast.File, pkgs ...string) bool {
	for _, i := range file.Imports {
		if i.Path == nil {
			continue
		}

		path, err := strconv.Unquote(i.Path.Value)
		if err != nil {
			continue
		}
		// NOTE(a.telyshev): Don't use `slices.Contains` to keep the minimum module version 1.20.
		for _, pkg := range pkgs { // Small O(n).
			if pkg == path {
				return true
			}
		}
	}
	return false
}
