package golinters

import (
	"github.com/nunnatsa/ginkgolinter"
	"golang.org/x/tools/go/analysis"

	"github.com/golangci/golangci-lint/pkg/config"
	"github.com/golangci/golangci-lint/pkg/golinters/goanalysis"
)

func NewGinkgoLinter(cfg *config.GinkgoLinterSettings) *goanalysis.Linter {
	a := ginkgolinter.NewAnalyzer()

	cfgMap := make(map[string]map[string]interface{})
	if cfg != nil {
		cfgMap[a.Name] = map[string]interface{}{
			"suppress-len-assertion": cfg.SuppressLenAssertion,
			"suppress-nil-assertion": cfg.SuppressNilAssertion,
			"suppress-err-assertion": cfg.SuppressErrAssertion,
		}
	}

	return goanalysis.NewLinter(
		a.Name,
		a.Doc,
		[]*analysis.Analyzer{a},
		cfgMap,
	).WithLoadMode(goanalysis.LoadModeTypesInfo)
}
