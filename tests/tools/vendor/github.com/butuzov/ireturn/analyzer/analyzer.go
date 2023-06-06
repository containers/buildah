package analyzer

import (
	"flag"
	"go/ast"
	gotypes "go/types"
	"strings"
	"sync"

	"github.com/butuzov/ireturn/analyzer/internal/config"
	"github.com/butuzov/ireturn/analyzer/internal/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const name string = "ireturn" // linter name

type validator interface {
	IsValid(types.IFace) bool
}

type analyzer struct {
	once    sync.Once
	handler validator
	err     error

	found []analysis.Diagnostic
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	// 00. Part 1. Handling Configuration Only Once.
	a.once.Do(func() { a.readConfiguration(&pass.Analyzer.Flags) })

	// 00. Part 2. Handling Errors
	if a.err != nil {
		return nil, a.err
	}

	ins, _ := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// 00. does file have dot-imported standard packages?
	dotImportedStd := make(map[string]struct{})
	ins.Preorder([]ast.Node{(*ast.ImportSpec)(nil)}, func(node ast.Node) {
		i, _ := node.(*ast.ImportSpec)
		if i.Name != nil && i.Name.Name == "." {
			dotImportedStd[strings.Trim(i.Path.Value, `"`)] = struct{}{}
		}
	})

	// 01. Running Inspection.
	ins.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(node ast.Node) {
		// 001. Casting to funcdecl
		f, _ := node.(*ast.FuncDecl)

		// 002. Does it return any results ?
		if f.Type == nil || f.Type.Results == nil {
			return
		}

		// 003. Is it allowed to be checked?
		// TODO(butuzov): add inline comment
		if hasDisallowDirective(f.Doc) {
			return
		}

		seen := make(map[string]bool, 4)

		// 004. Filtering Results.
		for _, issue := range filterInterfaces(pass, f.Type, dotImportedStd) {

			if a.handler.IsValid(issue) {
				continue
			}

			issue.Enrich(f)

			key := issue.HashString()

			if ok := seen[key]; ok {
				continue
			}
			seen[key] = true

			a.found = append(a.found, issue.ExportDiagnostic())
		}
	})

	// 02. Printing reports.
	for i := range a.found {
		pass.Report(a.found[i])
	}

	return nil, nil
}

func (a *analyzer) readConfiguration(fs *flag.FlagSet) {
	cnf, err := config.New(fs)
	if err != nil {
		a.err = err
		return
	}

	if validatorImpl, ok := cnf.(validator); ok {
		a.handler = validatorImpl
		return
	}

	a.handler = config.DefaultValidatorConfig()
}

func NewAnalyzer() *analysis.Analyzer {
	a := analyzer{} //nolint: exhaustivestruct

	return &analysis.Analyzer{
		Name:     name,
		Doc:      "Accept Interfaces, Return Concrete Types",
		Run:      a.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Flags:    flags(),
	}
}

func flags() flag.FlagSet {
	set := flag.NewFlagSet("", flag.PanicOnError)
	set.String("allow", "", "accept-list of the comma-separated interfaces")
	set.String("reject", "", "reject-list of the comma-separated interfaces")
	return *set
}

func filterInterfaces(p *analysis.Pass, ft *ast.FuncType, di map[string]struct{}) []types.IFace {
	var results []types.IFace

	if ft.Results == nil { // this can't happen, but double checking.
		return results
	}

	tp := newTypeParams(ft.TypeParams)

	for _, el := range ft.Results.List {
		switch v := el.Type.(type) {
		// ----- empty or anonymous interfaces
		case *ast.InterfaceType:

			if len(v.Methods.List) == 0 {
				results = append(results, types.NewIssue("interface{}", types.EmptyInterface))
				continue
			}

			results = append(results, types.NewIssue("anonymous interface", types.AnonInterface))

		// ------ Errors and interfaces from same package
		case *ast.Ident:

			t1 := p.TypesInfo.TypeOf(el.Type)
			if !gotypes.IsInterface(t1.Underlying()) {
				continue
			}

			word := t1.String()
			// only build in interface is error
			if obj := gotypes.Universe.Lookup(word); obj != nil {
				results = append(results, types.NewIssue(obj.Name(), types.ErrorInterface))
				continue
			}

			// found in type params
			if tp.In(word) {
				results = append(results, types.NewIssue(word, types.Generic))
				continue
			}

			// is it dot-imported package?
			// handling cases when stdlib package imported via "." dot-import
			if len(di) > 0 {
				name := stdPkgInterface(word)
				if _, ok := di[name]; ok {
					results = append(results, types.NewIssue(word, types.NamedStdInterface))

					continue
				}
			}

			results = append(results, types.NewIssue(word, types.NamedInterface))

		// ------- standard library and 3rd party interfaces
		case *ast.SelectorExpr:

			t1 := p.TypesInfo.TypeOf(el.Type)
			if !gotypes.IsInterface(t1.Underlying()) {
				continue
			}

			word := t1.String()
			if isStdPkgInterface(word) {
				results = append(results, types.NewIssue(word, types.NamedStdInterface))
				continue
			}

			results = append(results, types.NewIssue(word, types.NamedInterface))
		}
	}

	return results
}

// stdPkgInterface will return package name if tis std lib package
// or empty string on fail.
func stdPkgInterface(named string) string {
	// find last "." index.
	idx := strings.LastIndex(named, ".")
	if idx == -1 {
		return ""
	}

	return stdPkg(named[0:idx])
}

// isStdPkgInterface will run small checks against pkg to find out if named
// interface we looking on - comes from a standard library or not.
func isStdPkgInterface(namedInterface string) bool {
	return stdPkgInterface(namedInterface) != ""
}

func stdPkg(pkg string) string {
	if _, ok := std[pkg]; ok {
		return pkg
	}

	return ""
}
