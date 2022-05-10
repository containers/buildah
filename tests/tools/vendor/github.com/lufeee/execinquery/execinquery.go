package execinquery

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "execinquery is a linter about query string checker in Query function which reads your Go src files and warning it finds"

// Analyzer is checking database/sql pkg Query's function
var Analyzer = &analysis.Analyzer{
	Name: "execinquery",
	Doc:  doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	inspect.Preorder(nil, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CallExpr:
			selector, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				break
			}

			if !strings.Contains(selector.Sel.Name, "Query") {
				break
			}

			var i int
			if strings.Contains(selector.Sel.Name, "Context") {
				i = 1
			}

			var s string
			switch arg := n.Args[i].(type) {
			case *ast.BasicLit:
				s = strings.Replace(arg.Value, "\"", "", -1)
			case *ast.Ident:
				stmt, ok := arg.Obj.Decl.(*ast.AssignStmt)
				if !ok {
					break
				}
				for _, stmt := range stmt.Rhs {
					basicLit, ok := stmt.(*ast.BasicLit)
					if !ok {
						continue
					}
					s = strings.Replace(basicLit.Value, "\"", "", -1)
				}
			default:
				break
			}

			if strings.HasPrefix(strings.ToLower(s), "select") {
				break
			}
			s = strings.ToTitle(strings.Split(s, " ")[0])
			pass.Reportf(n.Fun.Pos(), "It's better to use Execute method instead of %s method to execute `%s` query", selector.Sel.Name, s)
		}
	})
	return nil, nil
}
