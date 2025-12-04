package checkers

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

func newUseFunctionDiagnostic(
	checker string,
	call *CallMeta,
	proposedFn string,
	fix *analysis.SuggestedFix,
) *analysis.Diagnostic {
	f := proposedFn
	if call.Fn.IsFmt {
		f += "f"
	}
	msg := fmt.Sprintf("use %s.%s", call.SelectorXStr, f)

	return newDiagnostic(checker, call, msg, fix)
}

func newRemoveSprintfDiagnostic(
	pass *analysis.Pass,
	checker string,
	call *CallMeta,
	sprintfPos analysis.Range,
	sprintfArgs []ast.Expr,
) *analysis.Diagnostic {
	return newDiagnostic(checker, call, "remove unnecessary fmt.Sprintf", &analysis.SuggestedFix{
		Message: "Remove `fmt.Sprintf`",
		TextEdits: []analysis.TextEdit{
			{
				Pos:     sprintfPos.Pos(),
				End:     sprintfPos.End(),
				NewText: formatAsCallArgs(pass, sprintfArgs...),
			},
		},
	})
}

func newDiagnostic(
	checker string,
	rng analysis.Range,
	msg string,
	fix *analysis.SuggestedFix,
) *analysis.Diagnostic {
	d := analysis.Diagnostic{
		Pos:      rng.Pos(),
		End:      rng.End(),
		Category: checker,
		Message:  checker + ": " + msg,
	}
	if fix != nil {
		d.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	return &d
}

func newSuggestedFuncReplacement(
	call *CallMeta,
	proposedFn string,
	additionalEdits ...analysis.TextEdit,
) *analysis.SuggestedFix {
	if call.Fn.IsFmt {
		proposedFn += "f"
	}
	return &analysis.SuggestedFix{
		Message: fmt.Sprintf("Replace `%s` with `%s`", call.Fn.Name, proposedFn),
		TextEdits: append([]analysis.TextEdit{
			{
				Pos:     call.Fn.Pos(),
				End:     call.Fn.End(),
				NewText: []byte(proposedFn),
			},
		}, additionalEdits...),
	}
}
