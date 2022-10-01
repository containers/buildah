package analyzer

import (
	"fmt"
	"go/ast"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const FlagPattern = "pattern"

func New() *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "reassign",
		Doc:  "Checks that package variables are not reassigned",
		Run:  run,
	}
	a.Flags.String(FlagPattern, `^(Err.*|EOF)$`, "Pattern to match package variables against to prevent reassignment")
	return a
}

func run(pass *analysis.Pass) (interface{}, error) {
	checkRE, err := regexp.Compile(pass.Analyzer.Flags.Lookup(FlagPattern).Value.String())
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}
	for _, f := range pass.Files {
		state := &fileState{imports: make(map[string]struct{})}
		ast.Inspect(f, func(node ast.Node) bool {
			return inspect(pass, node, checkRE, state)
		})
	}
	return nil, nil
}

type fileState struct {
	imports map[string]struct{}
}

func inspect(pass *analysis.Pass, node ast.Node, checkRE *regexp.Regexp, state *fileState) bool {
	if importSpec, ok := node.(*ast.ImportSpec); ok {
		if importSpec.Name != nil {
			state.imports[importSpec.Name.Name] = struct{}{}
		} else {
			n, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				return true
			}
			if idx := strings.LastIndexByte(n, '/'); idx != -1 {
				n = n[idx+1:]
			}
			state.imports[n] = struct{}{}
		}
		return true
	}

	assignStmt, ok := node.(*ast.AssignStmt)
	if !ok {
		return true
	}

	for _, lhs := range assignStmt.Lhs {
		selector, ok := lhs.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if !checkRE.MatchString(selector.Sel.Name) {
			return true
		}

		selectIdent, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}

		if _, ok := state.imports[selectIdent.Name]; ok {
			pass.Reportf(node.Pos(), "reassigning variable %s in other package %s", selector.Sel.Name, selectIdent.Name)
		}
	}

	return true
}
