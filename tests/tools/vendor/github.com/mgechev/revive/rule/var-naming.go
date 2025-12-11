package rule

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"sync"

	"github.com/mgechev/revive/lint"
)

var anyCapsRE = regexp.MustCompile(`[A-Z]`)

// regexp for constant names like `SOME_CONST`, `SOME_CONST_2`, `X123_3`, `_SOME_PRIVATE_CONST` (#851, #865)
var upperCaseConstRE = regexp.MustCompile(`^_?[A-Z][A-Z\d]*(_[A-Z\d]+)*$`)

// VarNamingRule lints given else constructs.
type VarNamingRule struct {
	configured            bool
	allowlist             []string
	blocklist             []string
	upperCaseConst        bool // if true - allows to use UPPER_SOME_NAMES for constants
	skipPackageNameChecks bool
	sync.Mutex
}

func (r *VarNamingRule) configure(arguments lint.Arguments) {
	r.Lock()
	defer r.Unlock()
	if r.configured {
		return
	}

	r.configured = true
	if len(arguments) >= 1 {
		r.allowlist = getList(arguments[0], "allowlist")
	}

	if len(arguments) >= 2 {
		r.blocklist = getList(arguments[1], "blocklist")
	}

	if len(arguments) >= 3 {
		// not pretty code because should keep compatibility with TOML (no mixed array types) and new map parameters
		thirdArgument := arguments[2]
		asSlice, ok := thirdArgument.([]any)
		if !ok {
			panic(fmt.Sprintf("Invalid third argument to the var-naming rule. Expecting a %s of type slice, got %T", "options", arguments[2]))
		}
		if len(asSlice) != 1 {
			panic(fmt.Sprintf("Invalid third argument to the var-naming rule. Expecting a %s of type slice, of len==1, but %d", "options", len(asSlice)))
		}
		args, ok := asSlice[0].(map[string]any)
		if !ok {
			panic(fmt.Sprintf("Invalid third argument to the var-naming rule. Expecting a %s of type slice, of len==1, with map, but %T", "options", asSlice[0]))
		}
		r.upperCaseConst = fmt.Sprint(args["upperCaseConst"]) == "true"
		r.skipPackageNameChecks = fmt.Sprint(args["skipPackageNameChecks"]) == "true"
	}
}

func (r *VarNamingRule) applyPackageCheckRules(walker *lintNames) {
	// Package names need slightly different handling than other names.
	if strings.Contains(walker.fileAst.Name.Name, "_") && !strings.HasSuffix(walker.fileAst.Name.Name, "_test") {
		walker.onFailure(lint.Failure{
			Failure:    "don't use an underscore in package name",
			Confidence: 1,
			Node:       walker.fileAst.Name,
			Category:   "naming",
		})
	}
	if anyCapsRE.MatchString(walker.fileAst.Name.Name) {
		walker.onFailure(lint.Failure{
			Failure:    fmt.Sprintf("don't use MixedCaps in package name; %s should be %s", walker.fileAst.Name.Name, strings.ToLower(walker.fileAst.Name.Name)),
			Confidence: 1,
			Node:       walker.fileAst.Name,
			Category:   "naming",
		})
	}

}

// Apply applies the rule to given file.
func (r *VarNamingRule) Apply(file *lint.File, arguments lint.Arguments) []lint.Failure {
	r.configure(arguments)

	var failures []lint.Failure

	fileAst := file.AST

	walker := lintNames{
		file:      file,
		fileAst:   fileAst,
		allowlist: r.allowlist,
		blocklist: r.blocklist,
		onFailure: func(failure lint.Failure) {
			failures = append(failures, failure)
		},
		upperCaseConst: r.upperCaseConst,
	}

	if !r.skipPackageNameChecks {
		r.applyPackageCheckRules(&walker)
	}

	ast.Walk(&walker, fileAst)

	return failures
}

// Name returns the rule name.
func (*VarNamingRule) Name() string {
	return "var-naming"
}

func (w *lintNames) checkList(fl *ast.FieldList, thing string) {
	if fl == nil {
		return
	}
	for _, f := range fl.List {
		for _, id := range f.Names {
			w.check(id, thing)
		}
	}
}

