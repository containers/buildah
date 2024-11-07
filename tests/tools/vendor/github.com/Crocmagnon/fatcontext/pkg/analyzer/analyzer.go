package analyzer

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "fatcontext",
	Doc:      "detects nested contexts in loops and function literals",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

var errUnknown = errors.New("unknown node type")

func run(pass *analysis.Pass) (interface{}, error) {
	inspctr := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.FuncLit)(nil),
	}

	inspctr.Preorder(nodeFilter, func(node ast.Node) {
		body, err := getBody(node)
		if err != nil {
			return
		}

		assignStmt := findNestedContext(pass, node, body.List)
		if assignStmt == nil {
			return
		}

		suggestedStmt := ast.AssignStmt{
			Lhs:    assignStmt.Lhs,
			TokPos: assignStmt.TokPos,
			Tok:    token.DEFINE,
			Rhs:    assignStmt.Rhs,
		}
		suggested, err := render(pass.Fset, &suggestedStmt)

		var fixes []analysis.SuggestedFix
		if err == nil {
			fixes = append(fixes, analysis.SuggestedFix{
				Message: "replace `=` with `:=`",
				TextEdits: []analysis.TextEdit{
					{
						Pos:     assignStmt.Pos(),
						End:     assignStmt.End(),
						NewText: []byte(suggested),
					},
				},
			})
		}

		pass.Report(analysis.Diagnostic{
			Pos:            assignStmt.Pos(),
			Message:        getReportMessage(node),
			SuggestedFixes: fixes,
		})
	})

	return nil, nil
}

func getReportMessage(node ast.Node) string {
	switch node.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return "nested context in loop"
	case *ast.FuncLit:
		return "nested context in function literal"
	default:
		return "unsupported nested context type"
	}
}

func getBody(node ast.Node) (*ast.BlockStmt, error) {
	forStmt, ok := node.(*ast.ForStmt)
	if ok {
		return forStmt.Body, nil
	}

	rangeStmt, ok := node.(*ast.RangeStmt)
	if ok {
		return rangeStmt.Body, nil
	}

	funcLit, ok := node.(*ast.FuncLit)
	if ok {
		return funcLit.Body, nil
	}

	return nil, errUnknown
}

func findNestedContext(pass *analysis.Pass, node ast.Node, stmts []ast.Stmt) *ast.AssignStmt {
	for _, stmt := range stmts {
		// Recurse if necessary
		if inner, ok := stmt.(*ast.BlockStmt); ok {
			found := findNestedContext(pass, node, inner.List)
			if found != nil {
				return found
			}
		}

		if inner, ok := stmt.(*ast.IfStmt); ok {
			found := findNestedContext(pass, node, inner.Body.List)
			if found != nil {
				return found
			}
		}

		if inner, ok := stmt.(*ast.SwitchStmt); ok {
			found := findNestedContext(pass, node, inner.Body.List)
			if found != nil {
				return found
			}
		}

		if inner, ok := stmt.(*ast.CaseClause); ok {
			found := findNestedContext(pass, node, inner.Body)
			if found != nil {
				return found
			}
		}

		if inner, ok := stmt.(*ast.SelectStmt); ok {
			found := findNestedContext(pass, node, inner.Body.List)
			if found != nil {
				return found
			}
		}

		if inner, ok := stmt.(*ast.CommClause); ok {
			found := findNestedContext(pass, node, inner.Body)
			if found != nil {
				return found
			}
		}

		// Actually check for nested context
		assignStmt, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}

		t := pass.TypesInfo.TypeOf(assignStmt.Lhs[0])
		if t == nil {
			continue
		}

		if t.String() != "context.Context" {
			continue
		}

		if assignStmt.Tok == token.DEFINE {
			continue
		}

		// allow assignment to non-pointer children of values defined within the loop
		if lhs := getRootIdent(pass, assignStmt.Lhs[0]); lhs != nil {
			if obj := pass.TypesInfo.ObjectOf(lhs); obj != nil {
				if checkObjectScopeWithinNode(obj.Parent(), node) {
					continue // definition is within the loop
				}
			}
		}

		return assignStmt
	}

	return nil
}

func checkObjectScopeWithinNode(scope *types.Scope, node ast.Node) bool {
	if scope == nil {
		return false
	}

	if scope.Pos() >= node.Pos() && scope.End() <= node.End() {
		return true
	}

	return false
}

func getRootIdent(pass *analysis.Pass, node ast.Node) *ast.Ident {
	for {
		switch n := node.(type) {
		case *ast.Ident:
			return n
		case *ast.IndexExpr:
			node = n.X
		case *ast.SelectorExpr:
			if sel, ok := pass.TypesInfo.Selections[n]; ok && sel.Indirect() {
				return nil // indirected (pointer) roots don't imply a (safe) copy
			}
			node = n.X
		default:
			return nil
		}
	}
}

// render returns the pretty-print of the given node
func render(fset *token.FileSet, x interface{}) (string, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, x); err != nil {
		return "", fmt.Errorf("printing node: %w", err)
	}
	return buf.String(), nil
}
