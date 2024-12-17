package analyzer

import (
	"fmt"
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/Antonboom/testifylint/internal/analysisutil"
	"github.com/Antonboom/testifylint/internal/checkers"
	"github.com/Antonboom/testifylint/internal/config"
	"github.com/Antonboom/testifylint/internal/testify"
)

const (
	name = "testifylint"
	doc  = "Checks usage of " + testify.ModulePath + "."
	url  = "https://github.com/antonboom/" + name
)

// New returns a new instance of testifylint analyzer.
func New() *analysis.Analyzer {
	cfg := config.NewDefault()

	analyzer := &analysis.Analyzer{
		Name: name,
		Doc:  doc,
		URL:  url,
		Run: func(pass *analysis.Pass) (any, error) {
			regularCheckers, advancedCheckers, err := newCheckers(cfg)
			if err != nil {
				return nil, fmt.Errorf("build checkers: %v", err)
			}

			tl := &testifyLint{
				regularCheckers:  regularCheckers,
				advancedCheckers: advancedCheckers,
			}
			return tl.run(pass)
		},
	}
	config.BindToFlags(&cfg, &analyzer.Flags)

	return analyzer
}

type testifyLint struct {
	regularCheckers  []checkers.RegularChecker
	advancedCheckers []checkers.AdvancedChecker
}

func (tl *testifyLint) run(pass *analysis.Pass) (any, error) {
	filesToAnalysis := make([]*ast.File, 0, len(pass.Files))
	for _, f := range pass.Files {
		if !analysisutil.Imports(f, testify.AssertPkgPath, testify.RequirePkgPath, testify.SuitePkgPath) {
			continue
		}
		filesToAnalysis = append(filesToAnalysis, f)
	}

	insp := inspector.New(filesToAnalysis)

	// Regular checkers.
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
		tl.regularCheck(pass, node.(*ast.CallExpr))
	})

	// Advanced checkers.
	for _, ch := range tl.advancedCheckers {
		for _, d := range ch.Check(pass, insp) {
			pass.Report(d)
		}
	}

	return nil, nil
}

func (tl *testifyLint) regularCheck(pass *analysis.Pass, ce *ast.CallExpr) {
	call := checkers.NewCallMeta(pass, ce)
	if nil == call {
		return
	}

	for _, ch := range tl.regularCheckers {
		if d := ch.Check(pass, call); d != nil {
			pass.Report(*d)
			// NOTE(a.telyshev): I'm not interested in multiple diagnostics per assertion.
			// This simplifies the code and also makes the linter more efficient.
			return
		}
	}
}
