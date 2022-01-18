package bidichk

import (
	"bytes"
	"go/token"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "bidichk",
	Doc:  "Checks for dangerous unicode character sequences",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	var err error

	pass.Fset.Iterate(func(f *token.File) bool {
		if strings.HasPrefix(f.Name(), "$GOROOT") {
			return true
		}

		return check(f.Name(), f.Pos(0), pass) == nil
	})

	return nil, err
}

var disallowedRunes = map[string]rune{
	"LEFT-TO-RIGHT-EMBEDDING":    '\u202A',
	"RIGHT-TO-LEFT-EMBEDDING":    '\u202B',
	"POP-DIRECTIONAL-FORMATTING": '\u202C',
	"LEFT-TO-RIGHT-OVERRIDE":     '\u202D',
	"RIGHT-TO-LEFT-OVERRIDE":     '\u202E',
	"LEFT-TO-RIGHT-ISOLATE":      '\u2066',
	"RIGHT-TO-LEFT-ISOLATE":      '\u2067',
	"FIRST-STRONG-ISOLATE":       '\u2068',
	"POP-DIRECTIONAL-ISOLATE":    '\u2069',
}

func check(filename string, pos token.Pos, pass *analysis.Pass) error {
	body, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	for name, r := range disallowedRunes {
		start := 0
		for {
			idx := bytes.IndexRune(body[start:], r)
			if idx == -1 {
				break
			}
			start += idx

			pass.Reportf(pos+token.Pos(start), "found dangerous unicode character sequence %s", name)

			start += utf8.RuneLen(r)
		}
	}

	return nil
}
