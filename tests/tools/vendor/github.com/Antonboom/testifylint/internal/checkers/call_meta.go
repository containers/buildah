package checkers

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
	"github.com/Antonboom/testifylint/internal/testify"
)

// CallMeta stores meta info about assertion function/method call, for example
//
//	assert.Equal(t, 42, result, "helpful comment")
type CallMeta struct {
	// Range contains start and end position of assertion call.
	analysis.Range
	// IsPkg true if this is package (not object) call.
	IsPkg bool
	// IsAssert true if this is "testify/assert" package (or object) call.
	IsAssert bool
	// Selector is the AST expression of "assert.Equal".
	Selector *ast.SelectorExpr
	// SelectorXStr is a string representation of Selector's left part – value before point, e.g. "assert".
	SelectorXStr string
	// Fn stores meta info about assertion function itself.
	Fn FnMeta
	// Args stores assertion call arguments but without `t *testing.T` argument.
	// E.g [42, result, "helpful comment"].
	Args []ast.Expr
	// ArgsRaw stores assertion call initial arguments.
	// E.g [t, 42, result, "helpful comment"].
	ArgsRaw []ast.Expr
}

func (c CallMeta) String() string {
	return c.SelectorXStr + "." + c.Fn.Name
}

// FnMeta stores meta info about assertion function itself, for example "Equal".
type FnMeta struct {
	// Range contains start and end position of function Name.
	analysis.Range
	// Name is a function name.
	Name string
	// NameFTrimmed is a function name without "f" suffix.
	NameFTrimmed string
	// IsFmt is true if function is formatted, e.g. "Equalf".
	IsFmt bool
}

// NewCallMeta returns meta information about testify assertion call.
// Returns nil if ast.CallExpr is not testify call.
func NewCallMeta(pass *analysis.Pass, ce *ast.CallExpr) *CallMeta {
	se, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok || se.Sel == nil {
		return nil
	}
	fnName := se.Sel.Name

	initiatorPkg, isPkgCall := func() (*types.Package, bool) {
		// Examples:
		// s.Assert         -> method of *suite.Suite        -> package suite ("vendor/github.com/stretchr/testify/suite")
		// s.Assert().Equal -> method of *assert.Assertions  -> package assert ("vendor/github.com/stretchr/testify/assert")
		// s.Equal          -> method of *assert.Assertions  -> package assert ("vendor/github.com/stretchr/testify/assert")
		// reqObj.Falsef    -> method of *require.Assertions -> package require ("vendor/github.com/stretchr/testify/require")
		if sel, ok := pass.TypesInfo.Selections[se]; ok {
			return sel.Obj().Pkg(), false
		}

		// Examples:
		// assert.False      -> assert  -> package assert ("vendor/github.com/stretchr/testify/assert")
		// require.NotEqualf -> require -> package require ("vendor/github.com/stretchr/testify/require")
		if id, ok := se.X.(*ast.Ident); ok {
			if selObj := pass.TypesInfo.ObjectOf(id); selObj != nil {
				if pkg, ok := selObj.(*types.PkgName); ok {
					return pkg.Imported(), true
				}
			}
		}
		return nil, false
	}()
	if initiatorPkg == nil {
		return nil
	}

	isAssert := analysisutil.IsPkg(initiatorPkg, testify.AssertPkgName, testify.AssertPkgPath)
	isRequire := analysisutil.IsPkg(initiatorPkg, testify.RequirePkgName, testify.RequirePkgPath)
	if !(isAssert || isRequire) {
		return nil
	}

	return &CallMeta{
		Range:        ce,
		IsPkg:        isPkgCall,
		IsAssert:     isAssert,
		Selector:     se,
		SelectorXStr: analysisutil.NodeString(pass.Fset, se.X),
		Fn: FnMeta{
			Range:        se.Sel,
			Name:         fnName,
			NameFTrimmed: strings.TrimSuffix(fnName, "f"),
			IsFmt:        strings.HasSuffix(fnName, "f"),
		},
		Args:    trimTArg(pass, ce.Args),
		ArgsRaw: ce.Args,
	}
}

func trimTArg(pass *analysis.Pass, args []ast.Expr) []ast.Expr {
	if len(args) == 0 {
		return args
	}

	if implementsTestingT(pass, args[0]) {
		return args[1:]
	}
	return args
}
