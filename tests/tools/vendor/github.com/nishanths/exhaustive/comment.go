package exhaustive

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// For definition of generated file see:
// http://golang.org/s/generatedcode

var generatedCodeRe = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func isGeneratedFile(file *ast.File) bool {
	// NOTE: file.Comments includes file.Doc as well, so no need
	// to separately check file.Doc.
	for _, c := range file.Comments {
		for _, cc := range c.List {
			// This check handles the "must appear before the first
			// non-comment, non-blank text in the file" requirement.
			//
			// According to https://golang.org/ref/spec#Source_file_organization
			// the package clause is the first element in a file, which
			// should make it the first non-comment, non-blank text.
			if c.Pos() >= file.Package {
				return false
			}
			// According to the docs:
			//   '\r' has been removed.
			//   '\n' has been removed for //-style comments
			// This has also been manually verified.
			if generatedCodeRe.MatchString(cc.Text) {
				return true
			}
		}
	}

	return false
}

const (
	ignoreComment  = "//exhaustive:ignore"
	enforceComment = "//exhaustive:enforce"
)

func hasCommentPrefix(comments []*ast.CommentGroup, comment string) bool {
	for _, c := range comments {
		for _, cc := range c.List {
			if strings.HasPrefix(cc.Text, comment) {
				return true
			}
		}
	}
	return false
}

func fileCommentMap(fset *token.FileSet, file *ast.File) ast.CommentMap {
	return ast.NewCommentMap(fset, file, file.Comments)
}
