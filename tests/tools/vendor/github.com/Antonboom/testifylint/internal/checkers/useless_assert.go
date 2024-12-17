package checkers

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

// UselessAssert detects useless asserts like
//
// 1) Asserting of the same variable
//
//	assert.Equal(t, tt.value, tt.value)
//	assert.ElementsMatch(t, users, users)
//	...
//	assert.True(t, num > num)
//	assert.False(t, num == num)
//
// 2) Open for contribution...
type UselessAssert struct{}

// NewUselessAssert constructs UselessAssert checker.
func NewUselessAssert() UselessAssert { return UselessAssert{} }
func (UselessAssert) Name() string    { return "useless-assert" }

func (checker UselessAssert) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	var first, second ast.Node

	switch call.Fn.NameFTrimmed {
	case
		"Contains",
		"ElementsMatch",
		"Equal",
		"EqualExportedValues",
		"EqualValues",
		"ErrorAs",
		"ErrorIs",
		"Exactly",
		"Greater",
		"GreaterOrEqual",
		"Implements",
		"InDelta",
		"InDeltaMapValues",
		"InDeltaSlice",
		"InEpsilon",
		"InEpsilonSlice",
		"IsType",
		"JSONEq",
		"Less",
		"LessOrEqual",
		"NotEqual",
		"NotEqualValues",
		"NotErrorIs",
		"NotRegexp",
		"NotSame",
		"NotSubset",
		"Regexp",
		"Same",
		"Subset",
		"WithinDuration",
		"YAMLEq":
		if len(call.Args) < 2 {
			return nil
		}
		first, second = call.Args[0], call.Args[1]

	case "True", "False":
		if len(call.Args) < 1 {
			return nil
		}

		be, ok := call.Args[0].(*ast.BinaryExpr)
		if !ok {
			return nil
		}
		first, second = be.X, be.Y

	default:
		return nil
	}

	if analysisutil.NodeString(pass.Fset, first) == analysisutil.NodeString(pass.Fset, second) {
		return newDiagnostic(checker.Name(), call, "asserting of the same variable", nil)
	}
	return nil
}
