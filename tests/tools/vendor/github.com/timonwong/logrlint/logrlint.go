package logrlint

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const Doc = "Check logr arguments."

var Analyzer = &analysis.Analyzer{
	Name:     "logrlint",
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

var isValidName = map[string]struct{}{
	"Error":      {},
	"Info":       {},
	"WithValues": {},
}

func isValidPackage(pass *analysis.Pass, fn *types.Func) bool {
	// We allow only logr package import path
	const packageName = "github.com/go-logr/logr"

	pkg := fn.Pkg()
	if pkg == nil {
		return false
	}
	pkgPath := pkg.Path()
	// Fast path: for GOPATH or go mod enabled packages
	if pkgPath == packageName {
		return true
	}

	// Special case for vendor
	vendorPath := fmt.Sprintf("%s/vendor/%s", pass.Pkg.Name(), packageName)
	return pkgPath == vendorPath
}

func checkEvenArguments(pass *analysis.Pass, call *ast.CallExpr) {
	fn, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if fn == nil {
		return // function pointer is not supported
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok || !sig.Variadic() {
		return // not variadic
	}

	if _, ok := isValidName[fn.Name()]; !ok {
		return
	}

	if !isValidPackage(pass, fn) {
		return
	}

	params := sig.Params()
	nparams := params.Len() // variadic => nonzero
	args := params.At(nparams - 1)
	iface, ok := args.Type().(*types.Slice).Elem().(*types.Interface)
	if !ok || !iface.Empty() {
		return // final (args) param is not ...interface{}
	}

	startIndex := nparams - 1
	variadicLen := len(call.Args) - (startIndex)
	if variadicLen%2 != 0 {
		firstArg := call.Args[startIndex]
		lastArg := call.Args[len(call.Args)-1]
		pass.Report(analysis.Diagnostic{
			Pos:      firstArg.Pos(),
			End:      lastArg.End(),
			Category: "logging",
			Message:  "odd number of arguments passed as key-value pairs for logging"})
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	insp.Preorder(nodeFilter, func(node ast.Node) {
		call := node.(*ast.CallExpr)

		typ := pass.TypesInfo.Types[call.Fun].Type
		if typ == nil {
			// Skip checking functions with unknown type.
			return
		}

		checkEvenArguments(pass, call)
	})

	return nil, nil
}
