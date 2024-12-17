package rule

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"regexp"
	"strings"

	"github.com/mgechev/revive/lint"
)

// isBlank returns whether id is the blank identifier "_".
// If id == nil, the answer is false.
func isBlank(id *ast.Ident) bool { return id != nil && id.Name == "_" }

var commonMethods = map[string]bool{
	"Error":     true,
	"Read":      true,
	"ServeHTTP": true,
	"String":    true,
	"Write":     true,
	"Unwrap":    true,
}

var knownNameExceptions = map[string]bool{
	"LastInsertId": true, // must match database/sql
	"kWh":          true,
}

func isCgoExported(f *ast.FuncDecl) bool {
	if f.Recv != nil || f.Doc == nil {
		return false
	}

	cgoExport := regexp.MustCompile(fmt.Sprintf("(?m)^//export %s$", regexp.QuoteMeta(f.Name.Name)))
	for _, c := range f.Doc.List {
		if cgoExport.MatchString(c.Text) {
			return true
		}
	}
	return false
}

var allCapsRE = regexp.MustCompile(`^[A-Z0-9_]+$`)

func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}

var zeroLiteral = map[string]bool{
	"false": true, // bool
	// runes
	`'\x00'`: true,
	`'\000'`: true,
	// strings
	`""`: true,
	"``": true,
	// numerics
	"0":   true,
	"0.":  true,
	"0.0": true,
	"0i":  true,
}

func validType(t types.Type) bool {
	return t != nil &&
		t != types.Typ[types.Invalid] &&
		!strings.Contains(t.String(), "invalid type") // good but not foolproof
}

// isPkgDot checks if the expression is <pkg>.<name>
func isPkgDot(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkg) && isIdent(sel.Sel, name)
}

func srcLine(src []byte, p token.Position) string {
	// Run to end of line in both directions if not at line start/end.
	lo, hi := p.Offset, p.Offset+1
	for lo > 0 && src[lo-1] != '\n' {
		lo--
	}
	for hi < len(src) && src[hi-1] != '\n' {
		hi++
	}
	return string(src[lo:hi])
}

// pick yields a list of nodes by picking them from a sub-ast with root node n.
// Nodes are selected by applying the fselect function
func pick(n ast.Node, fselect func(n ast.Node) bool) []ast.Node {
	var result []ast.Node

	if n == nil {
		return result
	}

	onSelect := func(n ast.Node) {
		result = append(result, n)
	}
	p := picker{fselect: fselect, onSelect: onSelect}
	ast.Walk(p, n)
	return result
}

type picker struct {
	fselect  func(n ast.Node) bool
	onSelect func(n ast.Node)
}

func (p picker) Visit(node ast.Node) ast.Visitor {
	if p.fselect == nil {
		return nil
	}

	if p.fselect(node) {
		p.onSelect(node)
	}

	return p
}

// isBoolOp returns true if the given token corresponds to
// a bool operator
func isBoolOp(t token.Token) bool {
	switch t {
	case token.LAND, token.LOR, token.EQL, token.NEQ:
		return true
	}

	return false
}

const (
	trueName  = "true"
	falseName = "false"
)

func isExprABooleanLit(n ast.Node) (lexeme string, ok bool) {
	oper, ok := n.(*ast.Ident)

	if !ok {
		return "", false
	}

	return oper.Name, (oper.Name == trueName || oper.Name == falseName)
}

// gofmt returns a string representation of an AST subtree.
func gofmt(x any) string {
	buf := bytes.Buffer{}
	fs := token.NewFileSet()
	printer.Fprint(&buf, fs, x)
	return buf.String()
}

// checkNumberOfArguments fails if the given number of arguments is not, at least, the expected one
func checkNumberOfArguments(expected int, args lint.Arguments, ruleName string) {
	if len(args) < expected {
		panic(fmt.Sprintf("not enough arguments for %s rule, expected %d, got %d. Please check the rule's documentation", ruleName, expected, len(args)))
	}
}

var directiveCommentRE = regexp.MustCompile("^//(line |extern |export |[a-z0-9]+:[a-z0-9])") // see https://go-review.googlesource.com/c/website/+/442516/1..2/_content/doc/comment.md#494

func isDirectiveComment(line string) bool {
	return directiveCommentRE.MatchString(line)
}
