package rules

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/securego/gosec/v2"
)

type deferType struct {
	typ     string
	methods []string
}

type badDefer struct {
	gosec.MetaData
	types []deferType
}

func (r *badDefer) ID() string {
	return r.MetaData.ID
}

func normalize(typ string) string {
	return strings.TrimPrefix(typ, "*")
}

func contains(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

func (r *badDefer) Match(n ast.Node, c *gosec.Context) (*gosec.Issue, error) {
	if deferStmt, ok := n.(*ast.DeferStmt); ok {
		for _, deferTyp := range r.types {
			if issue := r.checkChild(n, c, deferStmt.Call, deferTyp); issue != nil {
				return issue, nil
			}
			if issue := r.checkFunction(n, c, deferStmt, deferTyp); issue != nil {
				return issue, nil
			}
		}
	}

	return nil, nil
}

func (r *badDefer) checkChild(n ast.Node, c *gosec.Context, callExp *ast.CallExpr, deferTyp deferType) *gosec.Issue {
	if typ, method, err := gosec.GetCallInfo(callExp, c); err == nil {
		if normalize(typ) == deferTyp.typ && contains(deferTyp.methods, method) {
			return gosec.NewIssue(c, n, r.ID(), fmt.Sprintf(r.What, method, typ), r.Severity, r.Confidence)
		}
	}
	return nil
}

func (r *badDefer) checkFunction(n ast.Node, c *gosec.Context, deferStmt *ast.DeferStmt, deferTyp deferType) *gosec.Issue {
	if anonFunc, isAnonFunc := deferStmt.Call.Fun.(*ast.FuncLit); isAnonFunc {
		for _, subElem := range anonFunc.Body.List {
			if issue := r.checkStmt(n, c, subElem, deferTyp); issue != nil {
				return issue
			}
		}
	}
	return nil
}

func (r *badDefer) checkStmt(n ast.Node, c *gosec.Context, subElem ast.Stmt, deferTyp deferType) *gosec.Issue {
	switch stmt := subElem.(type) {
	case *ast.AssignStmt:
		for _, rh := range stmt.Rhs {
			if e, isCallExp := rh.(*ast.CallExpr); isCallExp {
				return r.checkChild(n, c, e, deferTyp)
			}
		}
	case *ast.IfStmt:
		if s, is := stmt.Init.(*ast.AssignStmt); is {
			return r.checkStmt(n, c, s, deferTyp)
		}
	}
	return nil
}

// NewDeferredClosing detects unsafe defer of error returning methods
func NewDeferredClosing(id string, conf gosec.Config) (gosec.Rule, []ast.Node) {
	return &badDefer{
		types: []deferType{
			{
				typ:     "os.File",
				methods: []string{"Close"},
			},
		},
		MetaData: gosec.MetaData{
			ID:         id,
			Severity:   gosec.Medium,
			Confidence: gosec.High,
			What:       "Deferring unsafe method %q on type %q",
		},
	}, []ast.Node{(*ast.DeferStmt)(nil)}
}
