package checkers

import (
	"go/ast"
	"regexp"
	"strings"

	"github.com/go-critic/go-critic/checkers/internal/astwalk"
	"github.com/go-critic/go-critic/linter"
)

func init() {
	var info linter.CheckerInfo
	info.Name = "whyNoLint"
	info.Tags = []string{linter.StyleTag, linter.ExperimentalTag}
	info.Summary = "Ensures that `//nolint` comments include an explanation"
	info.Before = `//nolint`
	info.After = `//nolint // reason`

	collection.AddChecker(&info, func(ctx *linter.CheckerContext) (linter.FileWalker, error) {
		return astwalk.WalkerForComment(&whyNoLintChecker{
			ctx: ctx,
			re:  regexp.MustCompile(`^// *nolint(?::[^ ]+)? *(.*)$`),
		}), nil
	})
}

type whyNoLintChecker struct {
	astwalk.WalkHandler

	ctx *linter.CheckerContext
	re  *regexp.Regexp
}

func (c whyNoLintChecker) VisitComment(cg *ast.CommentGroup) {
	if strings.HasPrefix(cg.List[0].Text, "/*") {
		return
	}
	for _, comment := range cg.List {
		sl := c.re.FindStringSubmatch(comment.Text)
		if len(sl) < 2 {
			continue
		}

		if s := sl[1]; !strings.HasPrefix(s, "//") || strings.TrimPrefix(s, "//") == "" {
			c.ctx.Warn(cg, "include an explanation for nolint directive")
			return
		}
	}
}
