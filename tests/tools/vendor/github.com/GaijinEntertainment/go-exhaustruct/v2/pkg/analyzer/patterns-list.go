package analyzer

import (
	"fmt"
	"regexp"
)

type PatternsList []*regexp.Regexp

// MatchesAny matches provided string against all regexps in a slice.
func (l PatternsList) MatchesAny(str string) bool {
	for _, r := range l {
		if r.MatchString(str) {
			return true
		}
	}

	return false
}

// newPatternsList parses slice of strings to a slice of compiled regular
// expressions.
func newPatternsList(in []string) (PatternsList, error) {
	list := PatternsList{}

	for _, str := range in {
		re, err := strToRegexp(str)
		if err != nil {
			return nil, err
		}

		list = append(list, re)
	}

	return list, nil
}

type reListVar struct {
	values *PatternsList
}

func (v *reListVar) Set(value string) error {
	re, err := strToRegexp(value)
	if err != nil {
		return err
	}

	*v.values = append(*v.values, re)

	return nil
}

func (v *reListVar) String() string {
	return ""
}

func strToRegexp(str string) (*regexp.Regexp, error) {
	if str == "" {
		return nil, ErrEmptyPattern
	}

	re, err := regexp.Compile(str)
	if err != nil {
		return nil, fmt.Errorf("unable to compile %s as regular expression: %w", str, err)
	}

	return re, nil
}
