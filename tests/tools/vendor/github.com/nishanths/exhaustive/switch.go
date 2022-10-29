package exhaustive

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
)

// nodeVisitor is like the visitor function used by Inspector.WithStack,
// except that it returns an additional value: a short description of
// the result of this node visit.
//
// The result is typically useful in debugging or in unit tests to check
// that the nodeVisitor function took the expected code path.
type nodeVisitor func(n ast.Node, push bool, stack []ast.Node) (proceed bool, result string)

// Result values returned by a node visitor constructed via switchChecker.
const (
	resultNotPush                = "not push"
	resultGeneratedFile          = "generated file"
	resultNoSwitchTag            = "no switch tag"
	resultEmptyMapLiteral        = "empty map literal"
	resultNotMapLiteral          = "not map literal"
	resultMapKeyIsNotNamedType   = "map key is not named type"
	resultNilMapKeyTypePkg       = "nil map key type package"
	resultMapKeyNotEnum          = "map key not known enum type"
	resultMapIgnoreComment       = "map literal has ignore comment"
	resultMapNoEnforceComment    = "map literal has no enforce comment"
	resultTagNotValue            = "switch tag not value type"
	resultTagNotNamed            = "switch tag not named type"
	resultTagNoPkg               = "switch tag does not belong to regular package"
	resultTagNotEnum             = "switch tag not known enum type"
	resultSwitchIgnoreComment    = "switch statement has ignore comment"
	resultSwitchNoEnforceComment = "switch statement has no enforce comment"
	resultEnumMembersAccounted   = "requisite enum members accounted for"
	resultDefaultCaseSuffices    = "default case presence satisfies exhaustiveness"
	resultReportedDiagnostic     = "reported diagnostic"
)

// switchChecker returns a node visitor that checks exhaustiveness
// of enum switch statements for the supplied pass, and reports diagnostics for
// switch statements that are non-exhaustive.
// It expects to only see *ast.SwitchStmt nodes.
func switchChecker(pass *analysis.Pass, cfg switchConfig, generated generatedCache, comments commentsCache) nodeVisitor {
	return func(n ast.Node, push bool, stack []ast.Node) (bool, string) {
		if !push {
			// The proceed return value should not matter; it is ignored by
			// inspector package for pop calls.
			// Nevertheless, return true to be on the safe side for the future.
			return true, resultNotPush
		}

		file := stack[0].(*ast.File)

		if !cfg.checkGeneratedFiles && generated.IsGenerated(file) {
			// Don't check this file.
			// Return false because the children nodes of node `n` don't have to be checked.
			return false, resultGeneratedFile
		}

		sw := n.(*ast.SwitchStmt)

		switchComments := comments.GetComments(file, pass.Fset)[sw]
		if !cfg.explicitExhaustiveSwitch && containsIgnoreDirective(switchComments) {
			// Skip checking of this switch statement due to ignore directive comment.
			// Still return true because there may be nested switch statements
			// that are not to be ignored.
			return true, resultSwitchIgnoreComment
		}
		if cfg.explicitExhaustiveSwitch && !containsEnforceDirective(switchComments) {
			// Skip checking of this switch statement due to missing enforce directive comment.
			return true, resultSwitchNoEnforceComment
		}

		if sw.Tag == nil {
			return true, resultNoSwitchTag
		}

		t := pass.TypesInfo.Types[sw.Tag]
		if !t.IsValue() {
			return true, resultTagNotValue
		}

		tagType, ok := t.Type.(*types.Named)
		if !ok {
			return true, resultTagNotNamed
		}

		tagPkg := tagType.Obj().Pkg()
		if tagPkg == nil {
			// The Go documentation says: nil for labels and objects in the Universe scope.
			// This happens for the `error` type, for example.
			return true, resultTagNoPkg
		}

		enumTyp := enumType{tagType.Obj()}
		members, ok := importFact(pass, enumTyp)
		if !ok {
			// switch tag's type is not a known enum type.
			return true, resultTagNotEnum
		}

		samePkg := tagPkg == pass.Pkg // do the switch statement and the switch tag type (i.e. enum type) live in the same package?
		checkUnexported := samePkg    // we want to include unexported members in the exhaustiveness check only if we're in the same package
		checklist := makeChecklist(members, tagPkg, checkUnexported, cfg.ignoreEnumMembers)

		hasDefaultCase := analyzeSwitchClauses(sw, pass.TypesInfo, checklist.found)

		if len(checklist.remaining()) == 0 {
			// All enum members accounted for.
			// Nothing to report.
			return true, resultEnumMembersAccounted
		}
		if hasDefaultCase && cfg.defaultSignifiesExhaustive {
			// Though enum members are not accounted for,
			// the existence of the default case signifies exhaustiveness.
			// So don't report.
			return true, resultDefaultCaseSuffices
		}
		pass.Report(makeSwitchDiagnostic(sw, samePkg, enumTyp, members, checklist.remaining()))
		return true, resultReportedDiagnostic
	}
}

