package prealloc

import (
	"fmt"
	"sync"

	"github.com/alexkohler/prealloc/pkg"
	"golang.org/x/tools/go/analysis"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/goanalysis"
	"github.com/golangci/golangci-lint/pkg/golinters/internal"
	"github.com/golangci/golangci-lint/pkg/lint/linter"
	"github.com/golangci/golangci-lint/pkg/result"
)

const linterName = "prealloc"

func New(settings *config.PreallocSettings) *goanalysis.Linter {
	var mu sync.Mutex
	var resIssues []goanalysis.Issue

	analyzer := &analysis.Analyzer{
		Name: linterName,
		Doc:  goanalysis.TheOnlyanalyzerDoc,
		Run: func(pass *analysis.Pass) (any, error) {
			issues := runPreAlloc(pass, settings)

			if len(issues) == 0 {
				return nil, nil
			}

			mu.Lock()
			resIssues = append(resIssues, issues...)
			mu.Unlock()

			return nil, nil
		},
	}

	return goanalysis.NewLinter(
		linterName,
		"Finds slice declarations that could potentially be pre-allocated",
		[]*analysis.Analyzer{analyzer},
		nil,
	).WithIssuesReporter(func(*linter.Context) []goanalysis.Issue {
		return resIssues
	}).WithLoadMode(goanalysis.LoadModeSyntax)
}

func runPreAlloc(pass *analysis.Pass, settings *config.PreallocSettings) []goanalysis.Issue {
	var issues []goanalysis.Issue

	hints := pkg.Check(pass.Files, settings.Simple, settings.RangeLoops, settings.ForLoops)

	for _, hint := range hints {
		issues = append(issues, goanalysis.NewIssue(&result.Issue{
			Pos:        pass.Fset.Position(hint.Pos),
			Text:       fmt.Sprintf("Consider pre-allocating %s", internal.FormatCode(hint.DeclaredSliceName, nil)),
			FromLinter: linterName,
		}, pass))
	}

	return issues
}
