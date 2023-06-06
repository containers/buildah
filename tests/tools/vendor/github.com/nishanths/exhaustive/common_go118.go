//go:build go1.18
// +build go1.18

package exhaustive

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func fromNamed(pass *analysis.Pass, t *types.Named, typeparam bool) (result []typeAndMembers, ok bool) {
	if tpkg := t.Obj().Pkg(); tpkg == nil {
		// go/types documentation says: nil for labels and
		// objects in the Universe scope. This happens for the built-in
		// error type for example.
		return nil, false // not a valid enum type, so ok == false
	}

	et := enumType{t.Obj()}
	if em, ok := importFact(pass, et); ok {
		return []typeAndMembers{{et, em}}, true
	}

	if typeparam {
		// is it a named interface?
		if intf, ok := t.Underlying().(*types.Interface); ok {
			return fromInterface(pass, intf, typeparam)
		}
	}

	return nil, false // not a valid enum type, so ok == false
}

func fromInterface(pass *analysis.Pass, intf *types.Interface, typeparam bool) (result []typeAndMembers, ok bool) {
	allOk := true
	for i := 0; i < intf.NumEmbeddeds(); i++ {
		r, ok := fromType(pass, intf.EmbeddedType(i), typeparam)
		result = append(result, r...)
		allOk = allOk && ok
	}
	return result, allOk
}

func fromUnion(pass *analysis.Pass, union *types.Union, typeparam bool) (result []typeAndMembers, ok bool) {
	allOk := true
	// gather from each term in the union.
	for i := 0; i < union.Len(); i++ {
		r, ok := fromType(pass, union.Term(i).Type(), typeparam)
		result = append(result, r...)
		allOk = allOk && ok
	}
	return result, allOk
}

func fromTypeParam(pass *analysis.Pass, tp *types.TypeParam, typeparam bool) (result []typeAndMembers, ok bool) {
	// Does not appear to be explicitly documented, but based on Go language
	// spec (see section Type constraints) and Go standard library source code,
	// we can expect constraints to have underlying type *types.Interface
	// Regardless it will be handled in fromType.
	return fromType(pass, tp.Constraint().Underlying(), typeparam)
}

func fromType(pass *analysis.Pass, t types.Type, typeparam bool) (result []typeAndMembers, ok bool) {
	switch t := t.(type) {
	case *types.Named:
		return fromNamed(pass, t, typeparam)

	case *types.Union:
		return fromUnion(pass, t, typeparam)

	case *types.TypeParam:
		return fromTypeParam(pass, t, typeparam)

	case *types.Interface:
		if !typeparam {
			return nil, true
		}
		// anonymous interface.
		// e.g. func foo[T interface { M } | interface { N }](v T) {}
		return fromInterface(pass, t, typeparam)

	default:
		// ignore these.
		return nil, true
	}
}

func composingEnumTypes(pass *analysis.Pass, t types.Type) (result []typeAndMembers, ok bool) {
	_, typeparam := t.(*types.TypeParam)
	result, ok = fromType(pass, t, typeparam)

	if typeparam {
		var kind types.BasicKind
		var kindSet bool

		// sameBasicKind reports whether each type t that the function is called
		// with has the same underlying basic kind.
		sameBasicKind := func(t types.Type) (ok bool) {
			basic, ok := t.Underlying().(*types.Basic)
			if !ok {
				return false
			}
			if kindSet && kind != basic.Kind() {
				return false
			}
			kind = basic.Kind()
			kindSet = true
			return true
		}

		for _, rr := range result {
			if !sameBasicKind(rr.et.TypeName.Type()) {
				ok = false
				break
			}
		}
	}

	return result, ok
}
