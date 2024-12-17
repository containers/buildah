// Package musttag implements the musttag analyzer.
package musttag

import (
	"flag"
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Func describes a function call to look for, e.g. [json.Marshal].
type Func struct {
	Name   string // The full name of the function, including the package.
	Tag    string // The struct tag whose presence should be ensured.
	ArgPos int    // The position of the argument to check.

	// a list of interface names (including the package);
	// if at least one is implemented by the argument, no check is performed.
	ifaceWhitelist []string
}

// New creates a new musttag analyzer.
// To report a custom function, provide its description as [Func].
func New(funcs ...Func) *analysis.Analyzer {
	var flagFuncs []Func
	return &analysis.Analyzer{
		Name:     "musttag",
		Doc:      "enforce field tags in (un)marshaled structs",
		Flags:    flags(&flagFuncs),
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			l := len(builtins) + len(funcs) + len(flagFuncs)
			allFuncs := make(map[string]Func, l)

			merge := func(slice []Func) {
				for _, fn := range slice {
					allFuncs[fn.Name] = fn
				}
			}
			merge(builtins)
			merge(funcs)
			merge(flagFuncs)

			mainModule, err := getMainModule()
			if err != nil {
				return nil, err
			}

			return run(pass, mainModule, allFuncs)
		},
	}
}

func flags(funcs *[]Func) flag.FlagSet {
	fs := flag.NewFlagSet("musttag", flag.ContinueOnError)
	fs.Func("fn", "report a custom function (name:tag:arg-pos)", func(s string) error {
		parts := strings.Split(s, ":")
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
			return strconv.ErrSyntax
		}
		pos, err := strconv.Atoi(parts[2])
		if err != nil {
			return err
		}
		*funcs = append(*funcs, Func{
			Name:   parts[0],
			Tag:    parts[1],
			ArgPos: pos,
		})
		return nil
	})
	return *fs
}

func run(pass *analysis.Pass, mainModule string, funcs map[string]Func) (_ any, err error) {
	visit := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	filter := []ast.Node{(*ast.CallExpr)(nil)}

	visit.Preorder(filter, func(node ast.Node) {
		if err != nil {
			return // there is already an error.
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return // not a function call.
		}

		callee := typeutil.StaticCallee(pass.TypesInfo, call)
		if callee == nil {
			return // not a static call.
		}

		fn, ok := funcs[cutVendor(callee.FullName())]
		if !ok {
			return // unsupported function.
		}

		if len(call.Args) <= fn.ArgPos {
			err = fmt.Errorf("musttag: Func.ArgPos cannot be %d: %s accepts only %d argument(s)", fn.ArgPos, fn.Name, len(call.Args))
			return
		}

		arg := call.Args[fn.ArgPos]
		if ident, ok := arg.(*ast.Ident); ok && ident.Obj == nil {
			return // e.g. json.Marshal(nil)
		}

		typ := pass.TypesInfo.TypeOf(arg)
		if typ == nil {
			return // no type info found.
		}

		checker := checker{
			mainModule:     mainModule,
			seenTypes:      make(map[string]struct{}),
			ifaceWhitelist: fn.ifaceWhitelist,
			imports:        pass.Pkg.Imports(),
		}

		if valid := checker.checkType(typ, fn.Tag); valid {
			return // nothing to report.
		}

		pass.Reportf(arg.Pos(), "the given struct should be annotated with the `%s` tag", fn.Tag)
	})

	return nil, err
}

type checker struct {
	mainModule     string
	seenTypes      map[string]struct{}
	ifaceWhitelist []string
	imports        []*types.Package
}

func (c *checker) checkType(typ types.Type, tag string) bool {
	if _, ok := c.seenTypes[typ.String()]; ok {
		return true // already checked.
	}
	c.seenTypes[typ.String()] = struct{}{}

	styp, ok := c.parseStruct(typ)
	if !ok {
		return true // not a struct.
	}

	return c.checkStruct(styp, tag)
}

// recursively unwrap a type until we get to an underlying
// raw struct type that should have its fields checked
//
//	SomeStruct -> struct{SomeStructField: ... }
//	[]*SomeStruct -> struct{SomeStructField: ... }
//	...
//
// exits early if it hits a type that implements a whitelisted interface
func (c *checker) parseStruct(typ types.Type) (*types.Struct, bool) {
	if implementsInterface(typ, c.ifaceWhitelist, c.imports) {
		return nil, false // the type implements a Marshaler interface; see issue #64.
	}

	switch typ := typ.(type) {
	case *types.Pointer:
		return c.parseStruct(typ.Elem())

	case *types.Array:
		return c.parseStruct(typ.Elem())

	case *types.Slice:
		return c.parseStruct(typ.Elem())

	case *types.Map:
		return c.parseStruct(typ.Elem())

	case *types.Named: // a struct of the named type.
		pkg := typ.Obj().Pkg()
		if pkg == nil {
			return nil, false
		}
		if !strings.HasPrefix(pkg.Path(), c.mainModule) {
			return nil, false
		}
		styp, ok := typ.Underlying().(*types.Struct)
		if !ok {
			return nil, false
		}
		return styp, true

	case *types.Struct: // an anonymous struct.
		return typ, true

	default:
		return nil, false
	}
}

func (c *checker) checkStruct(styp *types.Struct, tag string) (valid bool) {
	for i := 0; i < styp.NumFields(); i++ {
		field := styp.Field(i)
		if !field.Exported() {
			continue
		}

		tagValue, ok := reflect.StructTag(styp.Tag(i)).Lookup(tag)
		if !ok {
			// tag is not required for embedded types; see issue #12.
			if !field.Embedded() {
				return false
			}
		}

		// Do not recurse into ignored fields.
		if tagValue == "-" {
			continue
		}

		if valid := c.checkType(field.Type(), tag); !valid {
			return false
		}
	}

	return true
}

func implementsInterface(typ types.Type, ifaces []string, imports []*types.Package) bool {
	findScope := func(pkgName string) (*types.Scope, bool) {
		// fast path: check direct imports (e.g. looking for "encoding/json.Marshaler").
		for _, direct := range imports {
			if pkgName == cutVendor(direct.Path()) {
				return direct.Scope(), true
			}
		}
		// slow path: check indirect imports (e.g. looking for "encoding.TextMarshaler").
		// TODO: only check indirect imports from the package (e.g. "encoding/json") of the analyzed function (e.g. "encoding/json.Marshal").
		for _, direct := range imports {
			for _, indirect := range direct.Imports() {
				if pkgName == cutVendor(indirect.Path()) {
					return indirect.Scope(), true
				}
			}
		}
		return nil, false
	}

	for _, ifacePath := range ifaces {
		// "encoding/json.Marshaler" -> "encoding/json" + "Marshaler"
		idx := strings.LastIndex(ifacePath, ".")
		if idx == -1 {
			continue
		}
		pkgName, ifaceName := ifacePath[:idx], ifacePath[idx+1:]

		scope, ok := findScope(pkgName)
		if !ok {
			continue
		}
		obj := scope.Lookup(ifaceName)
		if obj == nil {
			continue
		}
		iface, ok := obj.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		if types.Implements(typ, iface) || types.Implements(types.NewPointer(typ), iface) {
			return true
		}
	}

	return false
}
