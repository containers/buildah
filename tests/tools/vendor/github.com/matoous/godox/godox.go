package godox

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

var defaultKeywords = []string{"TODO", "BUG", "FIXME"}

// Message contains a message and position.
type Message struct {
	Pos     token.Position
	Message string
}

func getMessages(comment *ast.Comment, fset *token.FileSet, keywords []string) []Message {
	commentText := extractComment(comment.Text)

	b := bufio.NewReader(bytes.NewBufferString(commentText))

	var comments []Message

	for lineNum := 0; ; lineNum++ {
		line, _, err := b.ReadLine()
		if err != nil {
			break
		}

		const minimumSize = 4

		sComment := bytes.TrimSpace(line)
		if len(sComment) < minimumSize {
			continue
		}

		for _, kw := range keywords {
			if lkw := len(kw); !(bytes.EqualFold([]byte(kw), sComment[0:lkw]) &&
				!hasAlphanumRuneAdjacent(sComment[lkw:])) {
				continue
			}

			pos := fset.Position(comment.Pos())
			// trim the comment
			const commentLimit = 40
			if len(sComment) > commentLimit {
				sComment = []byte(fmt.Sprintf("%.40s...", sComment))
			}

			comments = append(comments, Message{
				Pos: pos,
				Message: fmt.Sprintf(
					"%s:%d: Line contains %s: %q",
					filepath.Clean(pos.Filename),
					pos.Line+lineNum,
					strings.Join(keywords, "/"),
					sComment,
				),
			})

			break
		}
	}

	return comments
}

func extractComment(commentText string) string {
	switch commentText[1] {
	case '/':
		commentText = commentText[2:]
		if len(commentText) > 0 && commentText[0] == ' ' {
			commentText = commentText[1:]
		}
	case '*':
		commentText = commentText[2 : len(commentText)-2]
	}

	return commentText
}

func hasAlphanumRuneAdjacent(rest []byte) bool {
	if len(rest) == 0 {
		return false
	}

	switch rest[0] { // most common cases
	case ':', ' ', '(':
		return false
	}

	r, _ := utf8.DecodeRune(rest)

	return unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsDigit(r)
}

// Run runs the godox linter on given file.
// Godox searches for comments starting with given keywords and reports them.
func Run(file *ast.File, fset *token.FileSet, keywords ...string) []Message {
	if len(keywords) == 0 {
		keywords = defaultKeywords
	}

	var messages []Message

	for _, c := range file.Comments {
		for _, ci := range c.List {
			messages = append(messages, getMessages(ci, fset, keywords)...)
		}
	}

	return messages
}
