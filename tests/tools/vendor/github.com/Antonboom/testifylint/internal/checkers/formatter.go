package checkers

import (
	"fmt"
	"go/ast"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"

	"github.com/Antonboom/testifylint/internal/analysisutil"
	"github.com/Antonboom/testifylint/internal/checkers/printf"
	"github.com/Antonboom/testifylint/internal/testify"
)

// Formatter detects situations like
//
//	assert.ElementsMatch(t, certConfig.Org, csr.Subject.Org, "organizations not equal")
//	assert.Error(t, err, fmt.Sprintf("Profile %s should not be valid", test.profile))
//	assert.Errorf(t, err, fmt.Sprintf("test %s", test.testName))
//	assert.Truef(t, targetTs.Equal(ts), "the timestamp should be as expected (%s) but was %s", targetTs)
//	...
//
// and requires
//
//	assert.ElementsMatchf(t, certConfig.Org, csr.Subject.Org, "organizations not equal")
//	assert.Errorf(t, err, "Profile %s should not be valid", test.profile)
//	assert.Errorf(t, err, "test %s", test.testName)
//	assert.Truef(t, targetTs.Equal(ts), "the timestamp should be as expected (%s) but was %s", targetTs, ts)
type Formatter struct {
	checkFormatString bool
	requireFFuncs     bool
}

// NewFormatter constructs Formatter checker.
func NewFormatter() *Formatter {
	return &Formatter{
		checkFormatString: true,
		requireFFuncs:     false,
	}
}

func (Formatter) Name() string { return "formatter" }

func (checker *Formatter) SetCheckFormatString(v bool) *Formatter {
	checker.checkFormatString = v
	return checker
}

func (checker *Formatter) SetRequireFFuncs(v bool) *Formatter {
	checker.requireFFuncs = v
	return checker
}

func (checker Formatter) Check(pass *analysis.Pass, call *CallMeta) (result *analysis.Diagnostic) {
	if call.Fn.IsFmt {
		return checker.checkFmtAssertion(pass, call)
	}
	return checker.checkNotFmtAssertion(pass, call)
}

func (checker Formatter) checkNotFmtAssertion(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	msgAndArgsPos, ok := isPrintfLikeCall(pass, call, call.Fn.Signature)
	if !ok {
		return nil
	}

	fFunc := call.Fn.Name + "f"

	if msgAndArgsPos == len(call.ArgsRaw)-1 {
		msgAndArgs := call.ArgsRaw[msgAndArgsPos]
		if args, ok := isFmtSprintfCall(pass, msgAndArgs); ok {
			if checker.requireFFuncs {
				msg := fmt.Sprintf("remove unnecessary fmt.Sprintf and use %s.%s", call.SelectorXStr, fFunc)
				return newDiagnostic(checker.Name(), call, msg,
					newSuggestedFuncReplacement(call, fFunc, analysis.TextEdit{
						Pos:     msgAndArgs.Pos(),
						End:     msgAndArgs.End(),
						NewText: formatAsCallArgs(pass, args...),
					}),
				)
			}
			return newRemoveSprintfDiagnostic(pass, checker.Name(), call, msgAndArgs, args)
		}
	}

	if checker.requireFFuncs {
		return newUseFunctionDiagnostic(checker.Name(), call, fFunc, newSuggestedFuncReplacement(call, fFunc))
	}
	return nil
}

func (checker Formatter) checkFmtAssertion(pass *analysis.Pass, call *CallMeta) (result *analysis.Diagnostic) {
	formatPos := getMsgPosition(call.Fn.Signature)
	if formatPos < 0 {
		return nil
	}

	msg := call.ArgsRaw[formatPos]

	if formatPos == len(call.ArgsRaw)-1 {
		if args, ok := isFmtSprintfCall(pass, msg); ok {
			return newRemoveSprintfDiagnostic(pass, checker.Name(), call, msg, args)
		}
	}

	if checker.checkFormatString {
		report := pass.Report
		defer func() { pass.Report = report }()

		pass.Report = func(d analysis.Diagnostic) {
			result = newDiagnostic(checker.Name(), call, d.Message, nil)
		}

		format, err := strconv.Unquote(analysisutil.NodeString(pass.Fset, msg))
		if err != nil {
			return nil
		}
		printf.CheckPrintf(pass, call.Call, call.String(), format, formatPos)
	}
	return result
}

func isPrintfLikeCall(pass *analysis.Pass, call *CallMeta, sig *types.Signature) (int, bool) {
	msgAndArgsPos := getMsgAndArgsPosition(sig)
	if msgAndArgsPos < 0 {
		return -1, false
	}

	fmtFn := analysisutil.ObjectOf(pass.Pkg, testify.AssertPkgPath, call.Fn.Name+"f")
	if fmtFn == nil {
		// NOTE(a.telyshev): No formatted analogue of assertion.
		return -1, false
	}

	return msgAndArgsPos, len(call.ArgsRaw) > msgAndArgsPos
}

func getMsgAndArgsPosition(sig *types.Signature) int {
	params := sig.Params()
	if params.Len() < 1 {
		return -1
	}

	lastIdx := params.Len() - 1
	lastParam := params.At(lastIdx)

	_, isSlice := lastParam.Type().(*types.Slice)
	if lastParam.Name() == "msgAndArgs" && isSlice {
		return lastIdx
	}
	return -1
}

func getMsgPosition(sig *types.Signature) int {
	for i := 0; i < sig.Params().Len(); i++ {
		param := sig.Params().At(i)

		if b, ok := param.Type().(*types.Basic); ok && b.Kind() == types.String && param.Name() == "msg" {
			return i
		}
	}
	return -1
}

func isFmtSprintfCall(pass *analysis.Pass, expr ast.Expr) ([]ast.Expr, bool) {
	ce, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	se, ok := ce.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	sprintfObj := analysisutil.ObjectOf(pass.Pkg, "fmt", "Sprintf")
	if sprintfObj == nil {
		return nil, false
	}

	if !analysisutil.IsObj(pass.TypesInfo, se.Sel, sprintfObj) {
		return nil, false
	}

	return ce.Args, true
}
