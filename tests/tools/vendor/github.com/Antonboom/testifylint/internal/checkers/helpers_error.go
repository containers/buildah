package checkers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

var (
	errorObj   = types.Universe.Lookup("error")
	errorType  = errorObj.Type()
	errorIface = errorType.Underlying().(*types.Interface)
)

func isError(pass *analysis.Pass, expr ast.Expr) bool {
	return pass.TypesInfo.TypeOf(expr) == errorType
}

func isErrorsIsCall(pass *analysis.Pass, ce *ast.CallExpr) bool {
	return isErrorsPkgFnCall(pass, ce, "Is")
}

func isErrorsAsCall(pass *analysis.Pass, ce *ast.CallExpr) bool {
	return isErrorsPkgFnCall(pass, ce, "As")
}

func isErrorsPkgFnCall(pass *analysis.Pass, ce *ast.CallExpr, fn string) bool {
	se, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	errorsIsObj := analysisutil.ObjectOf(pass.Pkg, "errors", fn)
	if errorsIsObj == nil {
		return false
	}

	return analysisutil.IsObj(pass.TypesInfo, se.Sel, errorsIsObj)
}
