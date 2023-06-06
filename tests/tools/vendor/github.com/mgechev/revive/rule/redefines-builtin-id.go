package rule

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mgechev/revive/lint"
)

var builtInConstAndVars = map[string]bool{
	"true":  true,
	"false": true,
	"iota":  true,
	"nil":   true,
}

var builtFunctions = map[string]bool{
	"append":  true,
	"cap":     true,
	"close":   true,
	"complex": true,
	"copy":    true,
	"delete":  true,
	"imag":    true,
	"len":     true,
	"make":    true,
	"new":     true,
	"panic":   true,
	"print":   true,
	"println": true,
	"real":    true,
	"recover": true,
}

var builtInTypes = map[string]bool{
	"bool":       true,
	"byte":       true,
	"complex128": true,
	"complex64":  true,
	"error":      true,
	"float32":    true,
	"float64":    true,
	"int":        true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"int8":       true,
	"rune":       true,
	"string":     true,
	"uint":       true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uint8":      true,
	"uintptr":    true,
	"any":        true,
}

// RedefinesBuiltinIDRule warns when a builtin identifier is shadowed.
type RedefinesBuiltinIDRule struct{}

// Apply applies the rule to given file.
func (*RedefinesBuiltinIDRule) Apply(file *lint.File, _ lint.Arguments) []lint.Failure {
	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	astFile := file.AST
	w := &lintRedefinesBuiltinID{onFailure}
	ast.Walk(w, astFile)

	return failures
}

// Name returns the rule name.
func (*RedefinesBuiltinIDRule) Name() string {
	return "redefines-builtin-id"
}

type lintRedefinesBuiltinID struct {
	onFailure func(lint.Failure)
}

func (w *lintRedefinesBuiltinID) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.GenDecl:
		switch n.Tok {
		case token.TYPE:
			if len(n.Specs) < 1 {
				return nil
			}
			typeSpec, ok := n.Specs[0].(*ast.TypeSpec)
			if !ok {
				return nil
			}
			id := typeSpec.Name.Name
			if ok, bt := w.isBuiltIn(id); ok {
				w.addFailure(n, fmt.Sprintf("redefinition of the built-in %s %s", bt, id))
			}
		case token.VAR, token.CONST:
			for _, vs := range n.Specs {
				valSpec, ok := vs.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range valSpec.Names {
					if ok, bt := w.isBuiltIn(name.Name); ok {
						w.addFailure(n, fmt.Sprintf("redefinition of the built-in %s %s", bt, name))
					}
				}
			}
		default:
			return nil // skip if not type/var/const declaration
		}

	case *ast.FuncDecl:
		if n.Recv != nil {
			return w // skip methods
		}

		id := n.Name.Name
		if ok, bt := w.isBuiltIn(id); ok {
			w.addFailure(n, fmt.Sprintf("redefinition of the built-in %s %s", bt, id))
		}
	case *ast.AssignStmt:
		for _, e := range n.Lhs {
			id, ok := e.(*ast.Ident)
			if !ok {
				continue
			}

			if ok, bt := w.isBuiltIn(id.Name); ok {
				var msg string
				switch bt {
				case "constant or variable":
					if n.Tok == token.DEFINE {
						msg = fmt.Sprintf("assignment creates a shadow of built-in identifier %s", id.Name)
					} else {
						msg = fmt.Sprintf("assignment modifies built-in identifier %s", id.Name)
					}
				default:
					msg = fmt.Sprintf("redefinition of the built-in %s %s", bt, id)
				}

				w.addFailure(n, msg)
			}
		}
	}

	return w
}

func (w lintRedefinesBuiltinID) addFailure(node ast.Node, msg string) {
	w.onFailure(lint.Failure{
		Confidence: 1,
		Node:       node,
		Category:   "logic",
		Failure:    msg,
	})
}

func (lintRedefinesBuiltinID) isBuiltIn(id string) (r bool, builtInKind string) {
	if builtFunctions[id] {
		return true, "function"
	}

	if builtInConstAndVars[id] {
		return true, "constant or variable"
	}

	if builtInTypes[id] {
		return true, "type"
	}

	return false, ""
}
