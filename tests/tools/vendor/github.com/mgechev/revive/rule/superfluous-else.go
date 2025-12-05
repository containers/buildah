package rule

import (
	"fmt"
	"github.com/mgechev/revive/internal/ifelse"
	"github.com/mgechev/revive/lint"
)

// SuperfluousElseRule lints given else constructs.
type SuperfluousElseRule struct{}

// Apply applies the rule to given file.
func (e *SuperfluousElseRule) Apply(file *lint.File, args lint.Arguments) []lint.Failure {
	return ifelse.Apply(e, file.AST, ifelse.TargetElse, args)
}

// Name returns the rule name.
func (*SuperfluousElseRule) Name() string {
	return "superfluous-else"
}

// CheckIfElse evaluates the rule against an ifelse.Chain.
func (*SuperfluousElseRule) CheckIfElse(chain ifelse.Chain, args ifelse.Args) (failMsg string) {
	if !chain.If.Deviates() {
		// this rule only applies if the if-block deviates control flow
		return
	}

	if chain.HasPriorNonDeviating {
		// if we de-indent the "else" block then a previous branch
		// might flow into it, affecting program behaviour
		return
	}

	if chain.If.Returns() {
		// avoid overlapping with indent-error-flow
		return
	}

	if args.PreserveScope && !chain.AtBlockEnd && (chain.HasInitializer || chain.Else.HasDecls) {
		// avoid increasing variable scope
		return
	}

	return fmt.Sprintf("if block ends with %v, so drop this else and outdent its block", chain.If.LongString())
}
