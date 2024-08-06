package mirror

import "github.com/butuzov/mirror/internal/checker"

var MaphashMethods = []checker.Violation{
	{ // (*hash/maphash).Write
		Targets:   checker.Bytes,
		Type:      checker.Method,
		Package:   "hash/maphash",
		Struct:    "Hash",
		Caller:    "Write",
		Args:      []int{0},
		AltCaller: "WriteString",

		Generate: &checker.Generate{
			PreCondition: `h := maphash.Hash{}`,
			Pattern:      `Write($0)`,
			Returns:      []string{"int", "error"},
		},
	},
	{ // (*hash/maphash).WriteString
		Targets:   checker.Strings,
		Type:      checker.Method,
		Package:   "hash/maphash",
		Struct:    "Hash",
		Caller:    "WriteString",
		Args:      []int{0},
		AltCaller: "Write",

		Generate: &checker.Generate{
			PreCondition: `h := maphash.Hash{}`,
			Pattern:      `WriteString($0)`,
			Returns:      []string{"int", "error"},
		},
	},
}
