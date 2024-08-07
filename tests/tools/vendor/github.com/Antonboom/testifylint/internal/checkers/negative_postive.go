package checkers

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
)

// NegativePositive detects situations like
//
//	assert.Less(t, a, 0)
//	assert.Greater(t, 0, a)
//	assert.True(t, a < 0)
//	assert.True(t, 0 > a)
//	assert.False(t, a >= 0)
//	assert.False(t, 0 <= a)
//
//	assert.Greater(t, a, 0)
//	assert.Less(t, 0, a)
//	assert.True(t, a > 0)
//	assert.True(t, 0 < a)
//	assert.False(t, a <= 0)
//	assert.False(t, 0 >= a)
//
// and requires
//
//	assert.Negative(t, value)
//	assert.Positive(t, value)
type NegativePositive struct{}

// NewNegativePositive constructs NegativePositive checker.
func NewNegativePositive() NegativePositive { return NegativePositive{} }
func (NegativePositive) Name() string       { return "negative-positive" }

func (checker NegativePositive) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	if d := checker.checkNegative(pass, call); d != nil {
		return d
	}
	return checker.checkPositive(pass, call)
}

func (checker NegativePositive) checkNegative(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	newUseNegativeDiagnostic := func(replaceStart, replaceEnd token.Pos, replaceWith ast.Expr) *analysis.Diagnostic {
		const proposed = "Negative"
		return newUseFunctionDiagnostic(checker.Name(), call, proposed,
			newSuggestedFuncReplacement(call, proposed, analysis.TextEdit{
				Pos:     replaceStart,
				End:     replaceEnd,
				NewText: analysisutil.NodeBytes(pass.Fset, replaceWith),
			}),
		)
	}

	switch call.Fn.NameFTrimmed {
	case "Less":
		if len(call.Args) < 2 {
			return nil
		}
		a, b := call.Args[0], call.Args[1]

		if isNotZero(a) && isZero(b) {
			return newUseNegativeDiagnostic(a.Pos(), b.End(), a)
		}

	case "Greater":
		if len(call.Args) < 2 {
			return nil
		}
		a, b := call.Args[0], call.Args[1]

		if isZero(a) && isNotZero(b) {
			return newUseNegativeDiagnostic(a.Pos(), b.End(), b)
		}

	case "True":
		if len(call.Args) < 1 {
			return nil
		}
		expr := call.Args[0]

		a, _, ok1 := isStrictComparisonWith(pass, expr, p(isNotZero), token.LSS, p(isZero)) // a < 0
		_, b, ok2 := isStrictComparisonWith(pass, expr, p(isZero), token.GTR, p(isNotZero)) // 0 > a

		survivingArg, ok := anyVal([]bool{ok1, ok2}, a, b)
		if ok {
			return newUseNegativeDiagnostic(expr.Pos(), expr.End(), survivingArg)
		}

	case "False":
		if len(call.Args) < 1 {
			return nil
		}
		expr := call.Args[0]

		a, _, ok1 := isStrictComparisonWith(pass, expr, p(isNotZero), token.GEQ, p(isZero)) // a >= 0
		_, b, ok2 := isStrictComparisonWith(pass, expr, p(isZero), token.LEQ, p(isNotZero)) // 0 <= a

		survivingArg, ok := anyVal([]bool{ok1, ok2}, a, b)
		if ok {
			return newUseNegativeDiagnostic(expr.Pos(), expr.End(), survivingArg)
		}
	}
	return nil
}

func (checker NegativePositive) checkPositive(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	newUsePositiveDiagnostic := func(replaceStart, replaceEnd token.Pos, replaceWith ast.Expr) *analysis.Diagnostic {
		const proposed = "Positive"
		return newUseFunctionDiagnostic(checker.Name(), call, proposed,
			newSuggestedFuncReplacement(call, proposed, analysis.TextEdit{
				Pos:     replaceStart,
				End:     replaceEnd,
				NewText: analysisutil.NodeBytes(pass.Fset, replaceWith),
			}),
		)
	}

	switch call.Fn.NameFTrimmed {
	case "Greater":
		if len(call.Args) < 2 {
			return nil
		}
		a, b := call.Args[0], call.Args[1]

		if isNotZero(a) && isZero(b) {
			return newUsePositiveDiagnostic(a.Pos(), b.End(), a)
		}

	case "Less":
		if len(call.Args) < 2 {
			return nil
		}
		a, b := call.Args[0], call.Args[1]

		if isZero(a) && isNotZero(b) {
			return newUsePositiveDiagnostic(a.Pos(), b.End(), b)
		}

	case "True":
		if len(call.Args) < 1 {
			return nil
		}
		expr := call.Args[0]

		a, _, ok1 := isStrictComparisonWith(pass, expr, p(isNotZero), token.GTR, p(isZero)) // a > 0
		_, b, ok2 := isStrictComparisonWith(pass, expr, p(isZero), token.LSS, p(isNotZero)) // 0 < a

		survivingArg, ok := anyVal([]bool{ok1, ok2}, a, b)
		if ok {
			return newUsePositiveDiagnostic(expr.Pos(), expr.End(), survivingArg)
		}

	case "False":
		if len(call.Args) < 1 {
			return nil
		}
		expr := call.Args[0]

		a, _, ok1 := isStrictComparisonWith(pass, expr, p(isNotZero), token.LEQ, p(isZero)) // a <= 0
		_, b, ok2 := isStrictComparisonWith(pass, expr, p(isZero), token.GEQ, p(isNotZero)) // 0 >= a

		survivingArg, ok := anyVal([]bool{ok1, ok2}, a, b)
		if ok {
			return newUsePositiveDiagnostic(expr.Pos(), expr.End(), survivingArg)
		}
	}
	return nil
}
