package rule

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/mgechev/revive/lint"
)

// ImportsBlocklistRule lints given else constructs.
type ImportsBlocklistRule struct {
	blocklist []*regexp.Regexp
	sync.Mutex
}

var replaceImportRegexp = regexp.MustCompile(`/?\*\*/?`)

func (r *ImportsBlocklistRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()

	if r.blocklist == nil {
		r.blocklist = make([]*regexp.Regexp, 0)

		for _, arg := range arguments {
			argStr, ok := arg.(string)
			if !ok {
				panic(fmt.Sprintf("Invalid argument to the imports-blocklist rule. Expecting a string, got %T", arg))
			}
			regStr, err := regexp.Compile(fmt.Sprintf(`(?m)"%s"$`, replaceImportRegexp.ReplaceAllString(argStr, `(\W|\w)*`)))
			if err != nil {
				panic(fmt.Sprintf("Invalid argument to the imports-blocklist rule. Expecting %q to be a valid regular expression, got: %v", argStr, err))
			}
			r.blocklist = append(r.blocklist, regStr)
		}
	}
}

func (r *ImportsBlocklistRule) isBlocklisted(path string) bool {
	for _, regex := range r.blocklist {
		if regex.MatchString(path) {
			return true
		}
	}
	return false
}

// Apply applies the rule to given file.
func (r *ImportsBlocklistRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure

	for _, is := range file.AST.Imports {
		path := is.Path
		if path != nil && r.isBlocklisted(path.Value) {
			failures = append(failures, lint.Failure{
				Confidence: 1,
				Failure:    "should not use the following blocklisted import: " + path.Value,
				Node:       is,
				Category:   "imports",
			})
		}
	}

	return failures
}

// Name returns the rule name.
func (*ImportsBlocklistRule) Name() string {
	return "imports-blocklist"
}
