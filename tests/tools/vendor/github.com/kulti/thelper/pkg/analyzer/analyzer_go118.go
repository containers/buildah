//go:build go1.18
// +build go1.18

package analyzer

import (
	"go/types"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
)

func (t thelper) buildFuzzCheckFuncOpts(pass *analysis.Pass, ctxType types.Type) (checkFuncOpts, bool) {
	fObj := analysisutil.ObjectOf(pass, "testing", "F")
	if fObj == nil {
		return checkFuncOpts{}, false
	}

	fHelper, _, _ := types.LookupFieldOrMethod(fObj.Type(), true, fObj.Pkg(), "Helper")
	if fHelper == nil {
		return checkFuncOpts{}, false
	}

	tFuzz, _, _ := types.LookupFieldOrMethod(fObj.Type(), true, fObj.Pkg(), "Fuzz")
	if tFuzz == nil {
		return checkFuncOpts{}, false
	}

	return checkFuncOpts{
		skipPrefix: "Fuzz",
		varName:    "f",
		fnHelper:   fHelper,
		subRun:     tFuzz,
		hpType:     types.NewPointer(fObj.Type()),
		ctxType:    ctxType,
		checkBegin: t.enabledChecks.Enabled(checkFBegin),
		checkFirst: t.enabledChecks.Enabled(checkFFirst),
		checkName:  t.enabledChecks.Enabled(checkFName),
	}, true
}