// switchConfig is configuration for switchChecker.
type switchConfig struct {
	explicitExhaustiveSwitch   bool
	defaultSignifiesExhaustive bool
	checkGeneratedFiles        bool
	ignoreEnumMembers          *regexp.Regexp // can be nil
}

func isDefaultCase(c *ast.CaseClause) bool {
	return c.List == nil // see doc comment on List field
}

func denotesPackage(ident *ast.Ident, info *types.Info) (*types.Package, bool) {
	obj := info.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}
	n, ok := obj.(*types.PkgName)
	if !ok {
		return nil, false
	}
	return n.Imported(), true
}

// analyzeSwitchClauses analyzes the clauses in the supplied switch statement.
// The info param should typically be pass.TypesInfo. The found function is
// called for each enum member name found in the switch statement.
// The hasDefaultCase return value indicates whether the switch statement has a
// default clause.
func analyzeSwitchClauses(sw *ast.SwitchStmt, info *types.Info, found func(val constantValue)) (hasDefaultCase bool) {
	for _, stmt := range sw.Body.List {
		caseCl := stmt.(*ast.CaseClause)
		if isDefaultCase(caseCl) {
			hasDefaultCase = true
			continue // nothing more to do if it's the default case
		}
		for _, expr := range caseCl.List {
			analyzeCaseClauseExpr(expr, info, found)
		}
	}
	return hasDefaultCase
}

func analyzeCaseClauseExpr(e ast.Expr, info *types.Info, found func(val constantValue)) {
	handleIdent := func(ident *ast.Ident) {
		obj := info.Uses[ident]
		if obj == nil {
			return
		}
		if _, ok := obj.(*types.Const); !ok {
			return
		}

		// There are two scenarios.
		// See related test cases in typealias/quux/quux.go.
		//
		// ### Scenario 1
		//
		// Tag package and constant package are the same.
		//
		// For example:
		//   var mode fs.FileMode
		//   switch mode {
		//   case fs.ModeDir:
		//   }
		//
		// This is simple: we just use fs.ModeDir's value.
		//
		// ### Scenario 2
		//
		// Tag package and constant package are different.
		//
		// For example:
		//   var mode fs.FileMode
		//   switch mode {
		//   case os.ModeDir:
		//   }
		//
		// Or equivalently:
		//   var mode os.FileMode // in effect, fs.FileMode because of type alias in package os
		//   switch mode {
		//   case os.ModeDir:
		//   }
		//
		// In this scenario, too, we accept the case clause expr constant
		// value, as is. If the Go type checker is okay with the
		// name being listed in the case clause, we don't care much further.
		//
		found(determineConstVal(ident, info))
	}

	e = astutil.Unparen(e)
	switch e := e.(type) {
	case *ast.Ident:
		handleIdent(e)

	case *ast.SelectorExpr:
		x := astutil.Unparen(e.X)
		// Ensure we only see the form `pkg.Const`, and not e.g. `structVal.f`
		// or `structVal.inner.f`.
		// Check that X, which is everything except the rightmost *ast.Ident (or
		// Sel), is also an *ast.Ident.
		xIdent, ok := x.(*ast.Ident)
		if !ok {
			return
		}
		// Doesn't matter which package, just that it denotes a package.
		if _, ok := denotesPackage(xIdent, info); !ok {
			return
		}
		handleIdent(e.Sel)
	}
}

