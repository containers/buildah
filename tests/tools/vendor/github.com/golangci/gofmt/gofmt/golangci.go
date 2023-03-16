package gofmt

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

type RewriteRule struct {
	Pattern     string
	Replacement string
}

// Run runs gofmt.
// Deprecated: use RunRewrite instead.
func Run(filename string, needSimplify bool) ([]byte, error) {
	return RunRewrite(filename, needSimplify, nil)
}

// RunRewrite runs gofmt.
// empty string `rewrite` will be ignored.
func RunRewrite(filename string, needSimplify bool, rewriteRules []RewriteRule) ([]byte, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()

	initParserMode()

	file, sourceAdj, indentAdj, err := parse(fset, filename, src, false)
	if err != nil {
		return nil, err
	}

	file, err = rewriteFileContent(fset, file, rewriteRules)
	if err != nil {
		return nil, err
	}

	ast.SortImports(fset, file)

	if needSimplify {
		simplify(file)
	}

	res, err := format(fset, file, sourceAdj, indentAdj, src, printer.Config{Mode: printerMode, Tabwidth: tabWidth})
	if err != nil {
		return nil, err
	}

	if bytes.Equal(src, res) {
		return nil, nil
	}

	// formatting has changed
	data, err := diffWithReplaceTempFile(src, res, filename)
	if err != nil {
		return nil, fmt.Errorf("error computing diff: %s", err)
	}

	return data, nil
}

func rewriteFileContent(fset *token.FileSet, file *ast.File, rewriteRules []RewriteRule) (*ast.File, error) {
	for _, rewriteRule := range rewriteRules {
		pattern, err := parseExpression(rewriteRule.Pattern, "pattern")
		if err != nil {
			return nil, err
		}

		replacement, err := parseExpression(rewriteRule.Replacement, "replacement")
		if err != nil {
			return nil, err
		}

		file = rewriteFile(fset, pattern, replacement, file)
	}

	return file, nil
}

func parseExpression(s, what string) (ast.Expr, error) {
	x, err := parser.ParseExpr(s)
	if err != nil {
		return nil, fmt.Errorf("parsing %s %q at %s\n", what, s, err)
	}
	return x, nil
}
