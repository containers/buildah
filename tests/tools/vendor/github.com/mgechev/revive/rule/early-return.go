package rule

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/mgechev/revive/lint"
)

// EarlyReturnRule finds opportunities to reduce nesting by inverting
// the condition of an "if" block.
type EarlyReturnRule struct{}

// Apply applies the rule to given file.
func (*EarlyReturnRule) Apply(file *lint.File, _ lint.Arguments) []lint.Failure {
	var failures []lint.Failure

	onFailure := func(failure lint.Failure) {
		failures = append(failures, failure)
	}

	w := lintEarlyReturnRule{onFailure: onFailure}
	ast.Walk(w, file.AST)
	return failures
}

// Name returns the rule name.
func (*EarlyReturnRule) Name() string {
	return "early-return"
}

type lintEarlyReturnRule struct {
	onFailure func(lint.Failure)
}

func (w lintEarlyReturnRule) Visit(node ast.Node) ast.Visitor {
	ifStmt, ok := node.(*ast.IfStmt)
	if !ok {
		return w
	}

	w.visitIf(ifStmt, false, false)
	return nil
}

func (w lintEarlyReturnRule) visitIf(ifStmt *ast.IfStmt, hasNonReturnBranch, hasIfInitializer bool) {
	// look for other if-else chains nested inside this if { } block
	ast.Walk(w, ifStmt.Body)

	if ifStmt.Else == nil {
		// no else branch
		return
	}

	if as, ok := ifStmt.Init.(*ast.AssignStmt); ok && as.Tok == token.DEFINE {
		hasIfInitializer = true
	}
	bodyFlow := w.branchFlow(ifStmt.Body)

	switch elseBlock := ifStmt.Else.(type) {
	case *ast.IfStmt:
		if bodyFlow.canFlowIntoNext() {
			hasNonReturnBranch = true
		}
		w.visitIf(elseBlock, hasNonReturnBranch, hasIfInitializer)

	case *ast.BlockStmt:
		// look for other if-else chains nested inside this else { } block
		ast.Walk(w, elseBlock)

		if hasNonReturnBranch && bodyFlow != branchFlowEmpty {
			// if we de-indent this block then a previous branch
			// might flow into it, affecting program behaviour
			return
		}

		if !bodyFlow.canFlowIntoNext() {
			// avoid overlapping with superfluous-else
			return
		}

		elseFlow := w.branchFlow(elseBlock)
		if !elseFlow.canFlowIntoNext() {
			failMsg := fmt.Sprintf("if c {%[1]s } else {%[2]s } can be simplified to if !c {%[2]s }%[1]s",
				bodyFlow, elseFlow)

			if hasIfInitializer {
				// if statement has a := initializer, so we might need to move the assignment
				// onto its own line in case the body references it
				failMsg += " (move short variable declaration to its own line if necessary)"
			}

			w.onFailure(lint.Failure{
				Confidence: 1,
				Node:       ifStmt,
				Failure:    failMsg,
			})
		}

	default:
		panic("invalid node type for else")
	}
}

type branchFlowKind int

const (
	branchFlowEmpty branchFlowKind = iota
	branchFlowReturn
	branchFlowPanic
	branchFlowContinue
	branchFlowBreak
	branchFlowGoto
	branchFlowRegular
)

func (w lintEarlyReturnRule) branchFlow(block *ast.BlockStmt) branchFlowKind {
	blockLen := len(block.List)
	if blockLen == 0 {
		return branchFlowEmpty
	}

	switch stmt := block.List[blockLen-1].(type) {
	case *ast.ReturnStmt:
		return branchFlowReturn
	case *ast.BlockStmt:
		return w.branchFlow(stmt)
	case *ast.BranchStmt:
		switch stmt.Tok {
		case token.BREAK:
			return branchFlowBreak
		case token.CONTINUE:
			return branchFlowContinue
		case token.GOTO:
			return branchFlowGoto
		}
	case *ast.ExprStmt:
		if call, ok := stmt.X.(*ast.CallExpr); ok && isIdent(call.Fun, "panic") {
			return branchFlowPanic
		}
	}

	return branchFlowRegular
}

// Whether this branch's control can flow into the next statement following the if-else chain
func (k branchFlowKind) canFlowIntoNext() bool {
	switch k {
	case branchFlowReturn, branchFlowPanic, branchFlowContinue, branchFlowBreak, branchFlowGoto:
		return false
	default:
		return true
	}
}

func (k branchFlowKind) String() string {
	switch k {
	case branchFlowEmpty:
		return ""
	case branchFlowReturn:
		return " ... return"
	case branchFlowPanic:
		return " ... panic()"
	case branchFlowContinue:
		return " ... continue"
	case branchFlowBreak:
		return " ... break"
	case branchFlowGoto:
		return " ... goto"
	case branchFlowRegular:
		return " ..."
	default:
		panic("invalid kind")
	}
}
