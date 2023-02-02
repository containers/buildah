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
		if intf, ok := t.Underlying().(*types.Interface); ok {
			return fromInterface(pass, intf, typeparam)
		}
	}

	return nil, false // not a valid enum type, so ok == false
}

func fromInterface(pass *analysis.Pass, intf *types.Interface, typeparam bool) (result []typeAndMembers, all bool) {
	all = true

	for i := 0; i < intf.NumEmbeddeds(); i++ {
		embed := intf.EmbeddedType(i)

		switch embed.(type) {
		case *types.Union:
			u := embed.(*types.Union)
			// gather from each term in the union.
			for i := 0; i < u.Len(); i++ {
				r, a := fromType(pass, u.Term(i).Type(), typeparam)
				result = append(result, r...)
				all = all && a
			}

		case *types.Named:
			r, a := fromNamed(pass, embed.(*types.Named), typeparam)
			result = append(result, r...)
			all = all && a

		default:
			// don't care about these.
			// e.g. basic type
		}
	}

	return
}

func fromType(pass *analysis.Pass, t types.Type, typeparam bool) (result []typeAndMembers, ok bool) {
	switch t := t.(type) {
	case *types.Named:
		return fromNamed(pass, t, typeparam)

	case *types.TypeParam:
		// does not appear to be explicitly documented, but based on
		// spec (see section Type constraints) and source code, we can
		// expect constraints to have underlying type *types.Interface.
		intf := t.Constraint().Underlying().(*types.Interface)
		return fromInterface(pass, intf, typeparam)

	case *types.Interface:
		// anonymous interface.
		// e.g. func foo[T interface { M } | interface { N }](v T) {}
		if !typeparam {
			return nil, true
		}
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

		// sameKind reports whether each type t that the function is called
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
