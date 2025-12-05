package ginkgolinter

import (
	"flag"
	"fmt"

	"golang.org/x/tools/go/analysis"

	"github.com/nunnatsa/ginkgolinter/linter"
	"github.com/nunnatsa/ginkgolinter/types"
	"github.com/nunnatsa/ginkgolinter/version"
)

// NewAnalyzerWithConfig returns an Analyzer.
func NewAnalyzerWithConfig(config *types.Config) *analysis.Analyzer {
	theLinter := linter.NewGinkgoLinter(config)

	return &analysis.Analyzer{
		Name: "ginkgolinter",
		Doc:  fmt.Sprintf(doc, version.Version()),
		Run:  theLinter.Run,
	}
}

// NewAnalyzer returns an Analyzer - the package interface with nogo
func NewAnalyzer() *analysis.Analyzer {
	config := &types.Config{
		SuppressLen:     false,
		SuppressNil:     false,
		SuppressErr:     false,
		SuppressCompare: false,
		ForbidFocus:     false,
		AllowHaveLen0:   false,
		ForceExpectTo:   false,
	}

	a := NewAnalyzerWithConfig(config)

	var ignored bool
	a.Flags.Init("ginkgolinter", flag.ExitOnError)
	a.Flags.Var(&config.SuppressLen, "suppress-len-assertion", "Suppress warning for wrong length assertions")
	a.Flags.Var(&config.SuppressNil, "suppress-nil-assertion", "Suppress warning for wrong nil assertions")
	a.Flags.Var(&config.SuppressErr, "suppress-err-assertion", "Suppress warning for wrong error assertions")
	a.Flags.Var(&config.SuppressCompare, "suppress-compare-assertion", "Suppress warning for wrong comparison assertions")
	a.Flags.Var(&config.SuppressAsync, "suppress-async-assertion", "Suppress warning for function call in async assertion, like Eventually")
	a.Flags.Var(&config.ValidateAsyncIntervals, "validate-async-intervals", "best effort validation of async intervals (timeout and polling); ignored the suppress-async-assertion flag is true")
	a.Flags.Var(&config.SuppressTypeCompare, "suppress-type-compare-assertion", "Suppress warning for comparing values from different types, like int32 and uint32")
	a.Flags.Var(&config.AllowHaveLen0, "allow-havelen-0", "Do not warn for HaveLen(0); default = false")
	a.Flags.Var(&config.ForceExpectTo, "force-expect-to", "force using `Expect` with `To`, `ToNot` or `NotTo`. reject using `Expect` with `Should` or `ShouldNot`; default = false (not forced)")
	a.Flags.BoolVar(&ignored, "suppress-focus-container", true, "Suppress warning for ginkgo focus containers like FDescribe, FContext, FWhen or FIt. Deprecated and ignored: use --forbid-focus-container instead")
	a.Flags.Var(&config.ForbidFocus, "forbid-focus-container", "trigger a warning for ginkgo focus containers like FDescribe, FContext, FWhen or FIt; default = false.")
	a.Flags.Var(&config.ForbidSpecPollution, "forbid-spec-pollution", "trigger a warning for variable assignments in ginkgo containers like Describe, Context and When, instead of in BeforeEach(); default = false.")

	return a
}

// Analyzer is the interface to go_vet
var Analyzer = NewAnalyzer()
