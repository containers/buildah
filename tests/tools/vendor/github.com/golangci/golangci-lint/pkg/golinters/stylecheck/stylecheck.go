package stylecheck

import (
	"golang.org/x/tools/go/analysis"
	scconfig "honnef.co/go/tools/config"
	"honnef.co/go/tools/stylecheck"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/goanalysis"
	"github.com/golangci/golangci-lint/pkg/golinters/internal"
)

func New(settings *config.StaticCheckSettings) *goanalysis.Linter {
	cfg := internal.StaticCheckConfig(settings)

	// `scconfig.Analyzer` is a singleton, then it's not possible to have more than one instance for all staticcheck "sub-linters".
	// When we will merge the 4 "sub-linters", the problem will disappear: https://github.com/golangci/golangci-lint/issues/357
	// Currently only stylecheck analyzer has a configuration in staticcheck.
	scconfig.Analyzer.Run = func(_ *analysis.Pass) (any, error) {
		return cfg, nil
	}

	analyzers := internal.SetupStaticCheckAnalyzers(stylecheck.Analyzers, cfg.Checks)

	return goanalysis.NewLinter(
		"stylecheck",
		"Stylecheck is a replacement for golint",
		analyzers,
		nil,
	).WithLoadMode(goanalysis.LoadModeTypesInfo)
}
