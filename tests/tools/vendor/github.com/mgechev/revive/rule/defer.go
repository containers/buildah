package rule

import (
	"fmt"
	"go/ast"
	"sync"

	"github.com/mgechev/revive/lint"
)

// DeferRule lints unused params in functions.
type DeferRule struct {
	allow map[string]bool
	sync.Mutex
}

func (r *DeferRule) configure(arguments lint.Arguments) {
	r.Lock()
	if r.allow == nil {
		r.allow = r.allowFromArgs(arguments)
	}
	r.Unlock()
}

// Apply applies the rule to given file.
func (r *DeferRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure
	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}
	w := lintDeferRule{onFailure: onFailure, allow: r.allow}

	ast.Walk(w, file.AST)

	return failures
}

// Name returns the rule name.
func (*DeferRule) Name() string {
	return "defer"
}

func (*DeferRule) allowFromArgs(args lint.Arguments) map[string]bool {
	if len(args) < 1 {
		allow := map[string]bool{
			"loop":              true,
			"call-chain":        true,
			"method-call":       true,
			"return":            true,
			"recover":           true,
			"immediate-recover": true,
		}

		return allow
	}

	aa, ok := args[0].([]interface{})
	if !ok {
		panic(fmt.Sprintf("Invalid argument '%v' for 'defer' rule. Expecting []string, got %T", args[0], args[0]))
	}

	allow := make(map[string]bool, len(aa))
	for _, subcase := range aa {
		sc, ok := subcase.(string)
		if !ok {
			panic(fmt.Sprintf("Invalid argument '%v' for 'defer' rule. Expecting string, got %T", subcase, subcase))
		}
		allow[sc] = true
	}

	return allow
}

type lintDeferRule struct {
	onFailure  func(lint.Failure)
	inALoop    bool
	inADefer   bool
	inAFuncLit bool
	allow      map[string]bool
}

func (w lintDeferRule) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ForStmt:
		w.visitSubtree(n.Body, w.inADefer, true, w.inAFuncLit)
		return nil
	case *ast.RangeStmt:
		w.visitSubtree(n.Body, w.inADefer, true, w.inAFuncLit)
		return nil
	case *ast.FuncLit:
		w.visitSubtree(n.Body, w.inADefer, false, true)
		return nil
	case *ast.ReturnStmt:
		if len(n.Results) != 0 && w.inADefer && w.inAFuncLit {
			w.newFailure("return in a defer function has no effect", n, 1.0, "logic", "return")
		}
	case *ast.CallExpr:
		if !w.inADefer && isIdent(n.Fun, "recover") {
			// func fn() { recover() }
			//
			// confidence is not 1 because recover can be in a function that is deferred elsewhere
			w.newFailure("recover must be called inside a deferred function", n, 0.8, "logic", "recover")
		} else if w.inADefer && !w.inAFuncLit && isIdent(n.Fun, "recover") {
			// defer helper(recover())
			//
			// confidence is not truly 1 because this could be in a correctly-deferred func,
			// but it is very likely to be a misunderstanding of defer's behavior around arguments.
			w.newFailure("recover must be called inside a deferred function, this is executing recover immediately", n, 1, "logic", "immediate-recover")
		}
	case *ast.DeferStmt:
		if isIdent(n.Call.Fun, "recover") {
			// defer recover()
			//
			// confidence is not truly 1 because this could be in a correctly-deferred func,
			// but normally this doesn't suppress a panic, and even if it did it would silently discard the value.
			w.newFailure("recover must be called inside a deferred function, this is executing recover immediately", n, 1, "logic", "immediate-recover")
		}
		w.visitSubtree(n.Call.Fun, true, false, false)
		for _, a := range n.Call.Args {
			w.visitSubtree(a, true, false, false) // check arguments, they should not contain recover()
		}

		if w.inALoop {
			w.newFailure("prefer not to defer inside loops", n, 1.0, "bad practice", "loop")
		}

		switch fn := n.Call.Fun.(type) {
		case *ast.CallExpr:
			w.newFailure("prefer not to defer chains of function calls", fn, 1.0, "bad practice", "call-chain")
		case *ast.SelectorExpr:
			if id, ok := fn.X.(*ast.Ident); ok {
				isMethodCall := id != nil && id.Obj != nil && id.Obj.Kind == ast.Typ
				if isMethodCall {
					w.newFailure("be careful when deferring calls to methods without pointer receiver", fn, 0.8, "bad practice", "method-call")
				}
			}
		}
		return nil
	}

	return w
}

func (w lintDeferRule) visitSubtree(n ast.Node, inADefer, inALoop, inAFuncLit bool) {
	nw := lintDeferRule{
		onFailure:  w.onFailure,
		inADefer:   inADefer,
		inALoop:    inALoop,
		inAFuncLit: inAFuncLit,
		allow:      w.allow,
	}
	ast.Walk(nw, n)
}

func (w lintDeferRule) newFailure(msg string, node ast.Node, confidence float64, cat, subcase string) {
	if !w.allow[subcase] {
		return
	}

	w.onFailure(lint.Failure{
		Confidence: confidence,
		Node:       node,
		Category:   cat,
		Failure:    msg,
	})
}
