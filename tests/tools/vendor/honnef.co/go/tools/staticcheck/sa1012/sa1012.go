package sa1012

import (
	"go/ast"
	"go/types"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/types/typeutil"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var SCAnalyzer = lint.InitializeAnalyzer(&lint.Analyzer{
	Analyzer: &analysis.Analyzer{
		Name:     "SA1012",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	},
	Doc: &lint.RawDocumentation{
		Title:    `A nil \'context.Context\' is being passed to a function, consider using \'context.TODO\' instead`,
		Since:    "2017.1",
		Severity: lint.SeverityWarning,
		MergeIf:  lint.MergeIfAny,
	},
})

var Analyzer = SCAnalyzer.Analyzer

var checkNilContextQ = pattern.MustParse(`(CallExpr fun@(Symbol _) (Builtin "nil"):_)`)

func run(pass *analysis.Pass) (interface{}, error) {
	todo := &ast.CallExpr{
		Fun: edit.Selector("context", "TODO"),
	}
	bg := &ast.CallExpr{
		Fun: edit.Selector("context", "Background"),
	}
	fn := func(node ast.Node) {
		m, ok := code.Match(pass, checkNilContextQ, node)
		if !ok {
			return
		}

		call := node.(*ast.CallExpr)
		fun, ok := m.State["fun"].(*types.Func)
		if !ok {
			// it might also be a builtin
			return
		}
		sig := fun.Type().(*types.Signature)
		if sig.Params().Len() == 0 {
			// Our CallExpr might've matched a method expression, like
			// (*T).Foo(nil) – here, nil isn't the first argument of
			// the Foo method, but the method receiver.
			return
		}
		if !typeutil.IsTypeWithName(sig.Params().At(0).Type(), "context.Context") {
			return
		}
		report.Report(pass, call.Args[0],
			"do not pass a nil Context, even if a function permits it; pass context.TODO if you are unsure about which Context to use", report.Fixes(
				edit.Fix("use context.TODO", edit.ReplaceWithNode(pass.Fset, call.Args[0], todo)),
				edit.Fix("use context.Background", edit.ReplaceWithNode(pass.Fset, call.Args[0], bg))))
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}
