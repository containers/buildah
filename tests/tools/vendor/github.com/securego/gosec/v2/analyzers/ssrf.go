// (c) Copyright gosec's authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analyzers

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/securego/gosec/v2/issue"
)

func newSSRFAnalyzer(id string, description string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     id,
		Doc:      description,
		Run:      runSSRF,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

func runSSRF(pass *analysis.Pass) (interface{}, error) {
	ssaResult, err := getSSAResult(pass)
	if err != nil {
		return nil, err
	}
	// TODO: implement the analysis
	for _, fn := range ssaResult.SSA.SrcFuncs {
		for _, block := range fn.DomPreorder() {
			for _, instr := range block.Instrs {
				switch instr := instr.(type) {
				case *ssa.Call:
					callee := instr.Call.StaticCallee()
					if callee != nil {
						ssaResult.Logger.Printf("callee: %s\n", callee)
						return newIssue(pass.Analyzer.Name,
							"not implemeted",
							pass.Fset, instr.Call.Pos(), issue.Low, issue.High), nil
					}
				}
			}
		}
	}
	return nil, nil
}
