/*
Package exhaustive defines an analyzer that checks exhaustiveness of switch
statements of enum-like constants in Go source code. The analyzer can be
configured to additionally check exhaustiveness of map literals whose key type
is enum-like.

# Definition of enum

The Go [language spec] does not provide an explicit definition for enums. For
the purpose of this analyzer, and by convention, an enum type is any named
type that has:

  - underlying type float, string, or integer (includes byte and
    rune, which are aliases for uint8 and int32, respectively); and
  - at least one constant of the type defined in the same scope.

In the example below, Biome is an enum type. The three constants are its
enum members.

	package eco

	type Biome int

	const (
		Tundra  Biome = 1
		Savanna Biome = 2
		Desert  Biome = 3
	)

Enum member constants for a particular enum type do not necessarily all
have to be declared in the same const block. The constant values may be
specified using iota, using literal values, or using any valid means for
declaring a Go constant. It is allowed for multiple enum member
constants for a particular enum type to have the same constant value.

# Definition of exhaustiveness

A switch statement that switches on a value of an enum type is exhaustive if
all enum members, by constant value, are listed in the switch
statement's cases. If multiple members have the same constant value, it is
sufficient for any one of these same-valued members to be listed.

For an enum type defined in the same package as the switch statement, both
exported and unexported enum members must be listed to satisfy exhaustiveness.
For an enum type defined in an external package, it is sufficient that only
exported enum members are listed. In a switch statement's cases, only
identifiers (e.g. Tundra) and qualified identifiers (e.g. somepkg.Grassland)
that name constants may contribute towards satisfying exhaustiveness; other
expressions such as literal values and function calls will not.

By default, the existence of a default case in a switch statement does not
unconditionally make a switch statement exhaustive. Use the
-default-signifies-exhaustive flag to adjust this behavior.

A similar definition of exhaustiveness applies to a map literal whose key type
is an enum type. For the map literal to be considered exhaustive, all enum
members, by constant value, must be listed as keys. Empty map literals are not
checked. For the analyzer to check map literals, the -check flag must include
the value "map".

# Type parameters

A switch statement that switches on a value whose type is a type parameter is
checked for exhaustiveness if each type element in the type constraint is an
enum type and shares the same underlying basic type kind.

In the following example, the switch statement on the value of type parameter
T will be checked, because each type element of T—namely M, N, and O—is an
enum type and shares the same underlying basic type kind (i.e. int8). To
satisfy exhaustiveness, all enum members, by constant value, for each of the
enum types M, N, and O—namely A, B, C, and D—must be listed in the switch
statement's cases.

	func bar[T M | I](v T) {
		switch v {
			case T(A):
			case T(B):
			case T(C):
			case T(D):
		}
	}

	type I interface{ N | J }
	type J interface{ O }

	type M int8
	const A M = 1

	type N int8
	const B N = 2
	const C N = 3

	type O int8
	const D O = 4

# Type aliases

The analyzer handles type aliases as shown in the example below. Here T2 is a
enum type. T1 is an alias for T2. Note that T1 itself isn't considered an enum
type; T1 is only an alias for an enum type.

	package pkg
	type T1 = newpkg.T2
	const (
		A = newpkg.A
		B = newpkg.B
	)

	package newpkg
	type T2 int
	const (
		A T2 = 1
		B T2 = 2
	)

A switch statement that switches on a value of type T1 (which, in reality, is
just an alternate spelling for type T2) is exhaustive if all of T2's enum
members, by constant value, are listed in the switch statement's cases.
(Recall that only constants declared in the same scope as type T2's scope can
be T2's enum members.)

The following switch statements are exhaustive.

	// Note: the type of v is effectively newpkg.T2, due to type aliasing.
	func f(v pkg.T1) {
		switch v {
		case newpkg.A:
		case newpkg.B:
		}
	}

	func g(v pkg.T1) {
		switch v {
		case pkg.A:
		case pkg.B:
		}
	}

The analyzer guarantees that introducing a type alias (such as type T1 =
newpkg.T2) will not result in new diagnostics from the analyzer, as long as
the set of enum member constant values of the alias RHS type is a subset of
the set of enum member constant values of the LHS type.

# Flags

Summary:

	flag                           type                     default value
	----                           ----                     -------------
	-check                         comma-separated string   switch
	-explicit-exhaustive-switch    bool                     false
	-explicit-exhaustive-map       bool                     false
	-check-generated               bool                     false
	-default-signifies-exhaustive  bool                     false
	-ignore-enum-members           regexp pattern           (none)
	-ignore-enum-types             regexp pattern           (none)
	-package-scope-only            bool                     false

Flag descriptions:

  - The -check flag specifies a comma-separated list of program elements
    that should be checked for exhaustiveness; supported program elements
    are "switch" and "map". The default flag value is "switch", which means
    that only switch statements are checked. Specify the flag value
    "switch,map" to check both switch statements and map literals.

  - If -explicit-exhaustive-switch is enabled, the analyzer checks a switch
    statement only if it is associated with a comment beginning with
    "//exhaustive:enforce". Otherwise, the analyzer checks every enum switch
    statement not associated with a comment beginning with
    "//exhaustive:ignore".

  - The -explicit-exhaustive-map flag is the map literal counterpart for the
    -explicit-exhaustive-switch flag.

  - If -check-generated is enabled, switch statements and map literals in
    generated Go source files are checked. By default, the analyzer does not
    check generated files. Refer to https://golang.org/s/generatedcode for
    the definition of generated files.

  - If -default-signifies-exhaustive is enabled, the presence of a default
    case in a switch statement unconditionally satisfies exhaustiveness (all
    enum members do not have to be listed). Enabling this flag usually tends
    to counter the purpose of exhaustiveness checking, so it is not
    recommended that you enable this flag.

  - The -ignore-enum-members flag specifies a regular expression in Go
    package regexp syntax. Constants matching the regular expression do not
    have to be listed in switch statement cases or map literals in order to
    satisfy exhaustiveness. The specified regular expression is matched
    against the constant name inclusive of the enum package import path. For
    example, if the package import path of the constant is "example.org/eco"
    and the constant name is "Tundra", the specified regular expression will
    be matched against the string "example.org/eco.Tundra".

  - The -ignore-enum-types flag is similar to the -ignore-enum-members flag,
    except that it applies to types.

  - If -package-scope-only is enabled, the analyzer only finds enums defined
    in package scope but not in inner scopes such as functions; consequently
    only switch statements and map literals that use such enums are checked
    for exhaustiveness. By default, the analyzer finds enums defined in all
    scopes, including in inner scopes such as functions.

# Skip analysis

To skip analysis of a switch statement or a map literal, associate it with a
comment that begins with "//exhaustive:ignore". For example:

	//exhaustive:ignore
	switch v {
	case A:
	case B:
	}

To ignore specific constants in exhaustiveness checks, use the
-ignore-enum-members flag:

	exhaustive -ignore-enum-members '^example\.org/eco\.Tundra$'

To ignore specific types, use the -ignore-enum-types flag:

	exhaustive -ignore-enum-types '^time\.Duration$|^example\.org/measure\.Unit$'

[language spec]: https://golang.org/ref/spec
*/
package exhaustive

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

