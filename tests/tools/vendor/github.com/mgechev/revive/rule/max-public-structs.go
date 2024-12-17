package rule

import (
	"go/ast"
	"strings"
	"sync"

	"github.com/mgechev/revive/lint"
)

// MaxPublicStructsRule lints given else constructs.
type MaxPublicStructsRule struct {
	max int64
	sync.Mutex
}

const defaultMaxPublicStructs = 5

func (r *MaxPublicStructsRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()
	if r.max < 1 {
		if len(arguments) < 1 {
			r.max = defaultMaxPublicStructs
			return
		}

		checkNumberOfArguments(1, arguments, r.Name())

		max, ok := arguments[0].(int64) // Alt. non panicking version
		if !ok {
			panic(`invalid value passed as argument number to the "max-public-structs" rule`)
		}
		r.max = max
	}
}

// Apply applies the rule to given file.
func (r *MaxPublicStructsRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure

	fileAst := file.AST

	walker := &lintMaxPublicStructs{
		fileAst: fileAst,
		onFailure: func(failure lint.Failure) {
			failures = append(failures, failure)
		},
	}

	ast.Walk(walker, fileAst)

	if walker.current > r.max {
		walker.onFailure(lint.Failure{
			Failure:    "you have exceeded the maximum number of public struct declarations",
			Confidence: 1,
			Node:       fileAst,
			Category:   "style",
		})
	}

	return failures
}

// Name returns the rule name.
func (*MaxPublicStructsRule) Name() string {
	return "max-public-structs"
}

type lintMaxPublicStructs struct {
	current   int64
	fileAst   *ast.File
	onFailure func(lint.Failure)
}

func (w *lintMaxPublicStructs) Visit(n ast.Node) ast.Visitor {
	switch v := n.(type) {
	case *ast.TypeSpec:
		name := v.Name.Name
		first := string(name[0])
		if strings.ToUpper(first) == first {
			w.current++
		}
	}
	return w
}
