// Package musttag implements the musttag analyzer.
package musttag

import (
	"flag"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Func describes a function call to look for, e.g. json.Marshal.
type Func struct {
	Name   string // Name is the full name of the function, including the package.
	Tag    string // Tag is the struct tag whose presence should be ensured.
	ArgPos int    // ArgPos is the position of the argument to check.
}

// builtin is a set of functions supported out of the box.
var builtin = []Func{
	{Name: "encoding/json.Marshal", Tag: "json", ArgPos: 0},
	{Name: "encoding/json.MarshalIndent", Tag: "json", ArgPos: 0},
	{Name: "encoding/json.Unmarshal", Tag: "json", ArgPos: 1},
	{Name: "(*encoding/json.Encoder).Encode", Tag: "json", ArgPos: 0},
	{Name: "(*encoding/json.Decoder).Decode", Tag: "json", ArgPos: 0},

	{Name: "encoding/xml.Marshal", Tag: "xml", ArgPos: 0},
	{Name: "encoding/xml.MarshalIndent", Tag: "xml", ArgPos: 0},
	{Name: "encoding/xml.Unmarshal", Tag: "xml", ArgPos: 1},
	{Name: "(*encoding/xml.Encoder).Encode", Tag: "xml", ArgPos: 0},
	{Name: "(*encoding/xml.Decoder).Decode", Tag: "xml", ArgPos: 0},
	{Name: "(*encoding/xml.Encoder).EncodeElement", Tag: "xml", ArgPos: 0},
	{Name: "(*encoding/xml.Decoder).DecodeElement", Tag: "xml", ArgPos: 0},

	{Name: "gopkg.in/yaml.v3.Marshal", Tag: "yaml", ArgPos: 0},
	{Name: "gopkg.in/yaml.v3.Unmarshal", Tag: "yaml", ArgPos: 1},
	{Name: "(*gopkg.in/yaml.v3.Encoder).Encode", Tag: "yaml", ArgPos: 0},
	{Name: "(*gopkg.in/yaml.v3.Decoder).Decode", Tag: "yaml", ArgPos: 0},

	{Name: "github.com/BurntSushi/toml.Unmarshal", Tag: "toml", ArgPos: 1},
	{Name: "github.com/BurntSushi/toml.Decode", Tag: "toml", ArgPos: 1},
	{Name: "github.com/BurntSushi/toml.DecodeFS", Tag: "toml", ArgPos: 2},
	{Name: "github.com/BurntSushi/toml.DecodeFile", Tag: "toml", ArgPos: 1},
	{Name: "(*github.com/BurntSushi/toml.Encoder).Encode", Tag: "toml", ArgPos: 0},
	{Name: "(*github.com/BurntSushi/toml.Decoder).Decode", Tag: "toml", ArgPos: 0},

	{Name: "github.com/mitchellh/mapstructure.Decode", Tag: "mapstructure", ArgPos: 1},
	{Name: "github.com/mitchellh/mapstructure.DecodeMetadata", Tag: "mapstructure", ArgPos: 1},
	{Name: "github.com/mitchellh/mapstructure.WeakDecode", Tag: "mapstructure", ArgPos: 1},
	{Name: "github.com/mitchellh/mapstructure.WeakDecodeMetadata", Tag: "mapstructure", ArgPos: 1},
}

// flags creates a flag set for the analyzer. The funcs slice will be filled
// with custom functions passed via CLI flags.
func flags(funcs *[]Func) flag.FlagSet {
	fs := flag.NewFlagSet("musttag", flag.ContinueOnError)
	fs.Func("fn", "report custom function (name:tag:argpos)", func(s string) error {
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

// New creates a new musttag analyzer. To report a custom function provide its
// description via Func, it will be added to the builtin ones.
func New(funcs ...Func) *analysis.Analyzer {
	var flagFuncs []Func
	return &analysis.Analyzer{
		Name:     "musttag",
		Doc:      "enforce field tags in (un)marshaled structs",
		Flags:    flags(&flagFuncs),
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			l := len(builtin) + len(funcs) + len(flagFuncs)
			m := make(map[string]Func, l)
			toMap := func(slice []Func) {
				for _, fn := range slice {
					m[fn.Name] = fn
				}
			}
			toMap(builtin)
			toMap(funcs)
			toMap(flagFuncs)
			return run(pass, m)
		},
	}
}

// for tests only.
var (
	// should the same struct be reported only once for the same tag?
	reportOnce = true

	// reportf is a wrapper for pass.Reportf (as a variable, so it could be mocked in tests).
	reportf = func(pass *analysis.Pass, pos token.Pos, fn Func) {
		pass.Reportf(pos, "exported fields should be annotated with the %q tag", fn.Tag)
	}
)

// run starts the analysis.
func run(pass *analysis.Pass, funcs map[string]Func) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	filter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	type report struct {
		pos token.Pos
		tag string
	}
	reported := make(map[report]struct{})

	insp.Preorder(filter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		callee := typeutil.StaticCallee(pass.TypesInfo, call)
		if callee == nil {
			return
		}

		fn, ok := funcs[callee.FullName()]
		if !ok {
			return
		}

		s, pos, ok := structAndPos(pass, call.Args[fn.ArgPos])
		if !ok {
			return
		}

		if ok := checkStruct(s, fn.Tag, &pos); ok {
			return
		}

		r := report{pos, fn.Tag}
		if _, ok := reported[r]; ok && reportOnce {
			return
		}

		reportf(pass, pos, fn)
		reported[r] = struct{}{}
	})

	return nil, nil
}

// structAndPos analyses the given expression and returns the struct to check
// and the position to report if needed.
func structAndPos(pass *analysis.Pass, expr ast.Expr) (*types.Struct, token.Pos, bool) {
	t := pass.TypesInfo.TypeOf(expr)
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	switch t := t.(type) {
	case *types.Named: // named type
		s, ok := t.Underlying().(*types.Struct)
		if ok {
			return s, t.Obj().Pos(), true
		}

	case *types.Struct: // anonymous struct
		if unary, ok := expr.(*ast.UnaryExpr); ok {
			expr = unary.X // &x
		}
		//nolint:gocritic // commentedOutCode: these are examples
		switch arg := expr.(type) {
		case *ast.Ident: // var x struct{}; json.Marshal(x)
			return t, arg.Obj.Pos(), true
		case *ast.CompositeLit: // json.Marshal(struct{}{})
			return t, arg.Pos(), true
		}
	}

	return nil, 0, false
}

// checkStruct checks that exported fields of the given struct are annotated
// with the tag and updates the position to report in case a nested struct of a
// named type is found.
func checkStruct(s *types.Struct, tag string, pos *token.Pos) (ok bool) {
	for i := 0; i < s.NumFields(); i++ {
		if !s.Field(i).Exported() {
			continue
		}

		st := reflect.StructTag(s.Tag(i))
		if _, ok := st.Lookup(tag); !ok {
			// it's ok for embedded types not to be tagged,
			// see https://github.com/junk1tm/musttag/issues/12
			if !s.Field(i).Embedded() {
				return false
			}
		}

		// check if the field is a nested struct.
		t := s.Field(i).Type()
		if ptr, ok := t.(*types.Pointer); ok {
			t = ptr.Elem()
		}
		nested, ok := t.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		if ok := checkStruct(nested, tag, pos); ok {
			continue
		}
		// update the position to point to the named type.
		if named, ok := t.(*types.Named); ok {
			*pos = named.Obj().Pos()
		}
		return false
	}

	return true
}
