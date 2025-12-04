package rule

import (
	"fmt"
	"go/ast"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/mgechev/revive/lint"
)

const (
	defaultStrLitLimit = 2
	kindFLOAT          = "FLOAT"
	kindINT            = "INT"
	kindSTRING         = "STRING"
)

type allowList map[string]map[string]bool

func newAllowList() allowList {
	return map[string]map[string]bool{kindINT: {}, kindFLOAT: {}, kindSTRING: {}}
}

func (wl allowList) add(kind, list string) {
	elems := strings.Split(list, ",")
	for _, e := range elems {
		wl[kind][e] = true
	}
}

// AddConstantRule lints unused params in functions.
type AddConstantRule struct {
	allowList       allowList
	ignoreFunctions []*regexp.Regexp
	strLitLimit     int
	sync.Mutex
}

// Apply applies the rule to given file.
func (r *AddConstantRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	w := &lintAddConstantRule{
		onFailure:       onFailure,
		strLits:         make(map[string]int),
		strLitLimit:     r.strLitLimit,
		allowList:       r.allowList,
		ignoreFunctions: r.ignoreFunctions,
		structTags:      make(map[*ast.BasicLit]struct{}),
	}

	ast.Walk(w, file.AST)

	return failures
}

// Name returns the rule name.
func (*AddConstantRule) Name() string {
	return "add-constant"
}

type lintAddConstantRule struct {
	onFailure       func(lint.Failure)
	strLits         map[string]int
	strLitLimit     int
	allowList       allowList
	ignoreFunctions []*regexp.Regexp
	structTags      map[*ast.BasicLit]struct{}
}

func (w *lintAddConstantRule) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.CallExpr:
		w.checkFunc(n)
		return nil
	case *ast.GenDecl:
		return nil // skip declarations
	case *ast.BasicLit:
		if !w.isStructTag(n) {
			w.checkLit(n)
		}
	case *ast.StructType:
		if n.Fields != nil {
			for _, field := range n.Fields.List {
				if field.Tag != nil {
					w.structTags[field.Tag] = struct{}{}
				}
			}
		}
	}

	return w
}

func (w *lintAddConstantRule) checkFunc(expr *ast.CallExpr) {
	fName := w.getFuncName(expr)

	for _, arg := range expr.Args {
		switch t := arg.(type) {
		case *ast.CallExpr:
			w.checkFunc(t)
		case *ast.BasicLit:
			if w.isIgnoredFunc(fName) {
				continue
			}
			w.checkLit(t)
		}
	}
}

func (*lintAddConstantRule) getFuncName(expr *ast.CallExpr) string {
	switch f := expr.Fun.(type) {
	case *ast.SelectorExpr:
		switch prefix := f.X.(type) {
		case *ast.Ident:
			return prefix.Name + "." + f.Sel.Name
		case *ast.CallExpr:
			// If the selector is an CallExpr, like `fn().Info`, we return `.Info` as function name
			if f.Sel != nil {
				return "." + f.Sel.Name
			}
		}
	case *ast.Ident:
		return f.Name
	}

	return ""
}

func (w *lintAddConstantRule) checkLit(n *ast.BasicLit) {
	switch kind := n.Kind.String(); kind {
	case kindFLOAT, kindINT:
		w.checkNumLit(kind, n)
	case kindSTRING:
		w.checkStrLit(n)
	}
}

func (w *lintAddConstantRule) isIgnoredFunc(fName string) bool {
	for _, pattern := range w.ignoreFunctions {
		if pattern.MatchString(fName) {
			return true
		}
	}

	return false
}

func (w *lintAddConstantRule) checkStrLit(n *ast.BasicLit) {
	if w.allowList[kindSTRING][n.Value] {
		return
	}

	count := w.strLits[n.Value]
	if count >= 0 {
		w.strLits[n.Value] = count + 1
		if w.strLits[n.Value] > w.strLitLimit {
			w.onFailure(lint.Failure{
				Confidence: 1,
				Node:       n,
				Category:   "style",
				Failure:    fmt.Sprintf("string literal %s appears, at least, %d times, create a named constant for it", n.Value, w.strLits[n.Value]),
			})
			w.strLits[n.Value] = -1 // mark it to avoid failing again on the same literal
		}
	}
}

func (w *lintAddConstantRule) checkNumLit(kind string, n *ast.BasicLit) {
	if w.allowList[kind][n.Value] {
		return
	}

	w.onFailure(lint.Failure{
		Confidence: 1,
		Node:       n,
		Category:   "style",
		Failure:    fmt.Sprintf("avoid magic numbers like '%s', create a named constant for it", n.Value),
	})
}

func (w *lintAddConstantRule) isStructTag(n *ast.BasicLit) bool {
	_, ok := w.structTags[n]
	return ok
}

func (r *AddConstantRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()

	if r.allowList == nil {
		r.strLitLimit = defaultStrLitLimit
		r.allowList = newAllowList()
		if len(arguments) > 0 {
			args, ok := arguments[0].(map[string]any)
			if !ok {
				panic(fmt.Sprintf("Invalid argument to the add-constant rule. Expecting a k,v map, got %T", arguments[0]))
			}
			for k, v := range args {
				kind := ""
				switch k {
				case "allowFloats":
					kind = kindFLOAT
					fallthrough
				case "allowInts":
					if kind == "" {
						kind = kindINT
					}
					fallthrough
				case "allowStrs":
					if kind == "" {
						kind = kindSTRING
					}
					list, ok := v.(string)
					if !ok {
						panic(fmt.Sprintf("Invalid argument to the add-constant rule, string expected. Got '%v' (%T)", v, v))
					}
					r.allowList.add(kind, list)
				case "maxLitCount":
					sl, ok := v.(string)
					if !ok {
						panic(fmt.Sprintf("Invalid argument to the add-constant rule, expecting string representation of an integer. Got '%v' (%T)", v, v))
					}

					limit, err := strconv.Atoi(sl)
					if err != nil {
						panic(fmt.Sprintf("Invalid argument to the add-constant rule, expecting string representation of an integer. Got '%v'", v))
					}
					r.strLitLimit = limit
				case "ignoreFuncs":
					excludes, ok := v.(string)
					if !ok {
						panic(fmt.Sprintf("Invalid argument to the ignoreFuncs parameter of add-constant rule, string expected. Got '%v' (%T)", v, v))
					}

					for _, exclude := range strings.Split(excludes, ",") {
						exclude = strings.Trim(exclude, " ")
						if exclude == "" {
							panic("Invalid argument to the ignoreFuncs parameter of add-constant rule, expected regular expression must not be empty.")
						}

						exp, err := regexp.Compile(exclude)
						if err != nil {
							panic(fmt.Sprintf("Invalid argument to the ignoreFuncs parameter of add-constant rule: regexp %q does not compile: %v", exclude, err))
						}

						r.ignoreFunctions = append(r.ignoreFunctions, exp)
					}
				}
			}
		}
	}
}
