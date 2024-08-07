package checkers

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
	"github.com/Antonboom/testifylint/internal/testify"
)

func isEmptyInterface(pass *analysis.Pass, expr ast.Expr) bool {
	t, ok := pass.TypesInfo.Types[expr]
	if !ok {
		return false
	}

	iface, ok := t.Type.Underlying().(*types.Interface)
	return ok && iface.NumMethods() == 0
}

func implementsTestifySuite(pass *analysis.Pass, rcv ast.Expr) bool {
	suiteIfaceObj := analysisutil.ObjectOf(pass.Pkg, testify.SuitePkgPath, "TestingSuite")
	return (suiteIfaceObj != nil) && implements(pass, rcv, suiteIfaceObj)
}

func implementsTestingT(pass *analysis.Pass, arg ast.Expr) bool {
	return implementsAssertTestingT(pass, arg) || implementsRequireTestingT(pass, arg)
}

func implementsAssertTestingT(pass *analysis.Pass, e ast.Expr) bool {
	assertTestingTObj := analysisutil.ObjectOf(pass.Pkg, testify.AssertPkgPath, "TestingT")
	return (assertTestingTObj != nil) && implements(pass, e, assertTestingTObj)
}

func implementsRequireTestingT(pass *analysis.Pass, e ast.Expr) bool {
	requireTestingTObj := analysisutil.ObjectOf(pass.Pkg, testify.RequirePkgPath, "TestingT")
	return (requireTestingTObj != nil) && implements(pass, e, requireTestingTObj)
}

func implements(pass *analysis.Pass, e ast.Expr, ifaceObj types.Object) bool {
	t := pass.TypesInfo.TypeOf(e)
	if t == nil {
		return false
	}
	return types.Implements(t, ifaceObj.Type().Underlying().(*types.Interface))
}
