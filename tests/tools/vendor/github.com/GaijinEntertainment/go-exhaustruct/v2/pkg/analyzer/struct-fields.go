package analyzer

import (
	"go/types"
	"reflect"

	"golang.org/x/exp/slices"
)

type StructFields struct {
	All         []string
	AllRequired []string

	Public         []string
	PublicRequired []string
}

func NewStructFields(strct *types.Struct) *StructFields {
	sf := StructFields{
		All:            make([]string, strct.NumFields()),
		AllRequired:    make([]string, 0, strct.NumFields()),
		Public:         make([]string, 0, strct.NumFields()),
		PublicRequired: make([]string, 0, strct.NumFields()),
	}

	for i := 0; i < strct.NumFields(); i++ {
		f := strct.Field(i)
		isOptional := isFieldOptional(strct.Tag(i))

		sf.All[i] = f.Name()
		if !isOptional {
			sf.AllRequired = append(sf.AllRequired, f.Name())
		}

		if f.Exported() {
			sf.Public = append(sf.Public, f.Name())

			if !isOptional {
				sf.PublicRequired = append(sf.PublicRequired, f.Name())
			}
		}
	}

	sf.All = slices.Clip(sf.All)
	sf.AllRequired = slices.Clip(sf.AllRequired)
	sf.Public = slices.Clip(sf.Public)
	sf.PublicRequired = slices.Clip(sf.PublicRequired)

	return &sf
}

const (
	TagName          = "exhaustruct"
	OptionalTagValue = "optional"
)

// isFieldOptional checks if field tags has an optional tag, and therefore can
// be omitted during structure initialization.
func isFieldOptional(tags string) bool {
	return reflect.StructTag(tags).Get(TagName) == OptionalTagValue
}
