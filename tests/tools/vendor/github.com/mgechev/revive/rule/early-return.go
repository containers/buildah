package rule

import (
	"fmt"

	"github.com/mgechev/revive/internal/ifelse"
	"github.com/mgechev/revive/lint"
)

// EarlyReturnRule finds opportunities to reduce nesting by inverting
// the condition of an "if" block.
type EarlyReturnRule struct{}

// Apply applies the rule to given file.
func (e *EarlyReturnRule) Apply(file *lint.File, args lint.Arguments) []lint.Failure {
	return ifelse.Apply(e, file.AST, ifelse.TargetIf, args)
}

// Name returns the rule name.
func (*EarlyReturnRule) Name() string {
	return "early-return"
}

// CheckIfElse evaluates the rule against an ifelse.Chain.
func (*EarlyReturnRule) CheckIfElse(chain ifelse.Chain, args ifelse.Args) (failMsg string) {
	if !chain.Else.Deviates() {
		// this rule only applies if the else-block deviates control flow
		return
	}

	if chain.HasPriorNonDeviating && !chain.If.IsEmpty() {
		// if we de-indent this block then a previous branch
		// might flow into it, affecting program behaviour
		return
	}

	if chain.If.Deviates() {
		// avoid overlapping with superfluous-else
		return
	}

	if args.PreserveScope && !chain.AtBlockEnd && (chain.HasInitializer || chain.If.HasDecls) {
		// avoid increasing variable scope
		return
	}

	if chain.If.IsEmpty() {
		return fmt.Sprintf("if c { } else { %[1]v } can be simplified to if !c { %[1]v }", chain.Else)
	}
	return fmt.Sprintf("if c { ... } else { %[1]v } can be simplified to if !c { %[1]v } ...", chain.Else)
}