func init() {
	Analyzer.Flags.Var(&fCheck, CheckFlag, "comma-separated list of program `elements` that should be checked for exhaustiveness; supported elements are: switch, map")
	Analyzer.Flags.BoolVar(&fExplicitExhaustiveSwitch, ExplicitExhaustiveSwitchFlag, false, `check switch statement only if associated with "//exhaustive:enforce" comment`)
	Analyzer.Flags.BoolVar(&fExplicitExhaustiveMap, ExplicitExhaustiveMapFlag, false, `check map literal only if associated with "//exhaustive:enforce" comment`)
	Analyzer.Flags.BoolVar(&fCheckGenerated, CheckGeneratedFlag, false, "check generated files")
	Analyzer.Flags.BoolVar(&fDefaultSignifiesExhaustive, DefaultSignifiesExhaustiveFlag, false, "presence of default case in switch statement unconditionally satisfies exhaustiveness")
	Analyzer.Flags.Var(&fIgnoreEnumMembers, IgnoreEnumMembersFlag, "constants matching `regexp` are ignored for exhaustiveness checks")
	Analyzer.Flags.Var(&fIgnoreEnumTypes, IgnoreEnumTypesFlag, "types matching `regexp` are ignored for exhaustiveness checks")
	Analyzer.Flags.BoolVar(&fPackageScopeOnly, PackageScopeOnlyFlag, false, "find enums only in package scopes, not inner scopes")

	var unused string
	Analyzer.Flags.StringVar(&unused, IgnorePatternFlag, "", "no effect (deprecated); use -"+IgnoreEnumMembersFlag)
	Analyzer.Flags.StringVar(&unused, CheckingStrategyFlag, "", "no effect (deprecated)")
}

