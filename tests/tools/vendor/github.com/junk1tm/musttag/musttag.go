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

// flags creates a flag set for the analyzer.
// The funcs slice will be filled with custom functions passed via CLI flags.
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

// New creates a new musttag analyzer.
// To report a custom function provide its description via Func,
// it will be added to the builtin ones.
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
		// TODO(junk1tm): print the name of the struct type as well?
		pass.Reportf(pos, "exported fields should be annotated with the %q tag", fn.Tag)
	}
)

// run starts the analysis.
func run(pass *analysis.Pass, funcs map[string]Func) (any, error) {
	type report struct {
		pos token.Pos // the position for report.
		tag string    // the missing struct tag.
	}

	// store previous reports to prevent reporting
	// the same struct more than once (if reportOnce is true).
	reports := make(map[report]struct{})

	walk := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	filter := []ast.Node{(*ast.CallExpr)(nil)}

	walk.Preorder(filter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return // not a function call.
		}

		callee := typeutil.StaticCallee(pass.TypesInfo, call)
		if callee == nil {
			return // not a static call.
		}

		fn, ok := funcs[callee.FullName()]
		if !ok {
			return // the function is not supported.
		}

		if len(call.Args) <= fn.ArgPos {
			return // TODO(junk1tm): return a proper error.
		}

		arg := call.Args[fn.ArgPos]
		if unary, ok := arg.(*ast.UnaryExpr); ok {
			arg = unary.X // e.g. json.Marshal(&foo)
		}

		initialPos := token.NoPos
		switch arg := arg.(type) {
		case *ast.Ident: // e.g. json.Marshal(foo)
			if arg.Obj == nil {
				return // e.g. json.Marshal(nil)
			}
			initialPos = arg.Obj.Pos()
		case *ast.CompositeLit: // e.g. json.Marshal(struct{}{})
			initialPos = arg.Pos()
		}

		t := pass.TypesInfo.TypeOf(arg)
		s, ok := parseStruct(t, initialPos)
		if !ok {
			return // not a struct argument.
		}

		reportPos, ok := checkStruct(s, fn.Tag, make(map[string]struct{}))
		if ok {
			return // nothing to report.
		}

		r := report{reportPos, fn.Tag}
		if _, ok := reports[r]; ok && reportOnce {
			return // already reported.
		}

		reportf(pass, reportPos, fn)
		reports[r] = struct{}{}
	})

	return nil, nil
}

// structInfo expands types.Struct with its position in the source code.
// If the struct is anonymous, Pos points to the corresponding identifier.
type structInfo struct {
	*types.Struct
	Pos token.Pos
}

// parseStruct parses the given types.Type, returning the underlying struct type.
// If it's a named type, the result will contain the position of its declaration,
// or the given token.Pos otherwise.
func parseStruct(t types.Type, pos token.Pos) (*structInfo, bool) {
	for {
		// unwrap pointers (if any) first.
		ptr, ok := t.(*types.Pointer)
		if !ok {
			break
		}
		t = ptr.Elem()
	}

	switch t := t.(type) {
	case *types.Named: // a struct of the named type.
		if s, ok := t.Underlying().(*types.Struct); ok {
			return &structInfo{Struct: s, Pos: t.Obj().Pos()}, true
		}
	case *types.Struct: // an anonymous struct.
		return &structInfo{Struct: t, Pos: pos}, true
	}

	return nil, false
}

// checkStruct recursively checks the given struct and returns the position for report,
// in case one of its fields is missing the tag.
func checkStruct(s *structInfo, tag string, visited map[string]struct{}) (token.Pos, bool) {
	visited[s.String()] = struct{}{}
	for i := 0; i < s.NumFields(); i++ {
		if !s.Field(i).Exported() {
			continue
		}

		st := reflect.StructTag(s.Tag(i))
		if _, ok := st.Lookup(tag); !ok && !s.Field(i).Embedded() {
			return s.Pos, false
		}

		t := s.Field(i).Type()
		nested, ok := parseStruct(t, s.Pos) // TODO(junk1tm): or s.Field(i).Pos()?
		if !ok {
			continue
		}
		if _, ok := visited[nested.String()]; ok {
			continue
		}
		if pos, ok := checkStruct(nested, tag, visited); !ok {
			return pos, false
		}
	}

	return token.NoPos, true
}
