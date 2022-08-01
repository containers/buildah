package analyzer

import (
	"go/types"
)

type StructFields struct {
	Public []string

	All []string
}

func NewStructFields(strct *types.Struct) *StructFields {
	sf := StructFields{} //nolint:exhaustruct

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)

		sf.All = append(sf.All, f.Name())

		if f.Exported() {
			sf.Public = append(sf.Public, f.Name())
		}
	}

	return &sf
}
