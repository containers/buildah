package rule

import (
	"go/ast"

	"github.com/mgechev/revive/lint"
)

// NestedStructs lints nested structs.
type NestedStructs struct{}

// Apply applies the rule to given file.
func (*NestedStructs) Apply(file *lint.File, _ lint.Arguments) []lint.Failure {
	var failures []lint.Failure

	walker := &lintNestedStructs{
		fileAST: file.AST,
		onFailure: func(failure lint.Failure) {
			failures = append(failures, failure)
		},
	}

	ast.Walk(walker, file.AST)

	return failures
}

// Name returns the rule name.
func (*NestedStructs) Name() string {
	return "nested-structs"
}

type lintNestedStructs struct {
	fileAST   *ast.File
	onFailure func(lint.Failure)
}

func (l *lintNestedStructs) Visit(n ast.Node) ast.Visitor {
	switch v := n.(type) {
	case *ast.FuncDecl:
		if v.Body != nil {
			ast.Walk(l, v.Body)
		}
		return nil
	case *ast.Field:
		_, isChannelField := v.Type.(*ast.ChanType)
		if isChannelField {
			return nil
		}

		filter := func(n ast.Node) bool {
			switch n.(type) {
			case *ast.StructType:
				return true
			default:
				return false
			}
		}
		structs := pick(v, filter, nil)
		for _, s := range structs {
			l.onFailure(lint.Failure{
				Failure:    "no nested structs are allowed",
				Category:   "style",
				Node:       s,
				Confidence: 1,
			})
		}
		return nil // no need to visit (again) the field
	}

	return l
}
