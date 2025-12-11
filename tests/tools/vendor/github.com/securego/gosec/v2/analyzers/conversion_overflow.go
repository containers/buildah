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
	"fmt"
	"regexp"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/securego/gosec/v2/issue"
)

func newConversionOverflowAnalyzer(id string, description string) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     id,
		Doc:      description,
		Run:      runConversionOverflow,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

func runConversionOverflow(pass *analysis.Pass) (interface{}, error) {
	ssaResult, err := getSSAResult(pass)
	if err != nil {
		return nil, fmt.Errorf("building ssa representation: %w", err)
	}

	issues := []*issue.Issue{}
	for _, mcall := range ssaResult.SSA.SrcFuncs {
		for _, block := range mcall.DomPreorder() {
			for _, instr := range block.Instrs {
				switch instr := instr.(type) {
				case *ssa.Convert:
					src := instr.X.Type().Underlying().String()
					dst := instr.Type().Underlying().String()
					if isIntOverflow(src, dst) {
						issue := newIssue(pass.Analyzer.Name,
							fmt.Sprintf("integer overflow conversion %s -> %s", src, dst),
							pass.Fset,
							instr.Pos(),
							issue.High,
							issue.Medium,
						)
						issues = append(issues, issue)
					}
				}
			}
		}
	}

	if len(issues) > 0 {
		return issues, nil
	}
	return nil, nil
}

type integer struct {
	signed bool
	size   int
}

func parseIntType(intType string) (integer, error) {
	re := regexp.MustCompile(`(?P<type>u?int)(?P<size>\d{1,2})?`)
	matches := re.FindStringSubmatch(intType)
	if matches == nil {
		return integer{}, fmt.Errorf("no integer type match found for %s", intType)
	}

	it := matches[re.SubexpIndex("type")]
	is := matches[re.SubexpIndex("size")]

	signed := false
	if it == "int" {
		signed = true
	}

	// use default system int type in case size is not present in the type
	intSize := strconv.IntSize
	if is != "" {
		var err error
		intSize, err = strconv.Atoi(is)
		if err != nil {
			return integer{}, fmt.Errorf("failed to parse the integer type size: %w", err)
		}
	}

	return integer{signed: signed, size: intSize}, nil
}

func isIntOverflow(src string, dst string) bool {
	srcInt, err := parseIntType(src)
	if err != nil {
		return false
	}

	dstInt, err := parseIntType(dst)
	if err != nil {
		return false
	}

	// converting uint to int of the same size or smaller might lead to overflow
	if !srcInt.signed && dstInt.signed && dstInt.size <= srcInt.size {
		return true
	}
	// converting uint to unit of a smaller size might lead to overflow
	if !srcInt.signed && !dstInt.signed && dstInt.size < srcInt.size {
		return true
	}
	// converting int to int of a smaller size might lead to overflow
	if srcInt.signed && dstInt.signed && dstInt.size < srcInt.size {
		return true
	}
	// converting int to uint of a smaller size might lead to overflow
	if srcInt.signed && !dstInt.signed && dstInt.size < srcInt.size && srcInt.size-dstInt.size > 8 {
		return true
	}

	return false
}
