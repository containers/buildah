package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	name = "nilnil"
	doc  = "Checks that there is no simultaneous return of `nil` error and an invalid value."

	reportMsg = "return both the `nil` error and invalid value: use a sentinel error instead"
)

// New returns new nilnil analyzer.
func New() *analysis.Analyzer {
	n := newNilNil()

	a := &analysis.Analyzer{
		Name:     name,
		Doc:      doc,
		Run:      n.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
	a.Flags.Var(&n.checkedTypes, "checked-types", "coma separated list")

	return a
}

type nilNil struct {
	checkedTypes checkedTypes
}

func newNilNil() *nilNil {
	return &nilNil{
		checkedTypes: newDefaultCheckedTypes(),
	}
}

var funcAndReturns = []ast.Node{
	(*ast.FuncDecl)(nil),
	(*ast.FuncLit)(nil),
	(*ast.ReturnStmt)(nil),
}

func (n *nilNil) run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var fs funcTypeStack
	insp.Nodes(funcAndReturns, func(node ast.Node, push bool) (proceed bool) {
		switch v := node.(type) {
		case *ast.FuncLit:
			if push {
				fs.Push(v.Type)
			} else {
				fs.Pop()
			}

		case *ast.FuncDecl:
			if push {
				fs.Push(v.Type)
			} else {
				fs.Pop()
			}

		case *ast.ReturnStmt:
			ft := fs.Top() // Current function.

			if !push || len(v.Results) != 2 || ft == nil || ft.Results == nil || len(ft.Results.List) != 2 {
				return false
			}

			fRes1Type := pass.TypesInfo.TypeOf(ft.Results.List[0].Type)
			if fRes1Type == nil {
				return false
			}

			fRes2Type := pass.TypesInfo.TypeOf(ft.Results.List[1].Type)
			if fRes2Type == nil {
				return false
			}

			ok, zv := n.isDangerNilType(fRes1Type)
			if !(ok && isErrorType(fRes2Type)) {
				return false
			}

			retVal, retErr := v.Results[0], v.Results[1]

			var needWarn bool
			switch zv {
			case zeroValueNil:
				needWarn = isNil(pass, retVal) && isNil(pass, retErr)
			case zeroValueZero:
				needWarn = isZero(retVal) && isNil(pass, retErr)
			}

			if needWarn {
				pass.Reportf(v.Pos(), reportMsg)
			}
		}

		return true
	})

	return nil, nil //nolint:nilnil
}

type zeroValue int

const (
	zeroValueNil = iota + 1
	zeroValueZero
)

func (n *nilNil) isDangerNilType(t types.Type) (bool, zeroValue) {
	switch v := t.(type) {
	case *types.Pointer:
		return n.checkedTypes.Contains(ptrType), zeroValueNil

	case *types.Signature:
		return n.checkedTypes.Contains(funcType), zeroValueNil

	case *types.Interface:
		return n.checkedTypes.Contains(ifaceType), zeroValueNil

	case *types.Map:
		return n.checkedTypes.Contains(mapType), zeroValueNil

	case *types.Chan:
		return n.checkedTypes.Contains(chanType), zeroValueNil

	case *types.Basic:
		if v.Kind() == types.Uintptr {
			return n.checkedTypes.Contains(uintptrType), zeroValueZero
		}
		if v.Kind() == types.UnsafePointer {
			return n.checkedTypes.Contains(unsafeptrType), zeroValueNil
		}

	case *types.Named:
		return n.isDangerNilType(v.Underlying())
	}
	return false, 0
}

var errorIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func isErrorType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Interface)
	return ok && types.Implements(t, errorIface)
}

func isNil(pass *analysis.Pass, e ast.Expr) bool {
	i, ok := e.(*ast.Ident)
	if !ok {
		return false
	}

	_, ok = pass.TypesInfo.ObjectOf(i).(*types.Nil)
	return ok
}

func isZero(e ast.Expr) bool {
	bl, ok := e.(*ast.BasicLit)
	if !ok {
		return false
	}
	if bl.Kind != token.INT {
		return false
	}

	v, err := strconv.ParseInt(bl.Value, 0, 64)
	if err != nil {
		return false
	}
	return v == 0
}
