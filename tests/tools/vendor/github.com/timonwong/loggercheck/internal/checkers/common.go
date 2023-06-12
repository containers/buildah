package checkers

import (
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"strconv"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"

	"github.com/timonwong/loggercheck/internal/bytebufferpool"
)

const (
	DiagnosticCategory = "logging"
)

// extractValueFromStringArg returns true if the argument is string literal or string constant.
func extractValueFromStringArg(pass *analysis.Pass, arg ast.Expr) (value string, ok bool) {
	switch arg := arg.(type) {
	case *ast.BasicLit: // literals, string literals specifically
		if arg.Kind == token.STRING {
			if val, err := strconv.Unquote(arg.Value); err == nil {
				return val, true
			}
		}
	case *ast.Ident: // identifiers, string constants specifically
		if arg.Obj != nil && arg.Obj.Kind == ast.Con {
			typeAndValue := pass.TypesInfo.Types[arg]
			if typ, ok := typeAndValue.Type.(*types.Basic); ok && typ.Kind() == types.String {
				return constant.StringVal(typeAndValue.Value), true
			}
		}
	}

	return "", false
}

func renderNodeEllipsis(fset *token.FileSet, v interface{}) string {
	const maxLen = 20

	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)

	_ = printer.Fprint(buf, fset, v)
	s := buf.String()
	if utf8.RuneCountInString(s) > maxLen {
		// Copied from go/constant/value.go
		i := 0
		for n := 0; n < maxLen-3; n++ {
			_, size := utf8.DecodeRuneInString(s[i:])
			i += size
		}
		s = s[:i] + "..."
	}
	return s
}
