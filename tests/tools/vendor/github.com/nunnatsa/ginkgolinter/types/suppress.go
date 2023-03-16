package types

import (
	"strings"

	"go/ast"
)

const (
	suppressPrefix                 = "ginkgo-linter:"
	suppressLengthAssertionWarning = suppressPrefix + "ignore-len-assert-warning"
	suppressNilAssertionWarning    = suppressPrefix + "ignore-nil-assert-warning"
	suppressErrAssertionWarning    = suppressPrefix + "ignore-err-assert-warning"
)

type Suppress struct {
	Len Boolean
	Nil Boolean
	Err Boolean
}

func (s Suppress) AllTrue() bool {
	return bool(s.Len && s.Nil && s.Err)
}

func (s Suppress) Clone() Suppress {
	return Suppress{
		Len: s.Len,
		Nil: s.Nil,
		Err: s.Err,
	}
}

func (s *Suppress) UpdateFromComment(commentGroup []*ast.CommentGroup) {
	for _, cmntList := range commentGroup {
		if s.AllTrue() {
			break
		}

		for _, cmnt := range cmntList.List {
			commentLines := strings.Split(cmnt.Text, "\n")
			for _, comment := range commentLines {
				comment = strings.TrimPrefix(comment, "//")
				comment = strings.TrimPrefix(comment, "/*")
				comment = strings.TrimSuffix(comment, "*/")
				comment = strings.TrimSpace(comment)

				s.Len = s.Len || (comment == suppressLengthAssertionWarning)
				s.Nil = s.Nil || (comment == suppressNilAssertionWarning)
				s.Err = s.Err || (comment == suppressErrAssertionWarning)
			}
		}
	}
}

func (s *Suppress) UpdateFromFile(cm ast.CommentMap) {

	for key, commentGroup := range cm {
		if s.AllTrue() {
			break
		}

		if _, ok := key.(*ast.GenDecl); ok {
			s.UpdateFromComment(commentGroup)
		}
	}
}
