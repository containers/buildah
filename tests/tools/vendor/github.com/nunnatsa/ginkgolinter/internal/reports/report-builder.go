package reports

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type Builder struct {
	pos        token.Pos
	end        token.Pos
	oldExpr    string
	issues     []string
	fixOffer   string
	suggestFix bool
}

func NewBuilder(fset *token.FileSet, oldExpr ast.Expr) *Builder {
	b := &Builder{
		pos:        oldExpr.Pos(),
		end:        oldExpr.End(),
		oldExpr:    goFmt(fset, oldExpr),
		suggestFix: false,
	}

	return b
}

func (b *Builder) AddIssue(suggestFix bool, issue string, args ...any) {
	if len(args) > 0 {
		issue = fmt.Sprintf(issue, args...)
	}
	b.issues = append(b.issues, issue)

	if suggestFix {
		b.suggestFix = true
	}
}

func (b *Builder) SetFixOffer(fset *token.FileSet, fixOffer ast.Expr) {
	if offer := goFmt(fset, fixOffer); offer != b.oldExpr {
		b.fixOffer = offer
	}
}

func (b *Builder) HasReport() bool {
	return len(b.issues) > 0
}

func (b *Builder) Build() analysis.Diagnostic {
	diagnostic := analysis.Diagnostic{
		Pos:     b.pos,
		Message: b.getMessage(),
	}

	if b.suggestFix && len(b.fixOffer) > 0 {
		diagnostic.SuggestedFixes = []analysis.SuggestedFix{
			{
				Message: fmt.Sprintf("should replace %s with %s", b.oldExpr, b.fixOffer),
				TextEdits: []analysis.TextEdit{
					{
						Pos:     b.pos,
						End:     b.end,
						NewText: []byte(b.fixOffer),
					},
				},
			},
		}
	}

	return diagnostic
}

func goFmt(fset *token.FileSet, x ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, x)
	return b.String()
}

func (b *Builder) getMessage() string {
	sb := strings.Builder{}
	sb.WriteString("ginkgo-linter: ")
	if len(b.issues) > 1 {
		sb.WriteString("multiple issues: ")
	}
	sb.WriteString(strings.Join(b.issues, "; "))

	if b.suggestFix && len(b.fixOffer) != 0 {
		sb.WriteString(fmt.Sprintf(". Consider using `%s` instead", b.fixOffer))
	}

	return sb.String()
}
