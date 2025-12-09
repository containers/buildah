package rule

import (
	"fmt"
	"go/ast"
	"sync"

	"github.com/mgechev/revive/lint"
)

// ArgumentsLimitRule lints given else constructs.
type ArgumentsLimitRule struct {
	total int
	sync.Mutex
}

const defaultArgumentsLimit = 8

func (r *ArgumentsLimitRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()
	if r.total == 0 {
		if len(arguments) < 1 {
			r.total = defaultArgumentsLimit
			return
		}

		total, ok := arguments[0].(int64) // Alt. non panicking version
		if !ok {
			panic(`invalid value passed as argument number to the "argument-limit" rule`)
		}
		r.total = int(total)
	}
}

// Apply applies the rule to given file.
func (r *ArgumentsLimitRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure
	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	walker := lintArgsNum{
		total:     r.total,
		onFailure: onFailure,
	}

	ast.Walk(walker, file.AST)

	return failures
}

// Name returns the rule name.
func (*ArgumentsLimitRule) Name() string {
	return "argument-limit"
}

type lintArgsNum struct {
	total     int
	onFailure func(lint.Failure)
}

func (w lintArgsNum) Visit(n ast.Node) ast.Visitor {
	node, ok := n.(*ast.FuncDecl)
	if ok {
		num := 0
		for _, l := range node.Type.Params.List {
			for range l.Names {
				num++
			}
		}
		if num > w.total {
			w.onFailure(lint.Failure{
				Confidence: 1,
				Failure:    fmt.Sprintf("maximum number of arguments per function exceeded; max %d but got %d", w.total, num),
				Node:       node.Type,
			})
			return w
		}
	}
	return w
}
