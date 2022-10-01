package rules

import (
	"go/ast"

	"github.com/securego/gosec/v2"
)

type httpServeWithoutTimeouts struct {
	gosec.MetaData
	pkg   string
	calls []string
}

func (r *httpServeWithoutTimeouts) ID() string {
	return r.MetaData.ID
}

func (r *httpServeWithoutTimeouts) Match(n ast.Node, c *gosec.Context) (gi *gosec.Issue, err error) {
	if _, matches := gosec.MatchCallByPackage(n, c, r.pkg, r.calls...); matches {
		return gosec.NewIssue(c, n, r.ID(), r.What, r.Severity, r.Confidence), nil
	}
	return nil, nil
}

// NewHTTPServeWithoutTimeouts detects use of net/http serve functions that have no support for setting timeouts.
func NewHTTPServeWithoutTimeouts(id string, conf gosec.Config) (gosec.Rule, []ast.Node) {
	return &httpServeWithoutTimeouts{
		pkg:   "net/http",
		calls: []string{"ListenAndServe", "ListenAndServeTLS", "Serve", "ServeTLS"},
		MetaData: gosec.MetaData{
			ID:         id,
			What:       "Use of net/http serve function that has no support for setting timeouts",
			Severity:   gosec.Medium,
			Confidence: gosec.High,
		},
	}, []ast.Node{(*ast.CallExpr)(nil)}
}
