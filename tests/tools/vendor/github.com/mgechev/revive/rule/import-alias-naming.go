package rule

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/mgechev/revive/lint"
)

// ImportAliasNamingRule lints import alias naming.
type ImportAliasNamingRule struct {
	configured  bool
	allowRegexp *regexp.Regexp
	denyRegexp  *regexp.Regexp
	sync.Mutex
}

const defaultImportAliasNamingAllowRule = "^[a-z][a-z0-9]{0,}$"

var defaultImportAliasNamingAllowRegexp = regexp.MustCompile(defaultImportAliasNamingAllowRule)

func (r *ImportAliasNamingRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()
	if r.configured {
		return
	}

	if len(arguments) == 0 {
		r.allowRegexp = defaultImportAliasNamingAllowRegexp
		return
	}

	switch namingRule := arguments[0].(type) {
	case string:
		r.setAllowRule(namingRule)
	case map[string]any: // expecting map[string]string
		for k, v := range namingRule {
			switch k {
			case "allowRegex":
				r.setAllowRule(v)
			case "denyRegex":
				r.setDenyRule(v)
			default:
				panic(fmt.Sprintf("Invalid map key for 'import-alias-naming' rule. Expecting 'allowRegex' or 'denyRegex', got %v", k))
			}
		}
	default:
		panic(fmt.Sprintf("Invalid argument '%v' for 'import-alias-naming' rule. Expecting string or map[string]string, got %T", arguments[0], arguments[0]))
	}

	if r.allowRegexp == nil && r.denyRegexp == nil {
		r.allowRegexp = defaultImportAliasNamingAllowRegexp
	}
}

// Apply applies the rule to given file.
func (r *ImportAliasNamingRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure

	for _, is := range file.AST.Imports {
		path := is.Path
		if path == nil {
			continue
		}

		alias := is.Name
		if alias == nil || alias.Name == "_" || alias.Name == "." { // "_" and "." are special types of import aiases and should be processed by another linter rule
			continue
		}

		if r.allowRegexp != nil && !r.allowRegexp.MatchString(alias.Name) {
			failures = append(failures, lint.Failure{
				Confidence: 1,
				Failure:    fmt.Sprintf("import name (%s) must match the regular expression: %s", alias.Name, r.allowRegexp.String()),
				Node:       alias,
				Category:   "imports",
			})
		}

		if r.denyRegexp != nil && r.denyRegexp.MatchString(alias.Name) {
			failures = append(failures, lint.Failure{
				Confidence: 1,
				Failure:    fmt.Sprintf("import name (%s) must NOT match the regular expression: %s", alias.Name, r.denyRegexp.String()),
				Node:       alias,
				Category:   "imports",
			})
		}
	}

	return failures
}

// Name returns the rule name.
func (*ImportAliasNamingRule) Name() string {
	return "import-alias-naming"
}

func (r *ImportAliasNamingRule) setAllowRule(value any) {
	namingRule, ok := value.(string)
	if !ok {
		panic(fmt.Sprintf("Invalid argument '%v' for import-alias-naming allowRegexp rule. Expecting string, got %T", value, value))
	}

	namingRuleRegexp, err := regexp.Compile(namingRule)
	if err != nil {
		panic(fmt.Sprintf("Invalid argument to the import-alias-naming allowRegexp rule. Expecting %q to be a valid regular expression, got: %v", namingRule, err))
	}
	r.allowRegexp = namingRuleRegexp
}

func (r *ImportAliasNamingRule) setDenyRule(value any) {
	namingRule, ok := value.(string)
	if !ok {
		panic(fmt.Sprintf("Invalid argument '%v' for import-alias-naming denyRegexp rule. Expecting string, got %T", value, value))
	}

	namingRuleRegexp, err := regexp.Compile(namingRule)
	if err != nil {
		panic(fmt.Sprintf("Invalid argument to the import-alias-naming denyRegexp rule. Expecting %q to be a valid regular expression, got: %v", namingRule, err))
	}
	r.denyRegexp = namingRuleRegexp
}
