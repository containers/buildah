package rule

import (
	"fmt"
	"go/ast"

	"github.com/mgechev/revive/lint"
)

// DataRaceRule lints assignments to value method-receivers.
type DataRaceRule struct{}

// Apply applies the rule to given file.
func (*DataRaceRule) Apply(file *lint.File, _ lint.Arguments) []lint.Failure {
	var failures []lint.Failure
	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}
	w := lintDataRaces{onFailure: onFailure, go122for: file.Pkg.IsAtLeastGo122()}

	ast.Walk(w, file.AST)

	return failures
}

// Name returns the rule name.
func (*DataRaceRule) Name() string {
	return "datarace"
}

type lintDataRaces struct {
	onFailure func(failure lint.Failure)
	go122for  bool
}

func (w lintDataRaces) Visit(n ast.Node) ast.Visitor {
	node, ok := n.(*ast.FuncDecl)
	if !ok {
		return w // not function declaration
	}
	if node.Body == nil {
		return nil // empty body
	}

	results := node.Type.Results

	returnIDs := map[*ast.Object]struct{}{}
	if results != nil {
		returnIDs = w.ExtractReturnIDs(results.List)
	}
	fl := &lintFunctionForDataRaces{onFailure: w.onFailure, returnIDs: returnIDs, rangeIDs: map[*ast.Object]struct{}{}, go122for: w.go122for}
	ast.Walk(fl, node.Body)

	return nil
}

func (lintDataRaces) ExtractReturnIDs(fields []*ast.Field) map[*ast.Object]struct{} {
	r := map[*ast.Object]struct{}{}
	for _, f := range fields {
		for _, id := range f.Names {
			r[id.Obj] = struct{}{}
		}
	}

	return r
}

type lintFunctionForDataRaces struct {
	_         struct{}
	onFailure func(failure lint.Failure)
	returnIDs map[*ast.Object]struct{}
	rangeIDs  map[*ast.Object]struct{}
	go122for  bool
}

func (w lintFunctionForDataRaces) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.RangeStmt:
		if n.Body == nil {
			return nil
		}

		getIds := func(exprs ...ast.Expr) []*ast.Ident {
			r := []*ast.Ident{}
			for _, expr := range exprs {
				if id, ok := expr.(*ast.Ident); ok {
					r = append(r, id)
				}
			}
			return r
		}

		ids := getIds(n.Key, n.Value)
		for _, id := range ids {
			w.rangeIDs[id.Obj] = struct{}{}
		}

		ast.Walk(w, n.Body)

		for _, id := range ids {
			delete(w.rangeIDs, id.Obj)
		}

		return nil // do not visit the body of the range, it has been already visited
	case *ast.GoStmt:
		f := n.Call.Fun
		funcLit, ok := f.(*ast.FuncLit)
		if !ok {
			return nil
		}
		selectIDs := func(n ast.Node) bool {
			_, ok := n.(*ast.Ident)
			return ok
		}

		ids := pick(funcLit.Body, selectIDs)
		for _, id := range ids {
			id := id.(*ast.Ident)
			_, isRangeID := w.rangeIDs[id.Obj]
			_, isReturnID := w.returnIDs[id.Obj]

			switch {
			case isRangeID && !w.go122for:
				w.onFailure(lint.Failure{
					Confidence: 1,
					Node:       id,
					Category:   "logic",
					Failure:    fmt.Sprintf("datarace: range value %s is captured (by-reference) in goroutine", id.Name),
				})
			case isReturnID:
				w.onFailure(lint.Failure{
					Confidence: 0.8,
					Node:       id,
					Category:   "logic",
					Failure:    fmt.Sprintf("potential datarace: return value %s is captured (by-reference) in goroutine", id.Name),
				})
			}
		}

		return nil
	}

	return w
}