// Flag names used by the analyzer. They are exported for use by analyzer
// driver programs.
const (
	CheckFlag                      = "check"
	ExplicitExhaustiveSwitchFlag   = "explicit-exhaustive-switch"
	ExplicitExhaustiveMapFlag      = "explicit-exhaustive-map"
	CheckGeneratedFlag             = "check-generated"
	DefaultSignifiesExhaustiveFlag = "default-signifies-exhaustive"
	IgnoreEnumMembersFlag          = "ignore-enum-members"
	IgnoreEnumTypesFlag            = "ignore-enum-types"
	PackageScopeOnlyFlag           = "package-scope-only"

	IgnorePatternFlag    = "ignore-pattern"    // Deprecated: use IgnoreEnumMembersFlag.
	CheckingStrategyFlag = "checking-strategy" // Deprecated.
)

// checkElement is a program element supported by the -check flag.
type checkElement string

const (
	elementSwitch checkElement = "switch"
	elementMap    checkElement = "map"
)

func validCheckElement(s string) error {
	switch checkElement(s) {
	case elementSwitch:
		return nil
	case elementMap:
		return nil
	default:
		return fmt.Errorf("invalid program element %q", s)
	}
}

var defaultCheckElements = []string{
	string(elementSwitch),
}

// Flag values.
var (
	fCheck                      = stringsFlag{elements: defaultCheckElements, filter: validCheckElement}
	fExplicitExhaustiveSwitch   bool
	fExplicitExhaustiveMap      bool
	fCheckGenerated             bool
	fDefaultSignifiesExhaustive bool
	fIgnoreEnumMembers          regexpFlag
	fIgnoreEnumTypes            regexpFlag
	fPackageScopeOnly           bool
)

// resetFlags resets the flag variables to their default values.
// Useful in tests.
func resetFlags() {
	fCheck = stringsFlag{elements: defaultCheckElements, filter: validCheckElement}
	fExplicitExhaustiveSwitch = false
	fExplicitExhaustiveMap = false
	fCheckGenerated = false
	fDefaultSignifiesExhaustive = false
	fIgnoreEnumMembers = regexpFlag{}
	fIgnoreEnumTypes = regexpFlag{}
	fPackageScopeOnly = false
}

var Analyzer = &analysis.Analyzer{
	Name:      "exhaustive",
	Doc:       "check exhaustiveness of enum switch statements",
	Run:       run,
	Requires:  []*analysis.Analyzer{inspect.Analyzer},
	FactTypes: []analysis.Fact{&enumMembersFact{}},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	for typ, members := range findEnums(
		fPackageScopeOnly,
		pass.Pkg,
		inspect,
		pass.TypesInfo,
	) {
		exportFact(pass, typ, members)
	}

	generated := boolCache{value: isGeneratedFile}
	comments := commentCache{value: fileCommentMap}
	swConf := switchConfig{
		explicit:                   fExplicitExhaustiveSwitch,
		defaultSignifiesExhaustive: fDefaultSignifiesExhaustive,
		checkGenerated:             fCheckGenerated,
		ignoreConstant:             fIgnoreEnumMembers.re,
		ignoreType:                 fIgnoreEnumTypes.re,
	}
	mapConf := mapConfig{
		explicit:       fExplicitExhaustiveMap,
		checkGenerated: fCheckGenerated,
		ignoreConstant: fIgnoreEnumMembers.re,
		ignoreType:     fIgnoreEnumTypes.re,
	}
	swChecker := switchChecker(pass, swConf, generated, comments)
	mapChecker := mapChecker(pass, mapConf, generated, comments)

	// NOTE: should not share the same inspect.WithStack call for different
	// program elements: the visitor function for a program element may
	// exit traversal early, but this shouldn't affect traversal for
	// other program elements.
	for _, e := range fCheck.elements {
		switch checkElement(e) {
		case elementSwitch:
			inspect.WithStack([]ast.Node{&ast.SwitchStmt{}}, toVisitor(swChecker))
		case elementMap:
			inspect.WithStack([]ast.Node{&ast.CompositeLit{}}, toVisitor(mapChecker))
		default:
			panic(fmt.Sprintf("unknown checkElement %v", e))
		}
	}
	return nil, nil
}
