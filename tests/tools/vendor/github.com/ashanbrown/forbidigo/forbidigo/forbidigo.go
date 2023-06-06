// forbidigo provides a linter for forbidding the use of specific identifiers
package forbidigo

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"regexp"
	"strings"
)

type Issue interface {
	Details() string
	Pos() token.Pos
	Position() token.Position
	String() string
}

type UsedIssue struct {
	identifier string
	pattern    string
	pos        token.Pos
	position   token.Position
	customMsg  string
}

func (a UsedIssue) Details() string {
	explanation := fmt.Sprintf(` because %q`, a.customMsg)
	if a.customMsg == "" {
		explanation = fmt.Sprintf(" by pattern `%s`", a.pattern)
	}
	return fmt.Sprintf("use of `%s` forbidden", a.identifier) + explanation
}

func (a UsedIssue) Position() token.Position {
	return a.position
}

func (a UsedIssue) Pos() token.Pos {
	return a.pos
}

func (a UsedIssue) String() string { return toString(a) }

func toString(i UsedIssue) string {
	return fmt.Sprintf("%s at %s", i.Details(), i.Position())
}

type Linter struct {
	cfg      config
	patterns []*pattern
}

func DefaultPatterns() []string {
	return []string{`^(fmt\.Print(|f|ln)|print|println)$`}
}

//go:generate go-options config
type config struct {
	// don't check inside Godoc examples (see https://blog.golang.org/examples)
	ExcludeGodocExamples   bool `options:",true"`
	IgnorePermitDirectives bool // don't check for `permit` directives(for example, in favor of `nolint`)
	AnalyzeTypes bool // enable to match canonical names for types and interfaces using type info
}

func NewLinter(patterns []string, options ...Option) (*Linter, error) {
	cfg, err := newConfig(options...)
	if err != nil {
		return nil, fmt.Errorf("failed to process options: %w", err)
	}

	if len(patterns) == 0 {
		patterns = DefaultPatterns()
	}
	compiledPatterns := make([]*pattern, 0, len(patterns))
	for _, ptrn := range patterns {
		p, err := parse(ptrn)
		if err != nil {
			return nil, err
		}
		compiledPatterns = append(compiledPatterns, p)
	}
	return &Linter{
		cfg:      cfg,
		patterns: compiledPatterns,
	}, nil
}

type visitor struct {
	cfg        config
	isTestFile bool // godoc only runs on test files

	linter   *Linter
	comments []*ast.CommentGroup

	runConfig RunConfig
	issues    []Issue
}

// Deprecated: Run was the original entrypoint before RunWithConfig was introduced to support
// additional match patterns that need additional information.
func (l *Linter) Run(fset *token.FileSet, nodes ...ast.Node) ([]Issue, error) {
	return l.RunWithConfig(RunConfig{Fset: fset}, nodes...)
}

// RunConfig provides information that the linter needs for different kinds
// of match patterns. Ideally, all fields should get set. More fields may get
// added in the future as needed.
type RunConfig struct {
	// FSet is required.
	Fset *token.FileSet

	// TypesInfo is needed for expanding source code expressions.
	// Nil disables that step, i.e. patterns match the literal source code.
	TypesInfo *types.Info

	// DebugLog is used to print debug messages. May be nil.
	DebugLog func(format string, args ...interface{})
}

