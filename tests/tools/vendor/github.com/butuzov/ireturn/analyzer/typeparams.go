package analyzer

import (
	"go/ast"
)

type typeParams struct {
	found []string
}

func newTypeParams(fl *ast.FieldList) typeParams {
	tp := typeParams{}

	if fl == nil {
		return tp
	}

	for _, el := range fl.List {
		if el == nil {
			continue
		}

		for _, name := range el.Names {
			tp.found = append(tp.found, name.Name)
		}
	}

	return tp
}

func (tp typeParams) In(t string) bool {
	for _, i := range tp.found {
		if i == t {
			return true
		}
	}
	return false
}
