//go:build !go1.18
// +build !go1.18

package analyzer

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func (t thelper) buildFuzzCheckFuncOpts(pass *analysis.Pass, ctxType types.Type) (checkFuncOpts, bool) {
	return checkFuncOpts{}, true
}
