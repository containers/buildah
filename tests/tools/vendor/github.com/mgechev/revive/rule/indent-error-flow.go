package rule

import (
	"github.com/mgechev/revive/internal/ifelse"
	"github.com/mgechev/revive/lint"
)

// IndentErrorFlowRule lints given else constructs.
type IndentErrorFlowRule struct{}

// Apply applies the rule to given file.
func (e *IndentErrorFlowRule) Apply(file *lint.File, args lint.Arguments) []lint.Failure {
	return ifelse.Apply(e, file.AST, ifelse.TargetElse, args)
}

// Name returns the rule name.
func (*IndentErrorFlowRule) Name() string {
	return "indent-error-flow"
}

// CheckIfElse evaluates the rule against an ifelse.Chain.
func (*IndentErrorFlowRule) CheckIfElse(chain ifelse.Chain, args ifelse.Args) (failMsg string) {
	if !chain.If.Deviates() {
		// this rule only applies if the if-block deviates control flow
		return
	}

	if chain.HasPriorNonDeviating {
		// if we de-indent the "else" block then a previous branch
		// might flow into it, affecting program behaviour
		return
	}

	if !chain.If.Returns() {
		// avoid overlapping with superfluous-else
		return
	}

	if args.PreserveScope && !chain.AtBlockEnd && (chain.HasInitializer || chain.Else.HasDecls) {
		// avoid increasing variable scope
		return
	}

	return "if block ends with a return statement, so drop this else and outdent its block"
}
