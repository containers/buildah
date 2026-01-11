package rule

import (
	"fmt"
	"go/ast"
	"regexp"
	"sync"

	"github.com/mgechev/revive/lint"
)

// UnusedParamRule lints unused params in functions.
type UnusedParamRule struct {
	configured bool
	// regex to check if some name is valid for unused parameter, "^_$" by default
	allowRegex *regexp.Regexp
	failureMsg string
	sync.Mutex
}

func (r *UnusedParamRule) configure(args lint.Arguments) {
	r.Lock()
	defer r.Unlock()

	if r.configured {
		return
	}
	r.configured = true

	// while by default args is an array, i think it's good to provide structures inside it by default, not arrays or primitives
	// it's more compatible to JSON nature of configurations
	var allowedRegexStr string
	if len(args) == 0 {
		allowedRegexStr = "^_$"
		r.failureMsg = "parameter '%s' seems to be unused, consider removing or renaming it as _"
	} else {
		// Arguments = [{}]
		options := args[0].(map[string]any)
		// Arguments = [{allowedRegex="^_"}]

		if allowedRegexParam, ok := options["allowRegex"]; ok {
			allowedRegexStr, ok = allowedRegexParam.(string)
			if !ok {
				panic(fmt.Errorf("error configuring %s rule: allowedRegex is not string but [%T]", r.Name(), allowedRegexParam))
			}
		}
	}
	var err error
	r.allowRegex, err = regexp.Compile(allowedRegexStr)
	if err != nil {
		panic(fmt.Errorf("error configuring %s rule: allowedRegex is not valid regex [%s]: %v", r.Name(), allowedRegexStr, err))
	}

	if r.failureMsg == "" {
		r.failureMsg = "parameter '%s' seems to be unused, consider removing or renaming it to match " + r.allowRegex.String()
	}
}

// Apply applies the rule to given file.
func (r *UnusedParamRule) Apply(file *lint.File, args lint.Arguments) []lint.Failure {
	r.configure(args)
	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}
	w := lintUnusedParamRule{
		onFailure:  onFailure,
		allowRegex: r.allowRegex,
		failureMsg: r.failureMsg,
	}

	ast.Walk(w, file.AST)

	return failures
}

// Name returns the rule name.
func (*UnusedParamRule) Name() string {
	return "unused-parameter"
}

type lintUnusedParamRule struct {
	onFailure  func(lint.Failure)
	allowRegex *regexp.Regexp
	failureMsg string
}

func (w lintUnusedParamRule) Visit(node ast.Node) ast.Visitor {
	var (
		funcType *ast.FuncType
		funcBody *ast.BlockStmt
	)
	switch n := node.(type) {
	case *ast.FuncLit:
		funcType = n.Type
		funcBody = n.Body
	case *ast.FuncDecl:
		if n.Body == nil {
			return nil // skip, is a function prototype
		}

		funcType = n.Type
		funcBody = n.Body
	default:
		return w // skip, not a function
	}

	params := retrieveNamedParams(funcType.Params)
	if len(params) < 1 {
		return w // skip, func without parameters
	}

	// inspect the func body looking for references to parameters
	fselect := func(n ast.Node) bool {
		ident, isAnID := n.(*ast.Ident)

		if !isAnID {
			return false
		}

		_, isAParam := params[ident.Obj]
		if isAParam {
			params[ident.Obj] = false // mark as used
		}

		return false
	}
	_ = pick(funcBody, fselect)

	for _, p := range funcType.Params.List {
		for _, n := range p.Names {
			if w.allowRegex.FindStringIndex(n.Name) != nil {
				continue
			}
			if params[n.Obj] {
				w.onFailure(lint.Failure{
					Confidence: 1,
					Node:       n,
					Category:   "bad practice",
					Failure:    fmt.Sprintf(w.failureMsg, n.Name),
				})
			}
		}
	}

	return w // full method body was inspected
}

func retrieveNamedParams(params *ast.FieldList) map[*ast.Object]bool {
	result := map[*ast.Object]bool{}
	if params.List == nil {
		return result
	}

	for _, p := range params.List {
		for _, n := range p.Names {
			if n.Name == "_" {
				continue
			}

			result[n.Obj] = true
		}
	}

	return result
}
