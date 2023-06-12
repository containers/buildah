package exportloopref

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:             "exportloopref",
	Doc:              "checks for pointers to enclosing loop variables",
	Run:              run,
	RunDespiteErrors: true,
	Requires:         []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	search := &Searcher{
		LoopVars:  map[token.Pos]struct{}{},
		LocalVars: map[token.Pos]map[token.Pos]struct{}{},
		Pass:      pass,
	}

	nodeFilter := []ast.Node{
		(*ast.RangeStmt)(nil),
		(*ast.ForStmt)(nil),
		(*ast.DeclStmt)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.UnaryExpr)(nil),
	}

	inspect.WithStack(nodeFilter, search.CheckAndReport)

	return nil, nil
}

type Searcher struct {
	// LoopVars is positions that loop-variables are declared like below.
	//  - for <KEY>, <VALUE> := range ...
	//  - for <VALUE> := <INIT>; <CONDITION>; <INCREMENT>
	LoopVars map[token.Pos]struct{}
	// LocalVars is positions of loops and the variables declared in them.
	// Use this to determine if a point assignment is an export outside the loop.
	LocalVars map[token.Pos]map[token.Pos]struct{}

	Pass *analysis.Pass
}

// CheckAndReport inspects each node with stack.
// It is implemented as the I/F of the "golang.org/x/tools/go/analysis/passes/inspect".Analysis.WithStack.
func (s *Searcher) CheckAndReport(n ast.Node, push bool, stack []ast.Node) bool {
	id, insert, digg := s.Check(n, stack)
	if id == nil {
		// no prob.
		return digg
	}

	// suggests fix
	var suggest []analysis.SuggestedFix
	if insert != token.NoPos {
		suggest = []analysis.SuggestedFix{{
			Message: fmt.Sprintf("loop variable %s should be pinned", id.Name),
			TextEdits: []analysis.TextEdit{{
				Pos:     insert,
				End:     insert,
				NewText: []byte(fmt.Sprintf("%[1]s := %[1]s\n", id.Name)),
			}},
		}}
	}

	// report a diagnostic
	d := analysis.Diagnostic{Pos: id.Pos(),
		End:            id.End(),
		Message:        fmt.Sprintf("exporting a pointer for the loop variable %s", id.Name),
		Category:       "exportloopref",
		SuggestedFixes: suggest,
	}
	s.Pass.Report(d)
	return digg
}

// Check each node and stack, whether it exports loop variables or not.
// Finding export, report the *ast.Ident of exported loop variable,
// and token.Pos to insert assignment to fix the diagnostic.
func (s *Searcher) Check(n ast.Node, stack []ast.Node) (loopVar *ast.Ident, insertPos token.Pos, digg bool) {
	switch typed := n.(type) {
	case *ast.RangeStmt:
		s.parseRangeStmt(typed)
	case *ast.ForStmt:
		s.parseForStmt(typed)
	case *ast.DeclStmt:
		s.parseDeclStmt(typed, stack)
	case *ast.AssignStmt:
		s.parseAssignStmt(typed, stack)

	case *ast.UnaryExpr:
		return s.checkUnaryExpr(typed, stack)
	}
	return nil, token.NoPos, true
}

// parseRangeStmt will check range statement (i.e. `for <KEY>, <VALUE> := range ...`),
// and collect positions of <KEY> and <VALUE>.
func (s *Searcher) parseRangeStmt(n *ast.RangeStmt) {
	s.storeLoopVars(n.Key)
	s.storeLoopVars(n.Value)
}

// parseForStmt will check for statement (i.e. `for <VALUE> := <INIT>; <CONDITION>; <INCREMENT>`),
// and collect positions of <VALUE>.
func (s *Searcher) parseForStmt(n *ast.ForStmt) {
	switch post := n.Post.(type) {
	case *ast.AssignStmt:
		// e.g. for p = head; p != nil; p = p.next
		for _, lhs := range post.Lhs {
			s.storeLoopVars(lhs)
		}
	case *ast.IncDecStmt:
		// e.g. for i := 0; i < n; i++
		s.storeLoopVars(post.X)
	}
}

func (s *Searcher) storeLoopVars(expr ast.Expr) {
	if id, ok := expr.(*ast.Ident); ok {
		s.LoopVars[id.Pos()] = struct{}{}
	}
}

// parseDeclStmt will parse declaring statement (i.e. `var`, `type`, `const`),
// and store the position if it is "var" declaration and is in any loop.
func (s *Searcher) parseDeclStmt(n *ast.DeclStmt, stack []ast.Node) {
	genDecl, ok := n.Decl.(*ast.GenDecl)
	if !ok {
		// (dead branch)
		// if the Decl is not GenDecl (i.e. `var`, `type` or `const` statement), it is ignored
		return
	}
	if genDecl.Tok != token.VAR {
		// if the Decl is not `var` (may be `type` or `const`), it is ignored
		return
	}

	loop, _ := s.innermostLoop(stack)
	if loop == nil {
		return
	}

	// Register declared variables
	for _, spec := range genDecl.Specs {
		for _, name := range spec.(*ast.ValueSpec).Names {
			s.storeLocalVar(loop, name)
		}
	}
}

