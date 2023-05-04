package rule

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/mgechev/revive/lint"
)

// ImportsBlacklistRule lints given else constructs.
type ImportsBlacklistRule struct {
	blacklist []*regexp.Regexp
	sync.Mutex
}

var replaceRegexp = regexp.MustCompile(`/?\*\*/?`)

func (r *ImportsBlacklistRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()

	if r.blacklist == nil {
		r.blacklist = make([]*regexp.Regexp, 0)

		for _, arg := range arguments {
			argStr, ok := arg.(string)
			if !ok {
				panic(fmt.Sprintf("Invalid argument to the imports-blacklist rule. Expecting a string, got %T", arg))
			}
			regStr, err := regexp.Compile(fmt.Sprintf(`(?m)"%s"$`, replaceRegexp.ReplaceAllString(argStr, `(\W|\w)*`)))
			if err != nil {
				panic(fmt.Sprintf("Invalid argument to the imports-blacklist rule. Expecting %q to be a valid regular expression, got: %v", argStr, err))
			}
			r.blacklist = append(r.blacklist, regStr)
		}
	}
}

func (r *ImportsBlacklistRule) isBlacklisted(path string) bool {
	for _, regex := range r.blacklist {
		if regex.MatchString(path) {
			return true
		}
	}
	return false
}

// Apply applies the rule to given file.
func (r *ImportsBlacklistRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure

	if file.IsTest() {
		return failures // skip, test file
	}

	for _, is := range file.AST.Imports {
		path := is.Path
		if path != nil && r.isBlacklisted(path.Value) {
			failures = append(failures, lint.Failure{
				Confidence: 1,
				Failure:    "should not use the following blacklisted import: " + path.Value,
				Node:       is,
				Category:   "imports",
			})
		}
	}

	return failures
}

// Name returns the rule name.
func (*ImportsBlacklistRule) Name() string {
	return "imports-blacklist"
}
