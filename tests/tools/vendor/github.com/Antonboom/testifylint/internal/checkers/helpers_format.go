package checkers

import (
	"bytes"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

// formatAsCallArgs joins a, b and c and returns bytes like `a, b, c`.
func formatAsCallArgs(pass *analysis.Pass, args ...ast.Expr) []byte {
	if len(args) == 0 {
		return []byte("")
	}

	var buf bytes.Buffer
	for i, arg := range args {
		buf.Write(analysisutil.NodeBytes(pass.Fset, arg))
		if i != len(args)-1 {
			buf.WriteString(", ")
		}
	}
	return buf.Bytes()
}
