package ginkgohandler

import (
	"go/ast"
)

const (
	importPath   = `"github.com/onsi/ginkgo"`
	importPathV2 = `"github.com/onsi/ginkgo/v2"`

	focusSpec = "Focus"
)

// Handler provide different handling, depend on the way ginkgo was imported, whether
// in imported with "." name, custom name or without any name.
type Handler interface {
	GetFocusContainerName(*ast.CallExpr) (bool, *ast.Ident)
	IsWrapContainer(*ast.CallExpr) bool
	IsFocusSpec(ident ast.Expr) bool
}

// GetGinkgoHandler returns a ginkgor handler according to the way ginkgo was imported in the specific file
func GetGinkgoHandler(file *ast.File) Handler {
	for _, imp := range file.Imports {
		if imp.Path.Value != importPath && imp.Path.Value != importPathV2 {
			continue
		}

		switch name := imp.Name.String(); {
		case name == ".":
			return dotHandler{}
		case name == "<nil>": // import with no local name
			return nameHandler("ginkgo")
		default:
			return nameHandler(name)
		}
	}

	return nil // no ginkgo import; this file does not use ginkgo
}

// dotHandler is used when importing ginkgo with dot; i.e.
// import . "github.com/onsi/ginkgo"
type dotHandler struct{}

func (h dotHandler) GetFocusContainerName(exp *ast.CallExpr) (bool, *ast.Ident) {
	if fun, ok := exp.Fun.(*ast.Ident); ok {
		return isFocusContainer(fun.Name), fun
	}
	return false, nil
}

func (h dotHandler) IsWrapContainer(exp *ast.CallExpr) bool {
	if fun, ok := exp.Fun.(*ast.Ident); ok {
		return IsWrapContainer(fun.Name)
	}
	return false
}

func (h dotHandler) IsFocusSpec(exp ast.Expr) bool {
	id, ok := exp.(*ast.Ident)
	return ok && id.Name == focusSpec
}

// nameHandler is used when importing ginkgo without name; i.e.
// import "github.com/onsi/ginkgo"
//
// or with a custom name; e.g.
// import customname "github.com/onsi/ginkgo"
type nameHandler string

func (h nameHandler) GetFocusContainerName(exp *ast.CallExpr) (bool, *ast.Ident) {
	if sel, ok := exp.Fun.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == string(h) {
			return isFocusContainer(sel.Sel.Name), sel.Sel
		}
	}
	return false, nil
}

func (h nameHandler) IsWrapContainer(exp *ast.CallExpr) bool {
	if sel, ok := exp.Fun.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == string(h) {
			return IsWrapContainer(sel.Sel.Name)
		}
	}
	return false

}

func (h nameHandler) IsFocusSpec(exp ast.Expr) bool {
	if selExp, ok := exp.(*ast.SelectorExpr); ok {
		if x, ok := selExp.X.(*ast.Ident); ok && x.Name == string(h) {
			return selExp.Sel.Name == focusSpec
		}
	}

	return false
}

func isFocusContainer(name string) bool {
	switch name {
	case "FDescribe", "FContext", "FWhen", "FIt", "FDescribeTable", "FEntry":
		return true
	}
	return false
}

func IsContainer(name string) bool {
	switch name {
	case "It", "When", "Context", "Describe", "DescribeTable", "Entry",
		"PIt", "PWhen", "PContext", "PDescribe", "PDescribeTable", "PEntry",
		"XIt", "XWhen", "XContext", "XDescribe", "XDescribeTable", "XEntry":
		return true
	}
	return isFocusContainer(name)
}

func IsWrapContainer(name string) bool {
	switch name {
	case "When", "Context", "Describe",
		"FWhen", "FContext", "FDescribe",
		"PWhen", "PContext", "PDescribe",
		"XWhen", "XContext", "XDescribe":
		return true
	}

	return false
}
