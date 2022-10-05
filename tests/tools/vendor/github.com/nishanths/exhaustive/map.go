package exhaustive

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// mapConfig is configuration for mapChecker.
type mapConfig struct {
	explicitExhaustiveMap bool
	checkGeneratedFiles   bool
	ignoreEnumMembers     *regexp.Regexp // can be nil
}

// mapChecker returns a node visitor that checks exhaustiveness
// of enum keys in map literal for the supplied pass, and reports diagnostics if non-exhaustive.
// It expects to only see *ast.CompositeLit nodes.
func mapChecker(pass *analysis.Pass, cfg mapConfig, generated generatedCache, comments commentsCache) nodeVisitor {
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

		lit := n.(*ast.CompositeLit)

		mapType, ok := pass.TypesInfo.Types[lit.Type].Type.(*types.Map)
		if !ok {
			namedType, ok2 := pass.TypesInfo.Types[lit.Type].Type.(*types.Named)
			if !ok2 {
				return true, resultNotMapLiteral
			}

			mapType, ok = namedType.Underlying().(*types.Map)
			if !ok {
				return true, resultNotMapLiteral
			}
		}

		if len(lit.Elts) == 0 {
			// because it may be used as an alternative for make(map[...]...)
			return false, resultEmptyMapLiteral
		}

		keyType, ok := mapType.Key().(*types.Named)
		if !ok {
			return true, resultMapKeyIsNotNamedType
		}

		fileComments := comments.GetComments(file, pass.Fset)
		var relatedComments []*ast.CommentGroup
		for i := range stack {
			// iterate over stack in the reverse order (from bottom to top)
			node := stack[len(stack)-1-i]
			switch node.(type) {
			// need to check comments associated with following nodes,
			// because logic of ast package doesn't allow to associate comment with *ast.CompositeLit
			case *ast.CompositeLit, // stack[len(stack)-1]
				*ast.ReturnStmt, // return ...
				*ast.IndexExpr,  // map[enum]...{...}[key]
				*ast.CallExpr,   // myfunc(map...)
				*ast.UnaryExpr,  // &map...
				*ast.AssignStmt, // variable assignment (without var keyword)
				*ast.DeclStmt,   // var declaration, parent of *ast.GenDecl
				*ast.GenDecl,    // var declaration, parent of *ast.ValueSpec
				*ast.ValueSpec:  // var declaration
				relatedComments = append(relatedComments, fileComments[node]...)
				continue
			}
			// stop iteration on the first inappropriate node
			break
		}

		if !cfg.explicitExhaustiveMap && containsIgnoreDirective(relatedComments) {
			// Skip checking of this map literal due to ignore directive comment.
			// Still return true because there may be nested map literals
			// that are not to be ignored.
			return true, resultMapIgnoreComment
		}
		if cfg.explicitExhaustiveMap && !containsEnforceDirective(relatedComments) {
			// Skip checking of this map literal due to missing enforce directive comment.
			return true, resultMapNoEnforceComment
		}

		keyPkg := keyType.Obj().Pkg()
		if keyPkg == nil {
			// The Go documentation says: nil for labels and objects in the Universe scope.
			// This happens for the `error` type, for example.
			return true, resultNilMapKeyTypePkg
		}

		enumTyp := enumType{keyType.Obj()}
		members, ok := importFact(pass, enumTyp)
		if !ok {
			return true, resultMapKeyNotEnum
		}

		samePkg := keyPkg == pass.Pkg // do the map literal and the map key type (i.e. enum type) live in the same package?
		checkUnexported := samePkg    // we want to include unexported members in the exhaustiveness check only if we're in the same package
		checklist := makeChecklist(members, keyPkg, checkUnexported, cfg.ignoreEnumMembers)

		for _, e := range lit.Elts {
			expr, ok := e.(*ast.KeyValueExpr)
			if !ok {
				continue // is it possible for valid map literal?
			}
			analyzeCaseClauseExpr(expr.Key, pass.TypesInfo, checklist.found)
		}

		if len(checklist.remaining()) == 0 {
			// All enum members accounted for.
			// Nothing to report.
			return true, resultEnumMembersAccounted
		}

		pass.Report(makeMapDiagnostic(lit, samePkg, enumTyp, members, checklist.remaining()))
		return true, resultReportedDiagnostic
	}
}

// Makes a "missing map keys" diagnostic.
// samePkg should be true if the enum type and the map literal are defined in the same package.
func makeMapDiagnostic(lit *ast.CompositeLit, samePkg bool, enumTyp enumType, allMembers enumMembers, missingMembers map[string]struct{}) analysis.Diagnostic {
	message := fmt.Sprintf("missing map keys of type %s: %s",
		diagnosticEnumTypeName(enumTyp.TypeName, samePkg),
		strings.Join(diagnosticMissingMembers(missingMembers, allMembers), ", "))

	return analysis.Diagnostic{
		Pos:     lit.Pos(),
		End:     lit.End(),
		Message: message,
	}
}
