package analyzer

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "nonamedreturns",
	Doc:      "Reports all named returns",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// only filter function defintions
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspector.Preorder(nodeFilter, func(node ast.Node) {
		var funcResults *ast.FieldList

		switch n := node.(type) {
		case *ast.FuncLit:
			funcResults = n.Type.Results
		case *ast.FuncDecl:
			funcResults = n.Type.Results
		default:
			return
		}

		// no return values
		if funcResults == nil {
			return
		}

		resultsList := funcResults.List

		for _, p := range resultsList {
			if len(p.Names) == 0 {
				// all good, the parameter is not named
				continue
			}

			for _, n := range p.Names {
				pass.Reportf(node.Pos(), "named return %q with type %q found", n.Name, types.ExprString(p.Type))
			}
		}
	})

	return nil, nil
}
