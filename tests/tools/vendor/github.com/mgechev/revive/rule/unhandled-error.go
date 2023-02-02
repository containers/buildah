package rule

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"
	"sync"

	"github.com/mgechev/revive/lint"
)

// UnhandledErrorRule lints given else constructs.
type UnhandledErrorRule struct {
	ignoreList []*regexp.Regexp
	sync.Mutex
}

func (r *UnhandledErrorRule) configure(arguments lint.Arguments) {
	r.Lock()
	if r.ignoreList == nil {
		for _, arg := range arguments {
			argStr, ok := arg.(string)
			if !ok {
				panic(fmt.Sprintf("Invalid argument to the unhandled-error rule. Expecting a string, got %T", arg))
			}

			argStr = strings.Trim(argStr, " ")
			if argStr == "" {
				panic("Invalid argument to the unhandled-error rule, expected regular expression must not be empty.")
			}

			exp, err := regexp.Compile(argStr)
			if err != nil {
				panic(fmt.Sprintf("Invalid argument to the unhandled-error rule: regexp %q does not compile: %v", argStr, err))
			}

			r.ignoreList = append(r.ignoreList, exp)
		}
	}
	r.Unlock()
}

// Apply applies the rule to given file.
func (r *UnhandledErrorRule) Apply(file *lint.File, args lint.Arguments) []lint.Failure {
	r.configure(args)

	var failures []lint.Failure

	walker := &lintUnhandledErrors{
		ignoreList: r.ignoreList,
		pkg:        file.Pkg,
		onFailure: func(failure lint.Failure) {
			failures = append(failures, failure)
		},
	}

	file.Pkg.TypeCheck()
	ast.Walk(walker, file.AST)

	return failures
}

// Name returns the rule name.
func (*UnhandledErrorRule) Name() string {
	return "unhandled-error"
}

type lintUnhandledErrors struct {
	ignoreList []*regexp.Regexp
	pkg        *lint.Package
	onFailure  func(lint.Failure)
}

// Visit looks for statements that are function calls.
// If the called function returns a value of type error a failure will be created.
func (w *lintUnhandledErrors) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ExprStmt:
		fCall, ok := n.X.(*ast.CallExpr)
		if !ok {
			return nil // not a function call
		}

		funcType := w.pkg.TypeOf(fCall)
		if funcType == nil {
			return nil // skip, type info not available
		}

		switch t := funcType.(type) {
		case *types.Named:
			if !w.isTypeError(t) {
				return nil // func call does not return an error
			}

			w.addFailure(fCall)
		default:
			retTypes, ok := funcType.Underlying().(*types.Tuple)
			if !ok {
				return nil // skip, unable to retrieve return type of the called function
			}

			if w.returnsAnError(retTypes) {
				w.addFailure(fCall)
			}
		}
	}
	return w
}

func (w *lintUnhandledErrors) addFailure(n *ast.CallExpr) {
	name := w.funcName(n)
	if w.isIgnoredFunc(name) {
		return
	}

	w.onFailure(lint.Failure{
		Category:   "bad practice",
		Confidence: 1,
		Node:       n,
		Failure:    fmt.Sprintf("Unhandled error in call to function %v", gofmt(n.Fun)),
	})
}

func (w *lintUnhandledErrors) funcName(call *ast.CallExpr) string {
	fn, ok := w.getFunc(call)
	if !ok {
		return gofmt(call.Fun)
	}

	name := fn.FullName()
	name = strings.Replace(name, "(", "", -1)
	name = strings.Replace(name, ")", "", -1)
	name = strings.Replace(name, "*", "", -1)

	return name
}

func (w *lintUnhandledErrors) isIgnoredFunc(funcName string) bool {
	for _, pattern := range w.ignoreList {
		if len(pattern.FindString(funcName)) == len(funcName) {
			return true
		}
	}

	return false
}

func (*lintUnhandledErrors) isTypeError(t *types.Named) bool {
	const errorTypeName = "_.error"

	return t.Obj().Id() == errorTypeName
}

func (w *lintUnhandledErrors) returnsAnError(tt *types.Tuple) bool {
	for i := 0; i < tt.Len(); i++ {
		nt, ok := tt.At(i).Type().(*types.Named)
		if ok && w.isTypeError(nt) {
			return true
		}
	}
	return false
}

func (w *lintUnhandledErrors) getFunc(call *ast.CallExpr) (*types.Func, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	fn, ok := w.pkg.TypesInfo().ObjectOf(sel.Sel).(*types.Func)
	if !ok {
		return nil, false
	}

	return fn, true
}
