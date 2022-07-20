package section

import (
	"fmt"
	"strings"

	"github.com/daixiang0/gci/pkg/parse"
	"github.com/daixiang0/gci/pkg/specificity"
)

type Custom struct {
	Prefix string
}

func (c Custom) MatchSpecificity(spec *parse.GciImports) specificity.MatchSpecificity {
	if strings.HasPrefix(spec.Path, c.Prefix) {
		return specificity.Match{Length: len(c.Prefix)}
	}
	return specificity.MisMatch{}
}

func (c Custom) String() string {
	return fmt.Sprintf("prefix(%s)", c.Prefix)
}
