package varnamelen

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// varNameLen is an analyzer that checks that the length of a variable's name matches its usage scope.
// It will create a report for a variable's assignment if that variable has a short name, but its
// usage scope is not considered "small."
type varNameLen struct {
	// maxDistance is the longest distance, in source lines, that is being considered a "small scope."
	maxDistance int

	// minNameLength is the minimum length of a variable's name that is considered "long."
	minNameLength int

	// ignoreNames is an optional list of variable names that should be ignored completely.
	ignoreNames stringsValue

	// checkReceiver determines whether a method receiver's name should be checked.
	checkReceiver bool

	// checkReturn determines whether named return values should be checked.
	checkReturn bool
}

// stringsValue is the value of a list-of-strings flag.
type stringsValue struct {
	Values []string
}

// variable represents a declared variable.
type variable struct {
	// name is the name of the variable.
	name string

	// assign is the assign statement that declares the variable.
	assign *ast.AssignStmt
}

// parameter represents a declared function or method parameter.
type parameter struct {
	// name is the name of the parameter.
	name string

	// field is the declaration of the parameter.
	field *ast.Field
}

const (
	// defaultMaxDistance is the default value for the maximum distance between the declaration of a variable and its usage
	// that is considered a "small scope."
	defaultMaxDistance = 5

	// defaultMinNameLength is the default value for the minimum length of a variable's name that is considered "long."
	defaultMinNameLength = 3
)

// NewAnalyzer returns a new analyzer that checks variable name length.
func NewAnalyzer() *analysis.Analyzer {
	vnl := varNameLen{
		maxDistance:   defaultMaxDistance,
		minNameLength: defaultMinNameLength,
		ignoreNames:   stringsValue{},
	}

	analyzer := analysis.Analyzer{
		Name: "varnamelen",
		Doc: "checks that the length of a variable's name matches its scope\n\n" +
			"A variable with a short name can be hard to use if the variable is used\n" +
			"over a longer span of lines of code. A longer variable name may be easier\n" +
			"to comprehend.",

		Run: func(pass *analysis.Pass) (interface{}, error) {
			vnl.run(pass)
			return nil, nil
		},

		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
		},
	}

	analyzer.Flags.IntVar(&vnl.maxDistance, "maxDistance", defaultMaxDistance, "maximum number of lines of variable usage scope considered 'short'")
	analyzer.Flags.IntVar(&vnl.minNameLength, "minNameLength", defaultMinNameLength, "minimum length of variable name considered 'long'")
	analyzer.Flags.Var(&vnl.ignoreNames, "ignoreNames", "comma-separated list of ignored variable names")
	analyzer.Flags.BoolVar(&vnl.checkReceiver, "checkReceiver", false, "check method receiver names")
	analyzer.Flags.BoolVar(&vnl.checkReturn, "checkReturn", false, "check named return values")

	return &analyzer
}

// Run applies v to a package, according to pass.
func (v *varNameLen) run(pass *analysis.Pass) {
	varToDist, paramToDist, returnToDist := v.distances(pass)

	for variable, dist := range varToDist {
		if v.checkNameAndDistance(variable.name, dist) {
			continue
		}
		pass.Reportf(variable.assign.Pos(), "variable name '%s' is too short for the scope of its usage", variable.name)
	}

	for param, dist := range paramToDist {
		if param.isConventional() {
			continue
		}
		if v.checkNameAndDistance(param.name, dist) {
			continue
		}
		pass.Reportf(param.field.Pos(), "parameter name '%s' is too short for the scope of its usage", param.name)
	}

	for param, dist := range returnToDist {
		if v.checkNameAndDistance(param.name, dist) {
			continue
		}
		pass.Reportf(param.field.Pos(), "return value name '%s' is too short for the scope of its usage", param.name)
	}
}

// checkNameAndDistance returns true when name or dist are considered "short", or when name is to be ignored.
func (v *varNameLen) checkNameAndDistance(name string, dist int) bool {
	if len(name) >= v.minNameLength {
		return true
	}
	if dist <= v.maxDistance {
		return true
	}
	if v.ignoreNames.contains(name) {
		return true
	}
	return false
}

