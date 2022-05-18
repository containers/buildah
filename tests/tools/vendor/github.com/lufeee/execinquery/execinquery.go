package execinquery

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = "execinquery is a linter about query string checker in Query function which reads your Go src files and warning it finds"

// Analyzer is checking database/sql pkg Query's function
var Analyzer = &analysis.Analyzer{
	Name: "execinquery",
	Doc:  doc,
	Run:  newLinter().run,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
}

type linter struct {
	commentExp          *regexp.Regexp
	multilineCommentExp *regexp.Regexp
}

func newLinter() *linter {
	return &linter{
		commentExp:          regexp.MustCompile(`--[^\n]*\n`),
		multilineCommentExp: regexp.MustCompile(`(?s)/\*.*?\*/`),
	}
}

func (l linter) run(pass *analysis.Pass) (interface{}, error) {
	result := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	result.Preorder(nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.CallExpr:
			selector, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return
			}

			if pass.TypesInfo == nil || pass.TypesInfo.Uses[selector.Sel] == nil || pass.TypesInfo.Uses[selector.Sel].Pkg() == nil {
				return
			}

			if "database/sql" != pass.TypesInfo.Uses[selector.Sel].Pkg().Path() {
				return
			}

			if !strings.Contains(selector.Sel.Name, "Query") {
				return
			}

			replacement := "Exec"
			var i int // the index of the query argument
			if strings.Contains(selector.Sel.Name, "Context") {
				replacement += "Context"
				i = 1
			}

			if len(n.Args) <= i {
				return
			}

			query := l.getQueryString(n.Args[i])
			if query == "" {
				return
			}

			query = strings.TrimSpace(l.cleanValue(query))
			parts := strings.SplitN(query, " ", 2)
			cmd := strings.ToUpper(parts[0])

			if strings.HasPrefix(cmd, "SELECT") {
				return
			}

			pass.Reportf(n.Fun.Pos(), "Use %s instead of %s to execute `%s` query", replacement, selector.Sel.Name, cmd)
		}
	})

	return nil, nil
}

func (l linter) cleanValue(s string) string {
	v := strings.NewReplacer(`"`, "", "`", "").Replace(s)

	v = l.multilineCommentExp.ReplaceAllString(v, "")

	return l.commentExp.ReplaceAllString(v, "")
}

func (l linter) getQueryString(exp interface{}) string {
	switch e := exp.(type) {
	case *ast.AssignStmt:
		var v string
		for _, stmt := range e.Rhs {
			v += l.cleanValue(l.getQueryString(stmt))
		}
		return v

	case *ast.BasicLit:
		return e.Value

	case *ast.ValueSpec:
		var v string
		for _, value := range e.Values {
			v += l.cleanValue(l.getQueryString(value))
		}
		return v

	case *ast.Ident:
		if e.Obj == nil {
			return ""
		}
		return l.getQueryString(e.Obj.Decl)

	case *ast.BinaryExpr:
		v := l.cleanValue(l.getQueryString(e.X))
		v += l.cleanValue(l.getQueryString(e.Y))
		return v
	}

	return ""
}