// parseDeclStmt will parse assignment statement (i.e. `<VAR> = <VALUE>`),
// and store the position if it is .
func (s *Searcher) parseAssignStmt(n *ast.AssignStmt, stack []ast.Node) {
	if n.Tok != token.DEFINE {
		// if the statement is simple assignment (without definement), it is ignored
		return
	}

	loop, _ := s.innermostLoop(stack)
	if loop == nil {
		return
	}

	// Find statements declaring local variable
	for _, h := range n.Lhs {
		s.storeLocalVar(loop, h)
	}
}

func (s *Searcher) storeLocalVar(loop ast.Node, expr ast.Expr) {
	loopPos := loop.Pos()
	id, ok := expr.(*ast.Ident)
	if !ok {
		return
	}
	vars, ok := s.LocalVars[loopPos]
	if !ok {
		vars = map[token.Pos]struct{}{}
	}
	vars[id.Obj.Pos()] = struct{}{}
	s.LocalVars[loopPos] = vars
}

func insertionPosition(block *ast.BlockStmt) token.Pos {
	if len(block.List) > 0 {
		return block.List[0].Pos()
	}
	return token.NoPos
}

func (s *Searcher) innermostLoop(stack []ast.Node) (ast.Node, token.Pos) {
	for i := len(stack) - 1; i >= 0; i-- {
		switch typed := stack[i].(type) {
		case *ast.RangeStmt:
			return typed, insertionPosition(typed.Body)
		case *ast.ForStmt:
			return typed, insertionPosition(typed.Body)
		}
	}
	return nil, token.NoPos
}

// checkUnaryExpr check unary expression (i.e. <OPERATOR><VAR> like `-x`, `*p` or `&v`) and stack.
// THIS IS THE ESSENTIAL PART OF THIS PARSER.
func (s *Searcher) checkUnaryExpr(n *ast.UnaryExpr, stack []ast.Node) (*ast.Ident, token.Pos, bool) {
	if n.Op != token.AND {
		return nil, token.NoPos, true
	}

	loop, insert := s.innermostLoop(stack)
	if loop == nil {
		return nil, token.NoPos, true
	}

	// Get identity of the referred item
	id := s.getIdentity(n.X)
	if id == nil {
		return nil, token.NoPos, true
	}

	// If the identity is not the loop statement variable,
	// it will not be reported.
	if _, isDecl := s.LoopVars[id.Obj.Pos()]; !isDecl {
		return nil, token.NoPos, true
	}

	// check stack append(), []X{}, map[Type]X{}, Struct{}, &Struct{}, X.(Type), (X)
	// in the <outer> =
	var mayRHPos token.Pos
	for i := len(stack) - 2; i >= 0; i-- {
		switch typed := stack[i].(type) {
		case (*ast.UnaryExpr):
			// noop
		case (*ast.CompositeLit):
			// noop
		case (*ast.KeyValueExpr):
			// noop
		case (*ast.CallExpr):
			fun, ok := typed.Fun.(*ast.Ident)
			if !ok {
				return nil, token.NoPos, false // it's calling a function other of `append`. It cannot be checked
			}

			if fun.Name != "append" {
				return nil, token.NoPos, false // it's calling a function other of `append`. It cannot be checked
			}

		case (*ast.AssignStmt):
			if len(typed.Rhs) != len(typed.Lhs) {
				return nil, token.NoPos, false // dead logic
			}

			// search x where Rhs[x].Pos() == mayRHPos
			var index int
			for ri, rh := range typed.Rhs {
				if rh.Pos() == mayRHPos {
					index = ri
					break
				}
			}

			// check Lhs[x] is not local variable
			lh := typed.Lhs[index]
			isVar := s.isVar(loop, lh)
			if !isVar {
				return id, insert, false
			}

			return nil, token.NoPos, true
		default:
			// Other statement is not able to be checked.
			return nil, token.NoPos, false
		}

		// memory an expr that may be right-hand in the AssignStmt
		mayRHPos = stack[i].Pos()
	}
	return nil, token.NoPos, true
}

func (s *Searcher) isVar(loop ast.Node, expr ast.Expr) bool {
	vars := s.LocalVars[loop.Pos()] // map[token.Pos]struct{}
	if vars == nil {
		return false
	}
	switch typed := expr.(type) {
	case (*ast.Ident):
		if typed.Obj == nil {
			return false // global var in another file (ref: #13)
		}
		_, isVar := vars[typed.Obj.Pos()]
		return isVar
	case (*ast.IndexExpr): // like X[Y], check X
		return s.isVar(loop, typed.X)
	case (*ast.SelectorExpr): // like X.Y, check X
		return s.isVar(loop, typed.X)
	}
	return false
}

// Get variable identity
func (s *Searcher) getIdentity(expr ast.Expr) *ast.Ident {
	switch typed := expr.(type) {
	case *ast.SelectorExpr:
		// Ignore if the parent is pointer ref (fix for #2)
		if _, ok := s.Pass.TypesInfo.Types[typed.X].Type.(*types.Pointer); ok {
			return nil
		}

		// Get parent identity; i.e. `a.b` of the `a.b.c`.
		return s.getIdentity(typed.X)

	case *ast.Ident:
		// Get simple identity; i.e. `a` of the `a`.
		if typed.Obj == nil {
			return nil
		}
		return typed
	}
	return nil
}