// distances maps of variables or parameters and their longest usage distances.
func (v *varNameLen) distances(pass *analysis.Pass) (map[variable]int, map[parameter]int, map[parameter]int) {
	assignIdents, paramIdents, returnIdents := v.idents(pass)

	varToDist := map[variable]int{}

	for _, ident := range assignIdents {
		assign := ident.Obj.Decl.(*ast.AssignStmt)
		variable := variable{
			name:   ident.Name,
			assign: assign,
		}

		useLine := pass.Fset.Position(ident.NamePos).Line
		declLine := pass.Fset.Position(assign.Pos()).Line
		varToDist[variable] = useLine - declLine
	}

	paramToDist := map[parameter]int{}

	for _, ident := range paramIdents {
		field := ident.Obj.Decl.(*ast.Field)
		param := parameter{
			name:  ident.Name,
			field: field,
		}

		useLine := pass.Fset.Position(ident.NamePos).Line
		declLine := pass.Fset.Position(field.Pos()).Line
		paramToDist[param] = useLine - declLine
	}

	returnToDist := map[parameter]int{}

	for _, ident := range returnIdents {
		field := ident.Obj.Decl.(*ast.Field)
		param := parameter{
			name:  ident.Name,
			field: field,
		}

		useLine := pass.Fset.Position(ident.NamePos).Line
		declLine := pass.Fset.Position(field.Pos()).Line
		returnToDist[param] = useLine - declLine
	}

	return varToDist, paramToDist, returnToDist
}

// idents returns Idents referencing assign statements, parameters, and return values, respectively.
func (v *varNameLen) idents(pass *analysis.Pass) ([]*ast.Ident, []*ast.Ident, []*ast.Ident) { //nolint:gocognit
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	filter := []ast.Node{
		(*ast.Ident)(nil),
		(*ast.FuncDecl)(nil),
	}

	funcs := []*ast.FuncDecl{}
	methods := []*ast.FuncDecl{}
	assignIdents := []*ast.Ident{}
	paramIdents := []*ast.Ident{}
	returnIdents := []*ast.Ident{}

	inspector.Preorder(filter, func(node ast.Node) {
		if f, ok := node.(*ast.FuncDecl); ok {
			funcs = append(funcs, f)
			if f.Recv != nil {
				methods = append(methods, f)
			}
			return
		}

		ident := node.(*ast.Ident)
		if ident.Obj == nil {
			return
		}

		if _, ok := ident.Obj.Decl.(*ast.AssignStmt); ok {
			assignIdents = append(assignIdents, ident)
			return
		}

		if field, ok := ident.Obj.Decl.(*ast.Field); ok {
			if isReceiver(field, methods) && !v.checkReceiver {
				return
			}

			if isReturn(field, funcs) {
				if !v.checkReturn {
					return
				}
				returnIdents = append(returnIdents, ident)
				return
			}

			paramIdents = append(paramIdents, ident)
		}
	})

	return assignIdents, paramIdents, returnIdents
}

// isReceiver returns true when field is a receiver parameter of any of the given methods.
func isReceiver(field *ast.Field, methods []*ast.FuncDecl) bool {
	for _, m := range methods {
		for _, recv := range m.Recv.List {
			if recv == field {
				return true
			}
		}
	}
	return false
}

// isReturn returns true when field is a return value of any of the given funcs.
func isReturn(field *ast.Field, funcs []*ast.FuncDecl) bool {
	for _, f := range funcs {
		if f.Type.Results == nil {
			continue
		}
		for _, r := range f.Type.Results.List {
			if r == field {
				return true
			}
		}
	}
	return false
}

// Set implements Value.
func (sv *stringsValue) Set(s string) error {
	sv.Values = strings.Split(s, ",")
	return nil
}

// String implements Value.
func (sv *stringsValue) String() string {
	return strings.Join(sv.Values, ",")
}

// contains returns true when sv contains s.
func (sv *stringsValue) contains(s string) bool {
	for _, v := range sv.Values {
		if v == s {
			return true
		}
	}
	return false
}

// isConventional returns true when p is a conventional Go parameter, such as "ctx context.Context" or
// "t *testing.T".
func (p parameter) isConventional() bool { //nolint:gocyclo,gocognit
	switch {
	case p.name == "t" && p.isPointerType("testing.T"):
		return true
	case p.name == "b" && p.isPointerType("testing.B"):
		return true
	case p.name == "tb" && p.isType("testing.TB"):
		return true
	case p.name == "pb" && p.isPointerType("testing.PB"):
		return true
	case p.name == "m" && p.isPointerType("testing.M"):
		return true
	case p.name == "ctx" && p.isType("context.Context"):
		return true
	default:
		return false
	}
}

// isType returns true when p is of type typeName.
func (p parameter) isType(typeName string) bool {
	sel, ok := p.field.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return isType(sel, typeName)
}

// isPointerType returns true when p is a pointer type of type typeName.
func (p parameter) isPointerType(typeName string) bool {
	star, ok := p.field.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return isType(sel, typeName)
}

// isType returns true when sel is a selector for type typeName.
func isType(sel *ast.SelectorExpr, typeName string) bool {
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return typeName == ident.Name+"."+sel.Sel.Name
}
