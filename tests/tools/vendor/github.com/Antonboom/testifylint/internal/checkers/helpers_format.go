package checkers

import (
	"bytes"
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

// formatAsCallArgs joins a and b and return bytes like `a, b`.
func formatAsCallArgs(pass *analysis.Pass, a, b ast.Node) []byte {
	return bytes.Join([][]byte{
		analysisutil.NodeBytes(pass.Fset, a),
		analysisutil.NodeBytes(pass.Fset, b),
	}, []byte(", "))
}
