package exhaustive

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// Generated file definition
// http://golang.org/s/generatedcode
//
//  To convey to humans and machine tools that code is generated, generated
//  source should have a line that matches the following regular expression (in
//  Go syntax):
//
//    ^// Code generated .* DO NOT EDIT\.$
//
//  This line must appear before the first non-comment, non-blank
//  text in the file.

func isGeneratedFile(file *ast.File) bool {
	// NOTE: file.Comments includes file.Doc as well, so no need
	// to separately check file.Doc.

	for _, c := range file.Comments {
		for _, cc := range c.List {
			// This check is intended to handle "must appear before the
			// first non-comment, non-blank text in the file".
			// TODO: Is this check fully correct? Seems correct based
			// on https://golang.org/ref/spec#Source_file_organization.
			if c.Pos() >= file.Package {
				return false
			}
			// According to the docs:
			// '\r' has been removed.
			// '\n' has been removed for //-style comments, which is what we care about.
			// Also manually verified.
			if isGeneratedFileComment(cc.Text) {
				return true
			}
		}
	}

	return false
}

var generatedCodeRe = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

func isGeneratedFileComment(s string) bool {
	return generatedCodeRe.MatchString(s)
}

type generatedCache map[*ast.File]bool

func (c generatedCache) IsGenerated(file *ast.File) bool {
	if _, ok := c[file]; !ok {
		c[file] = isGeneratedFile(file)
	}
	return c[file]
}

// ignoreDirective is used to exclude checking of specific switch statements.
const ignoreDirective = "//exhaustive:ignore"
const enforceDirective = "//exhaustive:enforce"

type commentsCache map[*ast.File]ast.CommentMap

func (c commentsCache) GetComments(file *ast.File, set *token.FileSet) ast.CommentMap {
	if _, ok := c[file]; !ok {
		c[file] = ast.NewCommentMap(set, file, file.Comments)
	}
	return c[file]
}

func containsDirective(comments []*ast.CommentGroup, directive string) bool {
	for _, c := range comments {
		for _, cc := range c.List {
			if strings.HasPrefix(cc.Text, directive) {
				return true
			}
		}
	}
	return false
}

func containsEnforceDirective(comments []*ast.CommentGroup) bool {
	return containsDirective(comments, enforceDirective)
}

func containsIgnoreDirective(comments []*ast.CommentGroup) bool {
	return containsDirective(comments, ignoreDirective)
}
