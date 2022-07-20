package golinters

import (
	"sync"

	"github.com/ashanbrown/forbidigo/forbidigo"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/analysis"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/golinters/goanalysis"
	"github.com/golangci/golangci-lint/pkg/lint/linter"
	"github.com/golangci/golangci-lint/pkg/result"
)

const forbidigoName = "forbidigo"

//nolint:dupl
func NewForbidigo(settings *config.ForbidigoSettings) *goanalysis.Linter {
	var mu sync.Mutex
	var resIssues []goanalysis.Issue

	analyzer := &analysis.Analyzer{
		Name: forbidigoName,
		Doc:  goanalysis.TheOnlyanalyzerDoc,
		Run: func(pass *analysis.Pass) (interface{}, error) {
			issues, err := runForbidigo(pass, settings)
			if err != nil {
				return nil, err
			}

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
		forbidigoName,
		"Forbids identifiers",
		[]*analysis.Analyzer{analyzer},
		nil,
	).WithIssuesReporter(func(*linter.Context) []goanalysis.Issue {
		return resIssues
	}).WithLoadMode(goanalysis.LoadModeSyntax)
}

func runForbidigo(pass *analysis.Pass, settings *config.ForbidigoSettings) ([]goanalysis.Issue, error) {
	options := []forbidigo.Option{
		forbidigo.OptionExcludeGodocExamples(settings.ExcludeGodocExamples),
		// disable "//permit" directives so only "//nolint" directives matters within golangci-lint
		forbidigo.OptionIgnorePermitDirectives(true),
	}

	forbid, err := forbidigo.NewLinter(settings.Forbid, options...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create linter %q", forbidigoName)
	}

	var issues []goanalysis.Issue
	for _, file := range pass.Files {
		hints, err := forbid.Run(pass.Fset, file)
		if err != nil {
			return nil, errors.Wrapf(err, "forbidigo linter failed on file %q", file.Name.String())
		}

		for _, hint := range hints {
			issues = append(issues, goanalysis.NewIssue(&result.Issue{
				Pos:        hint.Position(),
				Text:       hint.Details(),
				FromLinter: forbidigoName,
			}, pass))
		}
	}

	return issues, nil
}
