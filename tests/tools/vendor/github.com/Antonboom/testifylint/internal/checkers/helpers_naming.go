package checkers

import (
	"go/ast"
	"regexp"
)

func isStructVarNamedAsExpected(pattern *regexp.Regexp, e ast.Expr) bool {
	s, ok := e.(*ast.SelectorExpr)
	return ok && isIdentNamedAsExpected(pattern, s.X)
}

func isStructFieldNamedAsExpected(pattern *regexp.Regexp, e ast.Expr) bool {
	s, ok := e.(*ast.SelectorExpr)
	return ok && isIdentNamedAsExpected(pattern, s.Sel)
}

func isIdentNamedAsExpected(pattern *regexp.Regexp, e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && pattern.MatchString(id.Name)
}
