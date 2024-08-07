package tagalign

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/structtag"

	"golang.org/x/tools/go/analysis"
)

type Mode int

const (
	StandaloneMode Mode = iota
	GolangciLintMode
)

type Style int

const (
	DefaultStyle Style = iota
	StrictStyle
)

const (
	errTagValueSyntax = "bad syntax for struct tag value"
)

func NewAnalyzer(options ...Option) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "tagalign",
		Doc:  "check that struct tags are well aligned",
		Run: func(p *analysis.Pass) (any, error) {
			Run(p, options...)
			return nil, nil
		},
	}
}

func Run(pass *analysis.Pass, options ...Option) []Issue {
	var issues []Issue
	for _, f := range pass.Files {
		h := &Helper{
			mode:  StandaloneMode,
			style: DefaultStyle,
			align: true,
		}
		for _, opt := range options {
			opt(h)
		}

		//  StrictStyle must be used with WithAlign(true) and WithSort(...) together, or it will be ignored.
		if h.style == StrictStyle && (!h.align || !h.sort) {
			h.style = DefaultStyle
		}

		if !h.align && !h.sort {
			// do nothing
			return nil
		}

		ast.Inspect(f, func(n ast.Node) bool {
			h.find(pass, n)
			return true
		})
		h.Process(pass)
		issues = append(issues, h.issues...)
	}
	return issues
}

type Helper struct {
	mode Mode

	style Style

	align         bool     // whether enable tags align.
	sort          bool     // whether enable tags sort.
	fixedTagOrder []string // the order of tags, the other tags will be sorted by name.

	singleFields            []*ast.Field
	consecutiveFieldsGroups [][]*ast.Field // fields in this group, must be consecutive in struct.
	issues                  []Issue
}

// Issue is used to integrate with golangci-lint's inline auto fix.
type Issue struct {
	Pos       token.Position
	Message   string
	InlineFix InlineFix
}
type InlineFix struct {
	StartCol  int // zero-based
	Length    int
	NewString string
}

func (w *Helper) find(pass *analysis.Pass, n ast.Node) {
	v, ok := n.(*ast.StructType)
	if !ok {
		return
	}

	fields := v.Fields.List
	if len(fields) == 0 {
		return
	}

	fs := make([]*ast.Field, 0)
	split := func() {
		n := len(fs)
		if n > 1 {
			w.consecutiveFieldsGroups = append(w.consecutiveFieldsGroups, fs)
		} else if n == 1 {
			w.singleFields = append(w.singleFields, fs[0])
		}

		fs = nil
	}

	for i, field := range fields {
		if field.Tag == nil {
			// field without tags
			split()
			continue
		}

		if i > 0 {
			if fields[i-1].Tag == nil {
				// if previous filed do not have a tag
				fs = append(fs, field)
				continue
			}
			preLineNum := pass.Fset.Position(fields[i-1].Tag.Pos()).Line
			lineNum := pass.Fset.Position(field.Tag.Pos()).Line
			if lineNum-preLineNum > 1 {
				// fields with tags are not consecutive, including two case:
				// 1. splited by lines
				// 2. splited by a struct
				split()

				// check if the field is a struct
				if _, ok := field.Type.(*ast.StructType); ok {
					continue
				}
			}
		}

		fs = append(fs, field)
	}

	split()
}

func (w *Helper) report(pass *analysis.Pass, field *ast.Field, startCol int, msg, replaceStr string) {
	if w.mode == GolangciLintMode {
		iss := Issue{
			Pos:     pass.Fset.Position(field.Tag.Pos()),
			Message: msg,
			InlineFix: InlineFix{
				StartCol:  startCol,
				Length:    len(field.Tag.Value),
				NewString: replaceStr,
			},
		}
		w.issues = append(w.issues, iss)
	}

	if w.mode == StandaloneMode {
		pass.Report(analysis.Diagnostic{
			Pos:     field.Tag.Pos(),
			End:     field.Tag.End(),
			Message: msg,
			SuggestedFixes: []analysis.SuggestedFix{
				{
					Message: msg,
					TextEdits: []analysis.TextEdit{
						{
							Pos:     field.Tag.Pos(),
							End:     field.Tag.End(),
							NewText: []byte(replaceStr),
						},
					},
				},
			},
		})
	}
}

