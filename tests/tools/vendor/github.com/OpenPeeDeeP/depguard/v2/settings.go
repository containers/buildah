package depguard

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/OpenPeeDeeP/depguard/v2/internal/utils"
	"github.com/gobwas/glob"
)

type List struct {
	Files []string          `json:"files" yaml:"files" toml:"files" mapstructure:"files"`
	Allow []string          `json:"allow" yaml:"allow" toml:"allow" mapstructure:"allow"`
	Deny  map[string]string `json:"deny" yaml:"deny" toml:"deny" mapstructure:"deny"`
}

type list struct {
	name        string
	files       []glob.Glob
	negFiles    []glob.Glob
	allow       []string
	deny        []string
	suggestions []string
}

func (l *List) compile() (*list, error) {
	if l == nil {
		return nil, nil
	}
	li := &list{}
	var errs utils.MultiError
	var err error

	// Compile Files
	for _, f := range l.Files {
		var negate bool
		if len(f) > 0 && f[0] == '!' {
			negate = true
			f = f[1:]
		}
		// Expand File if needed
		fs, err := utils.ExpandSlice([]string{f}, utils.PathExpandable)
		if err != nil {
			errs = append(errs, err)
		}
		for _, exp := range fs {
			g, err := glob.Compile(exp, '/')
			if err != nil {
				errs = append(errs, fmt.Errorf("%s could not be compiled: %w", exp, err))
				continue
			}
			if negate {
				li.negFiles = append(li.negFiles, g)
				continue
			}
			li.files = append(li.files, g)
		}
	}

	if len(l.Allow) > 0 {
		// Expand Allow
		l.Allow, err = utils.ExpandSlice(l.Allow, utils.PackageExpandable)
		if err != nil {
			errs = append(errs, err)
		}

		// Sort Allow
		li.allow = make([]string, len(l.Allow))
		copy(li.allow, l.Allow)
		sort.Strings(li.allow)
	}

	if l.Deny != nil {
		// Expand Deny Map (to keep suggestions)
		err = utils.ExpandMap(l.Deny, utils.PackageExpandable)
		if err != nil {
			errs = append(errs, err)
		}

		// Split Deny Into Package Slice
		li.deny = make([]string, 0, len(l.Deny))
		for pkg := range l.Deny {
			li.deny = append(li.deny, pkg)
		}

		// Sort Deny
		sort.Strings(li.deny)

		// Populate Suggestions to match the Deny order
		li.suggestions = make([]string, 0, len(li.deny))
		for _, dp := range li.deny {
			li.suggestions = append(li.suggestions, strings.TrimSpace(l.Deny[dp]))
		}
	}

	// Populate the type of this list
	if len(li.allow) == 0 && len(li.deny) == 0 {
		errs = append(errs, errors.New("must have an Allow and/or Deny package list"))
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return li, nil
}

func (l *list) fileMatch(fileName string) bool {
	inAllowed := len(l.files) == 0 || strInGlobList(fileName, l.files)
	inDenied := strInGlobList(fileName, l.negFiles)
	return inAllowed && !inDenied
}

func (l *list) importAllowed(imp string) (bool, string) {
	inAllowed := len(l.allow) == 0
	if !inAllowed {
		inAllowed, _ = strInPrefixList(imp, l.allow)
	}
	inDenied, suggIdx := strInPrefixList(imp, l.deny)
	sugg := ""
	if inDenied && suggIdx != -1 {
		sugg = l.suggestions[suggIdx]
	}
	return inAllowed && !inDenied, sugg
}

type LinterSettings map[string]*List

type linterSettings []*list

func (l LinterSettings) compile() (linterSettings, error) {
	if len(l) == 0 {
		// Only allow $gostd in all files
		set := &List{
			Files: []string{"$all"},
			Allow: []string{"$gostd"},
		}
		li, err := set.compile()
		if err != nil {
			return nil, err
		}
		li.name = "Main"
		return linterSettings{li}, nil
	}
	names := make([]string, 0, len(l))
	for name := range l {
		names = append(names, name)
	}
	sort.Strings(names)
	li := make(linterSettings, 0, len(l))
	var errs utils.MultiError
	for _, name := range names {
		c, err := l[name].compile()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if c == nil {
			continue
		}
		c.name = name
		li = append(li, c)
	}
	if len(errs) > 0 {
		return nil, errs
	}

	return li, nil
}

func (ls linterSettings) whichLists(fileName string) []*list {
	var matches []*list
	for _, l := range ls {
		if l.fileMatch(fileName) {
			matches = append(matches, l)
		}
	}
	return matches
}

func strInGlobList(str string, globList []glob.Glob) bool {
	for _, g := range globList {
		if g.Match(str) {
			return true
		}
	}
	return false
}

func strInPrefixList(str string, prefixList []string) (bool, int) {
	// Idx represents where in the prefix slice the passed in string would go
	// when sorted. -1 Just means that it would be at the very front of the slice.
	idx := sort.Search(len(prefixList), func(i int) bool {
		return strings.TrimRight(prefixList[i], "$") > str
	}) - 1
	// This means that the string passed in has no way to be prefixed by anything
	// in the prefix list as it is already smaller then everything
	if idx == -1 {
		return false, idx
	}
	ioc := prefixList[idx]
	if ioc[len(ioc)-1] == '$' {
		return str == ioc[:len(ioc)-1], idx
	}
	return strings.HasPrefix(str, prefixList[idx]), idx
}
