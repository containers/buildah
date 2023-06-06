// (c) Copyright 2016 Hewlett Packard Enterprise Development LP
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

package rules

import (
	"go/ast"
	"regexp"

	"github.com/securego/gosec/v2"
	"github.com/securego/gosec/v2/issue"
)

type badTempFile struct {
	issue.MetaData
	calls       gosec.CallList
	args        *regexp.Regexp
	argCalls    gosec.CallList
	nestedCalls gosec.CallList
}

func (t *badTempFile) ID() string {
	return t.MetaData.ID
}

func (t *badTempFile) findTempDirArgs(n ast.Node, c *gosec.Context, suspect ast.Node) *issue.Issue {
	if s, e := gosec.GetString(suspect); e == nil {
		if t.args.MatchString(s) {
			return c.NewIssue(n, t.ID(), t.What, t.Severity, t.Confidence)
		}
		return nil
	}
	if ce := t.argCalls.ContainsPkgCallExpr(suspect, c, false); ce != nil {
		return c.NewIssue(n, t.ID(), t.What, t.Severity, t.Confidence)
	}
	if be, ok := suspect.(*ast.BinaryExpr); ok {
		if ops := gosec.GetBinaryExprOperands(be); len(ops) != 0 {
			return t.findTempDirArgs(n, c, ops[0])
		}
		return nil
	}
	if ce := t.nestedCalls.ContainsPkgCallExpr(suspect, c, false); ce != nil {
		return t.findTempDirArgs(n, c, ce.Args[0])
	}
	return nil
}

func (t *badTempFile) Match(n ast.Node, c *gosec.Context) (gi *issue.Issue, err error) {
	if node := t.calls.ContainsPkgCallExpr(n, c, false); node != nil {
		return t.findTempDirArgs(n, c, node.Args[0]), nil
	}
	return nil, nil
}

// NewBadTempFile detects direct writes to predictable path in temporary directory
func NewBadTempFile(id string, _ gosec.Config) (gosec.Rule, []ast.Node) {
	calls := gosec.NewCallList()
	calls.Add("io/ioutil", "WriteFile")
	calls.AddAll("os", "Create", "WriteFile")
	argCalls := gosec.NewCallList()
	argCalls.Add("os", "TempDir")
	nestedCalls := gosec.NewCallList()
	nestedCalls.Add("path", "Join")
	nestedCalls.Add("path/filepath", "Join")
	return &badTempFile{
		calls:       calls,
		args:        regexp.MustCompile(`^(/(usr|var))?/tmp(/.*)?$`),
		argCalls:    argCalls,
		nestedCalls: nestedCalls,
		MetaData: issue.MetaData{
			ID:         id,
			Severity:   issue.Medium,
			Confidence: issue.High,
			What:       "File creation in shared tmp directory without using ioutil.Tempfile",
		},
	}, []ast.Node{(*ast.CallExpr)(nil)}
}
