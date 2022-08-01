package analyzer

import (
	"go/types"
)

type StructFields struct {
	All    []string
	Public []string
}

func NewStructFields(strct *types.Struct) *StructFields {
	sf := StructFields{
		All:    make([]string, strct.NumFields()),
		Public: []string{},
	}

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)

		sf.All[i] = f.Name()

		if f.Exported() {
			sf.Public = append(sf.Public, f.Name())
		}
	}

	return &sf
}