func (w *Helper) Process(pass *analysis.Pass) { //nolint:gocognit
	// process grouped fields
	for _, fields := range w.consecutiveFieldsGroups {
		offsets := make([]int, len(fields))

		var maxTagNum int
		var tagsGroup, notSortedTagsGroup [][]*structtag.Tag

		var uniqueKeys []string
		addKey := func(k string) {
			for _, key := range uniqueKeys {
				if key == k {
					return
				}
			}
			uniqueKeys = append(uniqueKeys, k)
		}

		for i := 0; i < len(fields); {
			field := fields[i]
			column := pass.Fset.Position(field.Tag.Pos()).Column - 1
			offsets[i] = column

			tag, err := strconv.Unquote(field.Tag.Value)
			if err != nil {
				// if tag value is not a valid string, report it directly
				w.report(pass, field, column, errTagValueSyntax, field.Tag.Value)
				fields = removeField(fields, i)
				continue
			}

			tags, err := structtag.Parse(tag)
			if err != nil {
				// if tag value is not a valid struct tag, report it directly
				w.report(pass, field, column, err.Error(), field.Tag.Value)
				fields = removeField(fields, i)
				continue
			}

			maxTagNum = max(maxTagNum, tags.Len())

			if w.sort {
				cp := make([]*structtag.Tag, tags.Len())
				for i, tag := range tags.Tags() {
					cp[i] = tag
				}
				notSortedTagsGroup = append(notSortedTagsGroup, cp)
				sortBy(w.fixedTagOrder, tags)
			}
			for _, t := range tags.Tags() {
				addKey(t.Key)
			}
			tagsGroup = append(tagsGroup, tags.Tags())

			i++
		}

		if w.sort && StrictStyle == w.style {
			sortAllKeys(w.fixedTagOrder, uniqueKeys)
			maxTagNum = len(uniqueKeys)
		}

		// record the max length of each column tag
		type tagLen struct {
			Key string // present only when sort enabled
			Len int
		}
		tagMaxLens := make([]tagLen, maxTagNum)
		for j := 0; j < maxTagNum; j++ {
			var maxLength int
			var key string
			for i := 0; i < len(tagsGroup); i++ {
				if w.style == StrictStyle {
					key = uniqueKeys[j]
					// search by key
					for _, tag := range tagsGroup[i] {
						if tag.Key == key {
							maxLength = max(maxLength, len(tag.String()))
							break
						}
					}
				} else {
					if len(tagsGroup[i]) <= j {
						// in case of index out of range
						continue
					}
					maxLength = max(maxLength, len(tagsGroup[i][j].String()))
				}
			}
			tagMaxLens[j] = tagLen{key, maxLength}
		}

		for i, field := range fields {
			tags := tagsGroup[i]

			var newTagStr string
			if w.align {
				// if align enabled, align tags.
				newTagBuilder := strings.Builder{}
				for i, n := 0, 0; i < len(tags) && n < len(tagMaxLens); {
					tag := tags[i]
					var format string
					if w.style == StrictStyle {
						if tagMaxLens[n].Key == tag.Key {
							// match
							format = alignFormat(tagMaxLens[n].Len + 1) // with an extra space
							newTagBuilder.WriteString(fmt.Sprintf(format, tag.String()))
							i++
							n++
						} else {
							// tag missing
							format = alignFormat(tagMaxLens[n].Len + 1)
							newTagBuilder.WriteString(fmt.Sprintf(format, ""))
							n++
						}
					} else {
						format = alignFormat(tagMaxLens[n].Len + 1) // with an extra space
						newTagBuilder.WriteString(fmt.Sprintf(format, tag.String()))
						i++
						n++
					}
				}
				newTagStr = newTagBuilder.String()
			} else {
				// otherwise check if tags order changed
				if w.sort && reflect.DeepEqual(notSortedTagsGroup[i], tags) {
					// if tags order not changed, do nothing
					continue
				}
				tagsStr := make([]string, len(tags))
				for i, tag := range tags {
					tagsStr[i] = tag.String()
				}
				newTagStr = strings.Join(tagsStr, " ")
			}

			unquoteTag := strings.TrimRight(newTagStr, " ")
			// unquoteTag := newTagStr
			newTagValue := fmt.Sprintf("`%s`", unquoteTag)
			if field.Tag.Value == newTagValue {
				// nothing changed
				continue
			}

			msg := "tag is not aligned, should be: " + unquoteTag

			w.report(pass, field, offsets[i], msg, newTagValue)
		}
	}

	// process single fields
	for _, field := range w.singleFields {
		column := pass.Fset.Position(field.Tag.Pos()).Column - 1
		tag, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			w.report(pass, field, column, errTagValueSyntax, field.Tag.Value)
			continue
		}

		tags, err := structtag.Parse(tag)
		if err != nil {
			w.report(pass, field, column, err.Error(), field.Tag.Value)
			continue
		}
		originalTags := append([]*structtag.Tag(nil), tags.Tags()...)
		if w.sort {
			sortBy(w.fixedTagOrder, tags)
		}

		newTagValue := fmt.Sprintf("`%s`", tags.String())
		if reflect.DeepEqual(originalTags, tags.Tags()) && field.Tag.Value == newTagValue {
			// if tags order not changed, do nothing
			continue
		}

		msg := "tag is not aligned , should be: " + tags.String()

		w.report(pass, field, column, msg, newTagValue)
	}
}

// Issues returns all issues found by the analyzer.
// It is used to integrate with golangci-lint.
func (w *Helper) Issues() []Issue {
	log.Println("tagalign 's Issues() should only be called in golangci-lint mode")
	return w.issues
}

// sortBy sorts tags by fixed order.
// If a tag is not in the fixed order, it will be sorted by name.
func sortBy(fixedOrder []string, tags *structtag.Tags) {
	// sort by fixed order
	sort.Slice(tags.Tags(), func(i, j int) bool {
		ti := tags.Tags()[i]
		tj := tags.Tags()[j]

		oi := findIndex(fixedOrder, ti.Key)
		oj := findIndex(fixedOrder, tj.Key)

		if oi == -1 && oj == -1 {
			return ti.Key < tj.Key
		}

		if oi == -1 {
			return false
		}

		if oj == -1 {
			return true
		}

		return oi < oj
	})
}

func sortAllKeys(fixedOrder []string, keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		oi := findIndex(fixedOrder, keys[i])
		oj := findIndex(fixedOrder, keys[j])

		if oi == -1 && oj == -1 {
			return keys[i] < keys[j]
		}

		if oi == -1 {
			return false
		}

		if oj == -1 {
			return true
		}

		return oi < oj
	})
}

func findIndex(s []string, e string) int {
	for i, a := range s {
		if a == e {
			return i
		}
	}
	return -1
}

func alignFormat(length int) string {
	return "%" + fmt.Sprintf("-%ds", length)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func removeField(fields []*ast.Field, index int) []*ast.Field {
	if index < 0 || index >= len(fields) {
		return fields
	}

	return append(fields[:index], fields[index+1:]...)
}