func (w *lintNames) check(id *ast.Ident, thing string) {
	if id.Name == "_" {
		return
	}
	if knownNameExceptions[id.Name] {
		return
	}

	// #851 upperCaseConst support
	// if it's const
	if thing == token.CONST.String() && w.upperCaseConst && upperCaseConstRE.MatchString(id.Name) {
		return
	}

	// Handle two common styles from other languages that don't belong in Go.
	if len(id.Name) >= 5 && allCapsRE.MatchString(id.Name) && strings.Contains(id.Name, "_") {
		w.onFailure(lint.Failure{
			Failure:    "don't use ALL_CAPS in Go names; use CamelCase",
			Confidence: 0.8,
			Node:       id,
			Category:   "naming",
		})
		return
	}

	should := lint.Name(id.Name, w.allowlist, w.blocklist)
	if id.Name == should {
		return
	}

	if len(id.Name) > 2 && strings.Contains(id.Name[1:], "_") {
		w.onFailure(lint.Failure{
			Failure:    fmt.Sprintf("don't use underscores in Go names; %s %s should be %s", thing, id.Name, should),
			Confidence: 0.9,
			Node:       id,
			Category:   "naming",
		})
		return
	}
	w.onFailure(lint.Failure{
		Failure:    fmt.Sprintf("%s %s should be %s", thing, id.Name, should),
		Confidence: 0.8,
		Node:       id,
		Category:   "naming",
	})
}

type lintNames struct {
	file           *lint.File
	fileAst        *ast.File
	onFailure      func(lint.Failure)
	allowlist      []string
	blocklist      []string
	upperCaseConst bool
}

func (w *lintNames) Visit(n ast.Node) ast.Visitor {
	switch v := n.(type) {
	case *ast.AssignStmt:
		if v.Tok == token.ASSIGN {
			return w
		}
		for _, exp := range v.Lhs {
			if id, ok := exp.(*ast.Ident); ok {
				w.check(id, "var")
			}
		}
	case *ast.FuncDecl:
		funcName := v.Name.Name
		if w.file.IsTest() &&
			(strings.HasPrefix(funcName, "Example") ||
				strings.HasPrefix(funcName, "Test") ||
				strings.HasPrefix(funcName, "Benchmark") ||
				strings.HasPrefix(funcName, "Fuzz")) {
			return w
		}

		thing := "func"
		if v.Recv != nil {
			thing = "method"
		}

		// Exclude naming warnings for functions that are exported to C but
		// not exported in the Go API.
		// See https://github.com/golang/lint/issues/144.
		if ast.IsExported(v.Name.Name) || !isCgoExported(v) {
			w.check(v.Name, thing)
		}

		w.checkList(v.Type.Params, thing+" parameter")
		w.checkList(v.Type.Results, thing+" result")
	case *ast.GenDecl:
		if v.Tok == token.IMPORT {
			return w
		}

		thing := v.Tok.String()
		for _, spec := range v.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				w.check(s.Name, thing)
			case *ast.ValueSpec:
				for _, id := range s.Names {
					w.check(id, thing)
				}
			}
		}
	case *ast.InterfaceType:
		// Do not check interface method names.
		// They are often constrained by the method names of concrete types.
		for _, x := range v.Methods.List {
			ft, ok := x.Type.(*ast.FuncType)
			if !ok { // might be an embedded interface name
				continue
			}
			w.checkList(ft.Params, "interface method parameter")
			w.checkList(ft.Results, "interface method result")
		}
	case *ast.RangeStmt:
		if v.Tok == token.ASSIGN {
			return w
		}
		if id, ok := v.Key.(*ast.Ident); ok {
			w.check(id, "range var")
		}
		if id, ok := v.Value.(*ast.Ident); ok {
			w.check(id, "range var")
		}
	case *ast.StructType:
		for _, f := range v.Fields.List {
			for _, id := range f.Names {
				w.check(id, "struct field")
			}
		}
	}
	return w
}

func getList(arg any, argName string) []string {
	temp, ok := arg.([]any)
	if !ok {
		panic(fmt.Sprintf("Invalid argument to the var-naming rule. Expecting a %s of type slice with initialisms, got %T", argName, arg))
	}
	var list []string
	for _, v := range temp {
		if val, ok := v.(string); ok {
			list = append(list, val)
		} else {
			panic(fmt.Sprintf("Invalid %s values of the var-naming rule. Expecting slice of strings but got element of type %T", val, arg))
		}
	}
	return list
}
