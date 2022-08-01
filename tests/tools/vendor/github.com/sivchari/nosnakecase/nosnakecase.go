package nosnakecase

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "nosnakecase is a linter that detects snake case of variable naming and function name."

// Analyzer is a nosnakecase linter.
var Analyzer = &analysis.Analyzer{
	Name: "nosnakecase",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	result := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.Ident)(nil),
	}

	result.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.Ident:
			report(pass, n.Pos(), n.Name)
		}
	})

	return nil, nil
}

func report(pass *analysis.Pass, pos token.Pos, name string) {
	// skip import _ "xxx"
	if name == "_" {
		return
	}

	// skip package xxx_test
	if strings.Contains(name, "_test") {
		return
	}

	// If prefix is Test or Benchmark, Fuzz, skip
	// FYI https://go.dev/blog/examples
	if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") || strings.HasPrefix(name, "Fuzz") {
		return
	}

	if strings.Contains(name, "_") {
		pass.Reportf(pos, "%s contains underscore. You should use mixedCap or MixedCap.", name)
		return
	}
}