func (l *Linter) RunWithConfig(config RunConfig, nodes ...ast.Node) ([]Issue, error) {
	if config.DebugLog == nil {
		config.DebugLog = func(format string, args ...interface{}) {}
	}
	var issues []Issue
	for _, node := range nodes {
		var comments []*ast.CommentGroup
		isTestFile := false
		isWholeFileExample := false
		if file, ok := node.(*ast.File); ok {
			comments = file.Comments
			fileName := config.Fset.Position(file.Pos()).Filename
			isTestFile = strings.HasSuffix(fileName, "_test.go")

			// From https://blog.golang.org/examples, a "whole file example" is:
			// a file that ends in _test.go and contains exactly one example function,
			// no test or benchmark functions, and at least one other package-level declaration.
			if l.cfg.ExcludeGodocExamples && isTestFile && len(file.Decls) > 1 {
				numExamples := 0
				numTestsAndBenchmarks := 0
				for _, decl := range file.Decls {
					funcDecl, isFuncDecl := decl.(*ast.FuncDecl)
					// consider only functions, not methods
					if !isFuncDecl || funcDecl.Recv != nil || funcDecl.Name == nil {
						continue
					}
					funcName := funcDecl.Name.Name
					if strings.HasPrefix(funcName, "Test") || strings.HasPrefix(funcName, "Benchmark") {
						numTestsAndBenchmarks++
						break // not a whole file example
					}
					if strings.HasPrefix(funcName, "Example") {
						numExamples++
					}
				}

				// if this is a whole file example, skip this node
				isWholeFileExample = numExamples == 1 && numTestsAndBenchmarks == 0
			}
		}
		if isWholeFileExample {
			continue
		}
		visitor := visitor{
			cfg:        l.cfg,
			isTestFile: isTestFile,
			linter:     l,
			runConfig:  config,
			comments:   comments,
		}
		ast.Walk(&visitor, node)
		issues = append(issues, visitor.issues...)
	}
	return issues, nil
}

func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch node := node.(type) {
	case *ast.FuncDecl:
		// don't descend into godoc examples if we are ignoring them
		isGodocExample := v.isTestFile && node.Recv == nil && node.Name != nil && strings.HasPrefix(node.Name.Name, "Example")
		if isGodocExample && v.cfg.ExcludeGodocExamples {
			return nil
		}
		return v
	// The following two are handled below.
	case *ast.SelectorExpr:
	case *ast.Ident:
	// Everything else isn't.
	default:
		return v
	}

	// The text as it appears in the source is always used because issues
	// use that. It's used for matching unless usage of type information
	// is enabled.
	srcText := v.textFor(node)
	matchTexts, pkgText := v.expandMatchText(node, srcText)
	v.runConfig.DebugLog("%s: match %v, package %q", v.runConfig.Fset.Position(node.Pos()), matchTexts, pkgText)
	for _, p := range v.linter.patterns {
		if p.matches(matchTexts) &&
			(p.Package == "" || p.pkgRe.MatchString(pkgText)) &&
			!v.permit(node) {
			v.issues = append(v.issues, UsedIssue{
				identifier: srcText, // Always report the expression as it appears in the source code.
				pattern:    p.re.String(),
				pos:        node.Pos(),
				position:   v.runConfig.Fset.Position(node.Pos()),
				customMsg:  p.Msg,
			})
		}
	}
	return nil
}

// textFor returns the expression as it appears in the source code (for
// example, <importname>.<function name>).
func (v *visitor) textFor(node ast.Node) string {
	buf := new(bytes.Buffer)
	if err := printer.Fprint(buf, v.runConfig.Fset, node); err != nil {
		log.Fatalf("ERROR: unable to print node at %s: %s", v.runConfig.Fset.Position(node.Pos()), err)
	}
	return buf.String()
}

