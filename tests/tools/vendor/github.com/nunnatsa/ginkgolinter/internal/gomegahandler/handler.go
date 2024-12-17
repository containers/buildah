package gomegahandler

import (
	"go/ast"
	"go/token"
)

const (
	importPath = `"github.com/onsi/gomega"`
)

// Handler provide different handling, depend on the way gomega was imported, whether
// in imported with "." name, custom name or without any name.
type Handler interface {
	// GetActualFuncName returns the name of the gomega function, e.g. `Expect`
	GetActualFuncName(*ast.CallExpr) (string, bool)
	// ReplaceFunction replaces the function with another one, for fix suggestions
	ReplaceFunction(*ast.CallExpr, *ast.Ident)

	getDefFuncName(expr *ast.CallExpr) string

	getFieldType(field *ast.Field) string

	GetActualExpr(assertionFunc *ast.SelectorExpr) *ast.CallExpr
}

// GetGomegaHandler returns a gomegar handler according to the way gomega was imported in the specific file
func GetGomegaHandler(file *ast.File) Handler {
	for _, imp := range file.Imports {
		if imp.Path.Value != importPath {
			continue
		}

		switch name := imp.Name.String(); {
		case name == ".":
			return dotHandler{}
		case name == "<nil>": // import with no local name
			return nameHandler("gomega")
		default:
			return nameHandler(name)
		}
	}

	return nil // no gomega import; this file does not use gomega
}

// dotHandler is used when importing gomega with dot; i.e.
// import . "github.com/onsi/gomega"
type dotHandler struct{}

// GetActualFuncName returns the name of the gomega function, e.g. `Expect`
func (h dotHandler) GetActualFuncName(expr *ast.CallExpr) (string, bool) {
	switch actualFunc := expr.Fun.(type) {
	case *ast.Ident:
		return actualFunc.Name, true
	case *ast.SelectorExpr:
		if isGomegaVar(actualFunc.X, h) {
			return actualFunc.Sel.Name, true
		}

		if x, ok := actualFunc.X.(*ast.CallExpr); ok {
			return h.GetActualFuncName(x)
		}

	case *ast.CallExpr:
		return h.GetActualFuncName(actualFunc)
	}
	return "", false
}

// ReplaceFunction replaces the function with another one, for fix suggestions
func (dotHandler) ReplaceFunction(caller *ast.CallExpr, newExpr *ast.Ident) {
	switch f := caller.Fun.(type) {
	case *ast.Ident:
		caller.Fun = newExpr
	case *ast.SelectorExpr:
		f.Sel = newExpr
	}
}

func (dotHandler) getDefFuncName(expr *ast.CallExpr) string {
	if f, ok := expr.Fun.(*ast.Ident); ok {
		return f.Name
	}
	return ""
}

func (dotHandler) getFieldType(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if name, ok := t.X.(*ast.Ident); ok {
			return name.Name
		}
	}
	return ""
}

// nameHandler is used when importing gomega without name; i.e.
// import "github.com/onsi/gomega"
//
// or with a custom name; e.g.
// import customname "github.com/onsi/gomega"
type nameHandler string

// GetActualFuncName returns the name of the gomega function, e.g. `Expect`
func (g nameHandler) GetActualFuncName(expr *ast.CallExpr) (string, bool) {
	selector, ok := expr.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	switch x := selector.X.(type) {
	case *ast.Ident:
		if x.Name != string(g) {
			if !isGomegaVar(x, g) {
				return "", false
			}
		}

		return selector.Sel.Name, true

	case *ast.CallExpr:
		return g.GetActualFuncName(x)
	}

	return "", false
}

// ReplaceFunction replaces the function with another one, for fix suggestions
func (nameHandler) ReplaceFunction(caller *ast.CallExpr, newExpr *ast.Ident) {
	caller.Fun.(*ast.SelectorExpr).Sel = newExpr
}

func (g nameHandler) getDefFuncName(expr *ast.CallExpr) string {
	if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
		if f, ok := sel.X.(*ast.Ident); ok && f.Name == string(g) {
			return sel.Sel.Name
		}
	}
	return ""
}

func (g nameHandler) getFieldType(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			if id.Name == string(g) {
				return t.Sel.Name
			}
		}
	case *ast.StarExpr:
		if sel, ok := t.X.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == string(g) {
				return sel.Sel.Name
			}
		}

	}
	return ""
}

func isGomegaVar(x ast.Expr, handler Handler) bool {
	if i, ok := x.(*ast.Ident); ok {
		if i.Obj != nil && i.Obj.Kind == ast.Var {
			switch decl := i.Obj.Decl.(type) {
			case *ast.AssignStmt:
				if decl.Tok == token.DEFINE {
					if defFunc, ok := decl.Rhs[0].(*ast.CallExpr); ok {
						fName := handler.getDefFuncName(defFunc)
						switch fName {
						case "NewGomega", "NewWithT", "NewGomegaWithT":
							return true
						}
					}
				}
			case *ast.Field:
				name := handler.getFieldType(decl)
				switch name {
				case "Gomega", "WithT", "GomegaWithT":
					return true
				}
			}
		}
	}
	return false
}

func (h dotHandler) GetActualExpr(assertionFunc *ast.SelectorExpr) *ast.CallExpr {
	actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch fun := actualExpr.Fun.(type) {
	case *ast.Ident:
		return actualExpr
	case *ast.SelectorExpr:
		if isHelperMethods(fun.Sel.Name) {
			return h.GetActualExpr(fun)
		}
		if isGomegaVar(fun.X, h) {
			return actualExpr
		}
	}
	return nil
}

func (g nameHandler) GetActualExpr(assertionFunc *ast.SelectorExpr) *ast.CallExpr {
	actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch fun := actualExpr.Fun.(type) {
	case *ast.Ident:
		return actualExpr
	case *ast.SelectorExpr:
		if x, ok := fun.X.(*ast.Ident); ok && x.Name == string(g) {
			return actualExpr
		}
		if isHelperMethods(fun.Sel.Name) {
			return g.GetActualExpr(fun)
		}

		if isGomegaVar(fun.X, g) {
			return actualExpr
		}
	}
	return nil
}

func isHelperMethods(funcName string) bool {
	switch funcName {
	case "WithOffset", "WithTimeout", "WithPolling", "Within", "ProbeEvery", "WithContext", "WithArguments", "MustPassRepeatedly":
		return true
	}

	return false
}
