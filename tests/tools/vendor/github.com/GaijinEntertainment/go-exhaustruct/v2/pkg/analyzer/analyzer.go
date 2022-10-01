package analyzer

import (
	"errors"
	"flag"
	"go/ast"
	"go/types"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	ErrEmptyPattern = errors.New("pattern can't be empty")
)

type analyzer struct {
	include PatternsList
	exclude PatternsList

	typesProcessCache   map[types.Type]bool
	typesProcessCacheMu sync.RWMutex

	structFieldsCache   map[types.Type]*StructFields
	structFieldsCacheMu sync.RWMutex
}

// NewAnalyzer returns a go/analysis-compatible analyzer.
//   -i arguments adds include patterns
//   -e arguments adds exclude patterns
func NewAnalyzer(include []string, exclude []string) (*analysis.Analyzer, error) {
	a := analyzer{ //nolint:exhaustruct
		typesProcessCache: map[types.Type]bool{},

		structFieldsCache: map[types.Type]*StructFields{},
	}

	var err error

	a.include, err = newPatternsList(include)
	if err != nil {
		return nil, err
	}

	a.exclude, err = newPatternsList(exclude)
	if err != nil {
		return nil, err
	}

	return &analysis.Analyzer{ //nolint:exhaustruct
		Name:     "exhaustruct",
		Doc:      "Checks if all structure fields are initialized",
		Run:      a.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Flags:    a.newFlagSet(),
	}, nil
}

func (a *analyzer) newFlagSet() flag.FlagSet {
	fs := flag.NewFlagSet("exhaustruct flags", flag.PanicOnError)

	fs.Var(
		&reListVar{values: &a.include},
		"i",
		"Regular expression to match struct packages and names, can receive multiple flags",
	)
	fs.Var(
		&reListVar{values: &a.exclude},
		"e",
		"Regular expression to exclude struct packages and names, can receive multiple flags",
	)

	return *fs
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector) //nolint:forcetypeassert

	nodeTypes := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.ReturnStmt)(nil),
	}

	insp.Preorder(nodeTypes, a.newVisitor(pass))

	return nil, nil //nolint:nilnil
}

//nolint:cyclop
func (a *analyzer) newVisitor(pass *analysis.Pass) func(node ast.Node) {
	var ret *ast.ReturnStmt

	return func(node ast.Node) {
		if retLit, ok := node.(*ast.ReturnStmt); ok {
			// save return statement for future (to detect error-containing returns)
			ret = retLit

			return
		}

		lit, _ := node.(*ast.CompositeLit)
		if lit.Type == nil {
			// we're not interested in non-typed literals
			return
		}

		typ := pass.TypesInfo.TypeOf(lit.Type)
		if typ == nil {
			return
		}

		strct, ok := typ.Underlying().(*types.Struct)
		if !ok {
			// we also not interested in non-structure literals
			return
		}

		strctName := exprName(lit.Type)
		if strctName == "" {
			return
		}

		if !a.shouldProcessType(typ) {
			return
		}

		if len(lit.Elts) == 0 && ret != nil {
			if ret.End() < lit.Pos() {
				// we're outside last return statement
				ret = nil
			} else if returnContainsLiteral(ret, lit) && returnContainsError(ret, pass) {
				// we're okay with empty literals in return statements with non-nil errors, like
				// `return my.Struct{}, fmt.Errorf("non-nil error!")`
				return
			}
		}

		missingFields := a.structMissingFields(lit, strct, strings.HasPrefix(typ.String(), pass.Pkg.Path()+"."))

		if len(missingFields) == 1 {
			pass.Reportf(node.Pos(), "%s is missing in %s", missingFields[0], strctName)
		} else if len(missingFields) > 1 {
			pass.Reportf(node.Pos(), "%s are missing in %s", strings.Join(missingFields, ", "), strctName)
		}
	}
}

func (a *analyzer) shouldProcessType(typ types.Type) bool {
	if len(a.include) == 0 && len(a.exclude) == 0 {
		// skip whole part with cache, since we have no restrictions and have to check everything
		return true
	}

	a.typesProcessCacheMu.RLock()
	v, ok := a.typesProcessCache[typ]
	a.typesProcessCacheMu.RUnlock()

	if !ok {
		a.typesProcessCacheMu.Lock()
		defer a.typesProcessCacheMu.Unlock()

		v = true
		typStr := typ.String()

		if len(a.include) > 0 && !a.include.MatchesAny(typStr) {
			v = false
		}

		if v && a.exclude.MatchesAny(typStr) {
			v = false
		}

		a.typesProcessCache[typ] = v
	}

	return v
}

func (a *analyzer) structMissingFields(lit *ast.CompositeLit, strct *types.Struct, private bool) []string {
	keys, unnamed := literalKeys(lit)
	fields := a.structFields(strct)

	if unnamed {
		if private {
			return fields.All[len(keys):]
		}

		return fields.Public[len(keys):]
	}

	if private {
		return difference(fields.AllRequired, keys)
	}

	return difference(fields.PublicRequired, keys)
}

func (a *analyzer) structFields(strct *types.Struct) *StructFields {
	typ := strct.Underlying()

	a.structFieldsCacheMu.RLock()
	fields, ok := a.structFieldsCache[typ]
	a.structFieldsCacheMu.RUnlock()

	if !ok {
		a.structFieldsCacheMu.Lock()
		defer a.structFieldsCacheMu.Unlock()

		fields = NewStructFields(strct)
		a.structFieldsCache[typ] = fields
	}

	return fields
}

func returnContainsLiteral(ret *ast.ReturnStmt, lit *ast.CompositeLit) bool {
	for _, result := range ret.Results {
		if l, ok := result.(*ast.CompositeLit); ok {
			if lit == l {
				return true
			}
		}
	}

	return false
}

func returnContainsError(ret *ast.ReturnStmt, pass *analysis.Pass) bool {
	for _, result := range ret.Results {
		if pass.TypesInfo.TypeOf(result).String() == "error" {
			return true
		}
	}

	return false
}

func literalKeys(lit *ast.CompositeLit) (keys []string, unnamed bool) {
	for _, elt := range lit.Elts {
		if k, ok := elt.(*ast.KeyValueExpr); ok {
			if ident, ok := k.Key.(*ast.Ident); ok {
				keys = append(keys, ident.Name)
			}

			continue
		}

		// in case we deal with unnamed initialization - no need to iterate over all
		// elements - simply create slice with proper size
		unnamed = true
		keys = make([]string, len(lit.Elts))

		return
	}

	return
}

// difference returns elements that are in `a` and not in `b`.
func difference(a, b []string) (diff []string) {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}

	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func exprName(expr ast.Expr) string {
	if i, ok := expr.(*ast.Ident); ok {
		return i.Name
	}

	s, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	return s.Sel.Name
}