// diagnosticMissingMembers constructs the list of missing enum members,
// suitable for use in a reported diagnostic message.
// Order is the same as in enumMembers.Names.
func diagnosticMissingMembers(missingMembers map[string]struct{}, em enumMembers) []string {
	missingNamesGroupedByValue := make([][]string, len(em.Names)) // empty groups will be filtered out later
	firstIndex := make(map[constantValue]int, len(em.ValueToNames))
	for i, name := range em.Names {
		value := em.NameToValue[name]
		j, ok := firstIndex[value]
		if !ok {
			firstIndex[value] = i
			j = i
		}

		if _, missing := missingMembers[name]; missing {
			missingNamesGroupedByValue[j] = append(missingNamesGroupedByValue[j], name)
		}
	}

	out := make([]string, 0, len(missingMembers))
	for _, names := range missingNamesGroupedByValue {
		if len(names) == 0 {
			continue
		}
		out = append(out, strings.Join(names, "|"))
	}
	return out
}

// diagnosticEnumTypeName returns a string representation of an enum type for
// use in reported diagnostics.
func diagnosticEnumTypeName(enumType *types.TypeName, samePkg bool) string {
	if samePkg {
		return enumType.Name()
	}
	return enumType.Pkg().Name() + "." + enumType.Name()
}

// Makes a "missing cases in switch" diagnostic.
// samePkg should be true if the enum type and the switch statement are defined
// in the same package.
func makeSwitchDiagnostic(sw *ast.SwitchStmt, samePkg bool, enumTyp enumType, allMembers enumMembers, missingMembers map[string]struct{}) analysis.Diagnostic {
	message := fmt.Sprintf("missing cases in switch of type %s: %s",
		diagnosticEnumTypeName(enumTyp.TypeName, samePkg),
		strings.Join(diagnosticMissingMembers(missingMembers, allMembers), ", "))

	return analysis.Diagnostic{
		Pos:     sw.Pos(),
		End:     sw.End(),
		Message: message,
	}
}

// A checklist holds a set of enum member names that have to be
// accounted for to satisfy exhaustiveness in an enum switch statement.
//
// The found method checks off member names from the set, based on
// constant value, when a constant value is encoutered in the switch
// statement's cases.
//
// The remaining method returns the member names not accounted for.
type checklist struct {
	em     enumMembers
	checkl map[string]struct{}
}

func makeChecklist(em enumMembers, enumPkg *types.Package, includeUnexported bool, ignore *regexp.Regexp) *checklist {
	checkl := make(map[string]struct{})

	add := func(memberName string) {
		if memberName == "_" {
			// Blank identifier is often used to skip entries in iota lists.
			// Also, it can't be referenced anywhere (including in a switch
			// statement's cases), so it doesn't make sense to include it
			// as required member to satisfy exhaustiveness.
			return
		}
		if !ast.IsExported(memberName) && !includeUnexported {
			return
		}
		if ignore != nil && ignore.MatchString(enumPkg.Path()+"."+memberName) {
			return
		}
		checkl[memberName] = struct{}{}
	}

	for _, name := range em.Names {
		add(name)
	}

	return &checklist{
		em:     em,
		checkl: checkl,
	}
}

func (c *checklist) found(val constantValue) {
	// Delete all of the same-valued names.
	for _, name := range c.em.ValueToNames[val] {
		delete(c.checkl, name)
	}
}

func (c *checklist) remaining() map[string]struct{} {
	return c.checkl
}
