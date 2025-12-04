package ifelse

import (
	"go/ast"
	"go/token"

	"github.com/mgechev/revive/lint"
)

// Rule is an interface for linters operating on if-else chains
type Rule interface {
	CheckIfElse(chain Chain, args Args) (failMsg string)
}

// Apply evaluates the given Rule on if-else chains found within the given AST,
// and returns the failures.
//
// Note that in if-else chain with multiple "if" blocks, only the *last* one is checked,
// that is to say, given:
//
//	if foo {
//	    ...
//	} else if bar {
//		...
//	} else {
//		...
//	}
//
// Only the block following "bar" is linted. This is because the rules that use this function
// do not presently have anything to say about earlier blocks in the chain.
func Apply(rule Rule, node ast.Node, target Target, args lint.Arguments) []lint.Failure {
	v := &visitor{rule: rule, target: target}
	for _, arg := range args {
		if arg == PreserveScope {
			v.args.PreserveScope = true
		}
	}
	ast.Walk(v, node)
	return v.failures
}

type visitor struct {
	failures []lint.Failure
	target   Target
	rule     Rule
	args     Args
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	block, ok := node.(*ast.BlockStmt)
	if !ok {
		return v
	}

	for i, stmt := range block.List {
		if ifStmt, ok := stmt.(*ast.IfStmt); ok {
			v.visitChain(ifStmt, Chain{AtBlockEnd: i == len(block.List)-1})
			continue
		}
		ast.Walk(v, stmt)
	}
	return nil
}

func (v *visitor) visitChain(ifStmt *ast.IfStmt, chain Chain) {
	// look for other if-else chains nested inside this if { } block
	ast.Walk(v, ifStmt.Body)

	if ifStmt.Else == nil {
		// no else branch
		return
	}

	if as, ok := ifStmt.Init.(*ast.AssignStmt); ok && as.Tok == token.DEFINE {
		chain.HasInitializer = true
	}
	chain.If = BlockBranch(ifStmt.Body)

	switch elseBlock := ifStmt.Else.(type) {
	case *ast.IfStmt:
		if !chain.If.Deviates() {
			chain.HasPriorNonDeviating = true
		}
		v.visitChain(elseBlock, chain)
	case *ast.BlockStmt:
		// look for other if-else chains nested inside this else { } block
		ast.Walk(v, elseBlock)

		chain.Else = BlockBranch(elseBlock)
		if failMsg := v.rule.CheckIfElse(chain, v.args); failMsg != "" {
			if chain.HasInitializer {
				// if statement has a := initializer, so we might need to move the assignment
				// onto its own line in case the body references it
				failMsg += " (move short variable declaration to its own line if necessary)"
			}
			v.failures = append(v.failures, lint.Failure{
				Confidence: 1,
				Node:       v.target.node(ifStmt),
				Failure:    failMsg,
			})
		}
	default:
		panic("invalid node type for else")
	}
}