// expandMatchText expands the selector in a selector expression to the full package
// name and (for variables) the type:
//
// - example.com/some/pkg.Function
// - example.com/some/pkg.CustomType.Method
//
// It updates the text to match against and fills the package string if possible,
// otherwise it just returns.
func (v *visitor) expandMatchText(node ast.Node, srcText string) (matchTexts []string, pkgText string) {
	// The text to match against is the literal source code if we cannot
	// come up with something different.
	matchText := srcText

	if !v.cfg.AnalyzeTypes || v.runConfig.TypesInfo == nil {
		return []string{matchText}, pkgText
	}

	location := v.runConfig.Fset.Position(node.Pos())

	switch node := node.(type) {
	case *ast.Ident:
		object, ok := v.runConfig.TypesInfo.Uses[node]
		if !ok {
			// No information about the identifier. Should
			// not happen, but perhaps there were compile
			// errors?
			v.runConfig.DebugLog("%s: unknown identifier %q", location, srcText)
			return []string{matchText}, pkgText
		}
		if pkg := object.Pkg(); pkg != nil {
			pkgText = pkg.Path()
			v.runConfig.DebugLog("%s: identifier: %q -> %q in package %q", location, srcText, matchText, pkgText)
			// match either with or without package name
			return []string{pkg.Name() + "." + srcText, srcText}, pkgText
		} else {
			v.runConfig.DebugLog("%s: identifier: %q -> %q without package", location, srcText, matchText)
		}
		return []string{matchText}, pkgText
	case *ast.SelectorExpr:
		selector := node.X
		field := node.Sel.Name

		// If we are lucky, the entire selector expression has a known
		// type. We don't care about the value.
		selectorText := v.textFor(node)
		if typeAndValue, ok := v.runConfig.TypesInfo.Types[selector]; ok {
			m, p, ok := pkgFromType(typeAndValue.Type)
			if !ok {
				v.runConfig.DebugLog("%s: selector %q with supported type %T", location, selectorText, typeAndValue.Type)
			}
			matchText = m + "." + field
			pkgText = p
			v.runConfig.DebugLog("%s: selector %q with supported type %q: %q -> %q, package %q", location, selectorText, typeAndValue.Type.String(), srcText, matchText, pkgText)
			return []string{matchText}, pkgText
		}
		// Some expressions need special treatment.
		switch selector := selector.(type) {
		case *ast.Ident:
			object, ok := v.runConfig.TypesInfo.Uses[selector]
			if !ok {
				// No information about the identifier. Should
				// not happen, but perhaps there were compile
				// errors?
				v.runConfig.DebugLog("%s: unknown selector identifier %q", location, selectorText)
				return []string{matchText}, pkgText
			}
			switch object := object.(type) {
			case *types.PkgName:
				pkgText = object.Imported().Path()
				matchText = object.Imported().Name() + "." + field
				v.runConfig.DebugLog("%s: selector %q is package: %q -> %q, package %q", location, selectorText, srcText, matchText, pkgText)
				return []string{matchText}, pkgText
			case *types.Var:
				m, p, ok := pkgFromType(object.Type())
				if !ok {
					v.runConfig.DebugLog("%s: selector %q is variable with unsupported type %T", location, selectorText, object.Type())
				}
				matchText = m + "." + field
				pkgText = p
				v.runConfig.DebugLog("%s: selector %q is variable of type %q: %q -> %q, package %q", location, selectorText, object.Type().String(), srcText, matchText, pkgText)
			default:
				// Something else?
				v.runConfig.DebugLog("%s: selector %q is identifier with unsupported type %T", location, selectorText, object)
			}
		default:
			v.runConfig.DebugLog("%s: selector %q of unsupported type %T", location, selectorText, selector)
		}
		return []string{matchText}, pkgText
	default:
		v.runConfig.DebugLog("%s: unsupported type %T", location, node)
		return []string{matchText}, pkgText
	}
}

// pkgFromType tries to determine `<package name>.<type name>` and the full
// package path. This only needs to work for types of a selector in a selector
// expression.
func pkgFromType(t types.Type) (typeStr, pkgStr string, ok bool) {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	switch t := t.(type) {
	case *types.Named:
		obj := t.Obj()
		pkg := obj.Pkg()
		if pkg == nil {
			return "", "", false
		}
		return pkg.Name() + "." + obj.Name(), pkg.Path(), true
	default:
		return "", "", false
	}
}

func (v *visitor) permit(node ast.Node) bool {
	if v.cfg.IgnorePermitDirectives {
		return false
	}
	nodePos := v.runConfig.Fset.Position(node.Pos())
	nolint := regexp.MustCompile(fmt.Sprintf(`^//\s?permit:%s\b`, regexp.QuoteMeta(v.textFor(node))))
	for _, c := range v.comments {
		commentPos := v.runConfig.Fset.Position(c.Pos())
		if commentPos.Line == nodePos.Line && len(c.List) > 0 && nolint.MatchString(c.List[0].Text) {
			return true
		}
	}
	return false
}
