package lint

import (
	"go/ast"
	"go/importer"
	"go/token"
	"go/types"
	"sync"

	"github.com/mgechev/revive/internal/typeparams"
)

// Package represents a package in the project.
type Package struct {
	fset  *token.FileSet
	files map[string]*File

	typesPkg  *types.Package
	typesInfo *types.Info

	// sortable is the set of types in the package that implement sort.Interface.
	sortable map[string]bool
	// main is whether this is a "main" package.
	main int
	sync.RWMutex
}

var (
	trueValue  = 1
	falseValue = 2
	notSet     = 3
)

// Files return package's files.
func (p *Package) Files() map[string]*File {
	return p.files
}

// IsMain returns if that's the main package.
func (p *Package) IsMain() bool {
	p.Lock()
	defer p.Unlock()

	if p.main == trueValue {
		return true
	} else if p.main == falseValue {
		return false
	}
	for _, f := range p.files {
		if f.isMain() {
			p.main = trueValue
			return true
		}
	}
	p.main = falseValue
	return false
}

// TypesPkg yields information on this package
func (p *Package) TypesPkg() *types.Package {
	p.RLock()
	defer p.RUnlock()
	return p.typesPkg
}

// TypesInfo yields type information of this package identifiers
func (p *Package) TypesInfo() *types.Info {
	p.RLock()
	defer p.RUnlock()
	return p.typesInfo
}

// Sortable yields a map of sortable types in this package
func (p *Package) Sortable() map[string]bool {
	p.RLock()
	defer p.RUnlock()
	return p.sortable
}

// TypeCheck performs type checking for given package.
func (p *Package) TypeCheck() error {
	p.Lock()
	defer p.Unlock()

	// If type checking has already been performed
	// skip it.
	if p.typesInfo != nil || p.typesPkg != nil {
		return nil
	}
	config := &types.Config{
		// By setting a no-op error reporter, the type checker does as much work as possible.
		Error:    func(error) {},
		Importer: importer.Default(),
	}
	info := &types.Info{
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	var anyFile *File
	var astFiles []*ast.File
	for _, f := range p.files {
		anyFile = f
		astFiles = append(astFiles, f.AST)
	}

	typesPkg, err := check(config, anyFile.AST.Name.Name, p.fset, astFiles, info)

	// Remember the typechecking info, even if config.Check failed,
	// since we will get partial information.
	p.typesPkg = typesPkg
	p.typesInfo = info

	return err
}

// check function encapsulates the call to go/types.Config.Check method and
// recovers if the called method panics (see issue #59)
func check(config *types.Config, n string, fset *token.FileSet, astFiles []*ast.File, info *types.Info) (p *types.Package, err error) {
	defer func() {
		if r := recover(); r != nil {
			err, _ = r.(error)
			p = nil
			return
		}
	}()

	return config.Check(n, fset, astFiles, info)
}

// TypeOf returns the type of an expression.
func (p *Package) TypeOf(expr ast.Expr) types.Type {
	if p.typesInfo == nil {
		return nil
	}
	return p.typesInfo.TypeOf(expr)
}

type walker struct {
	nmap map[string]int
	has  map[string]int
}

func (w *walker) Visit(n ast.Node) ast.Visitor {
	fn, ok := n.(*ast.FuncDecl)
	if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
		return w
	}
	// TODO(dsymonds): We could check the signature to be more precise.
	recv := typeparams.ReceiverType(fn)
	if i, ok := w.nmap[fn.Name.Name]; ok {
		w.has[recv] |= i
	}
	return w
}

func (p *Package) scanSortable() {
	p.sortable = make(map[string]bool)

	// bitfield for which methods exist on each type.
	const (
		Len = 1 << iota
		Less
		Swap
	)
	nmap := map[string]int{"Len": Len, "Less": Less, "Swap": Swap}
	has := make(map[string]int)
	for _, f := range p.files {
		ast.Walk(&walker{nmap, has}, f.AST)
	}
	for typ, ms := range has {
		if ms == Len|Less|Swap {
			p.sortable[typ] = true
		}
	}
}

func (p *Package) lint(rules []Rule, config Config, failures chan Failure) {
	p.scanSortable()
	var wg sync.WaitGroup
	for _, file := range p.files {
		wg.Add(1)
		go (func(file *File) {
			file.lint(rules, config, failures)
			defer wg.Done()
		})(file)
	}
	wg.Wait()
}
