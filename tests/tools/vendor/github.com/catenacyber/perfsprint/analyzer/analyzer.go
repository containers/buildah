package analyzer

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"golang.org/x/tools/go/analysis"
)

type perfSprint struct {
	intConv    bool
	errError   bool
	errorf     bool
	sprintf1   bool
	fiximports bool
	strconcat  bool
}

func newPerfSprint() *perfSprint {
	return &perfSprint{
		intConv:    true,
		errError:   false,
		errorf:     true,
		sprintf1:   true,
		fiximports: true,
		strconcat:  true,
	}
}

func New() *analysis.Analyzer {
	n := newPerfSprint()
	r := &analysis.Analyzer{
		Name:     "perfsprint",
		Doc:      "Checks that fmt.Sprintf can be replaced with a faster alternative.",
		Run:      n.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
	r.Flags.BoolVar(&n.intConv, "int-conversion", true, "optimizes even if it requires an int or uint type cast")
	r.Flags.BoolVar(&n.errError, "err-error", false, "optimizes into err.Error() even if it is only equivalent for non-nil errors")
	r.Flags.BoolVar(&n.errorf, "errorf", true, "optimizes fmt.Errorf")
	r.Flags.BoolVar(&n.sprintf1, "sprintf1", true, "optimizes fmt.Sprintf with only one argument")
	r.Flags.BoolVar(&n.fiximports, "fiximports", true, "fix needed imports from other fixes")
	r.Flags.BoolVar(&n.strconcat, "strconcat", true, "optimizes into strings concatenation")
	return r
}

// true if verb is a format string that could be replaced with concatenation.
func isConcatable(verb string) bool {
	hasPrefix :=
		(strings.HasPrefix(verb, "%s") && !strings.Contains(verb, "%[1]s")) ||
			(strings.HasPrefix(verb, "%[1]s") && !strings.Contains(verb, "%s"))
	hasSuffix :=
		(strings.HasSuffix(verb, "%s") && !strings.Contains(verb, "%[1]s")) ||
			(strings.HasSuffix(verb, "%[1]s") && !strings.Contains(verb, "%s"))

	if strings.Count(verb, "%[1]s") > 1 {
		return false
	}
	return (hasPrefix || hasSuffix) && !(hasPrefix && hasSuffix)
}

func (n *perfSprint) run(pass *analysis.Pass) (interface{}, error) {
	var fmtSprintObj, fmtSprintfObj, fmtErrorfObj types.Object
	for _, pkg := range pass.Pkg.Imports() {
		if pkg.Path() == "fmt" {
			fmtSprintObj = pkg.Scope().Lookup("Sprint")
			fmtSprintfObj = pkg.Scope().Lookup("Sprintf")
			fmtErrorfObj = pkg.Scope().Lookup("Errorf")
		}
	}
	if fmtSprintfObj == nil && fmtSprintObj == nil && fmtErrorfObj == nil {
		return nil, nil
	}
	removedFmtUsages := make(map[string]int)
	neededPackages := make(map[string]map[string]bool)

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	insp.Preorder(nodeFilter, func(node ast.Node) {
		call := node.(*ast.CallExpr)
		called, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		calledObj := pass.TypesInfo.ObjectOf(called.Sel)

		var (
			fn    string
			verb  string
			value ast.Expr
			err   error
		)
		switch {
		case calledObj == fmtErrorfObj && len(call.Args) == 1:
			if n.errorf {
				fn = "fmt.Errorf"
				verb = "%s"
				value = call.Args[0]
			} else {
				return
			}

		case calledObj == fmtSprintObj && len(call.Args) == 1:
			fn = "fmt.Sprint"
			verb = "%v"
			value = call.Args[0]

		case calledObj == fmtSprintfObj && len(call.Args) == 1:
			if n.sprintf1 {
				fn = "fmt.Sprintf"
				verb = "%s"
				value = call.Args[0]
			} else {
				return
			}

		case calledObj == fmtSprintfObj && len(call.Args) == 2:
			verbLit, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return
			}
			verb, err = strconv.Unquote(verbLit.Value)
			if err != nil {
				// Probably unreachable.
				return
			}
			// one single explicit arg is simplified
			if strings.HasPrefix(verb, "%[1]") {
				verb = "%" + verb[4:]
			}

			fn = "fmt.Sprintf"
			value = call.Args[1]

		default:
			return
		}

		switch verb {
		default:
			if fn == "fmt.Sprintf" && isConcatable(verb) && n.strconcat {
				break
			}
			return
		case "%d", "%v", "%x", "%t", "%s":
		}

		valueType := pass.TypesInfo.TypeOf(value)
		a, isArray := valueType.(*types.Array)
		s, isSlice := valueType.(*types.Slice)

		var d *analysis.Diagnostic
		switch {
		case isBasicType(valueType, types.String) && oneOf(verb, "%v", "%s"):
			fname := pass.Fset.File(call.Pos()).Name()
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			removedFmtUsages[fname]++
			if fn == "fmt.Errorf" {
				neededPackages[fname]["errors"] = true
				d = &analysis.Diagnostic{
					Pos:     call.Pos(),
					End:     call.End(),
					Message: fn + " can be replaced with errors.New",
					SuggestedFixes: []analysis.SuggestedFix{
						{
							Message: "Use errors.New",
							TextEdits: []analysis.TextEdit{{
								Pos:     call.Pos(),
								End:     value.Pos(),
								NewText: []byte("errors.New("),
							}},
						},
					},
				}
			} else {
				d = &analysis.Diagnostic{
					Pos:     call.Pos(),
					End:     call.End(),
					Message: fn + " can be replaced with just using the string",
					SuggestedFixes: []analysis.SuggestedFix{
						{
							Message: "Just use string value",
							TextEdits: []analysis.TextEdit{{
								Pos:     call.Pos(),
								End:     call.End(),
								NewText: []byte(formatNode(pass.Fset, value)),
							}},
						},
					},
				}
			}
		case types.Implements(valueType, errIface) && oneOf(verb, "%v", "%s") && n.errError:
			// known false positive if this error is nil
			// fmt.Sprint(nil) does not panic like nil.Error() does
			errMethodCall := formatNode(pass.Fset, value) + ".Error()"
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with " + errMethodCall,
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use " + errMethodCall,
						TextEdits: []analysis.TextEdit{{
							Pos:     call.Pos(),
							End:     call.End(),
							NewText: []byte(errMethodCall),
						}},
					},
				},
			}

		case isBasicType(valueType, types.Bool) && oneOf(verb, "%v", "%t"):
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["strconv"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster strconv.FormatBool",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use strconv.FormatBool",
						TextEdits: []analysis.TextEdit{{
							Pos:     call.Pos(),
							End:     value.Pos(),
							NewText: []byte("strconv.FormatBool("),
						}},
					},
				},
			}

		case isArray && isBasicType(a.Elem(), types.Uint8) && oneOf(verb, "%x"):
			if _, ok := value.(*ast.Ident); !ok {
				// Doesn't support array literals.
				return
			}

			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["encoding/hex"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster hex.EncodeToString",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use hex.EncodeToString",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     value.Pos(),
								NewText: []byte("hex.EncodeToString("),
							},
							{
								Pos:     value.End(),
								End:     value.End(),
								NewText: []byte("[:]"),
							},
						},
					},
				},
			}
		case isSlice && isBasicType(s.Elem(), types.Uint8) && oneOf(verb, "%x"):
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["encoding/hex"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster hex.EncodeToString",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use hex.EncodeToString",
						TextEdits: []analysis.TextEdit{{
							Pos:     call.Pos(),
							End:     value.Pos(),
							NewText: []byte("hex.EncodeToString("),
						}},
					},
				},
			}

		case isBasicType(valueType, types.Int8, types.Int16, types.Int32) && oneOf(verb, "%v", "%d") && n.intConv:
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["strconv"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster strconv.Itoa",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use strconv.Itoa",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     value.Pos(),
								NewText: []byte("strconv.Itoa(int("),
							},
							{
								Pos:     value.End(),
								End:     value.End(),
								NewText: []byte(")"),
							},
						},
					},
				},
			}
		case isBasicType(valueType, types.Int) && oneOf(verb, "%v", "%d"):
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["strconv"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster strconv.Itoa",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use strconv.Itoa",
						TextEdits: []analysis.TextEdit{{
							Pos:     call.Pos(),
							End:     value.Pos(),
							NewText: []byte("strconv.Itoa("),
						}},
					},
				},
			}
		case isBasicType(valueType, types.Int64) && oneOf(verb, "%v", "%d"):
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["strconv"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster strconv.FormatInt",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use strconv.FormatInt",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     value.Pos(),
								NewText: []byte("strconv.FormatInt("),
							},
							{
								Pos:     value.End(),
								End:     value.End(),
								NewText: []byte(", 10"),
							},
						},
					},
				},
			}

		case isBasicType(valueType, types.Uint8, types.Uint16, types.Uint32, types.Uint) && oneOf(verb, "%v", "%d", "%x") && n.intConv:
			base := []byte("), 10")
			if verb == "%x" {
				base = []byte("), 16")
			}
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["strconv"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster strconv.FormatUint",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use strconv.FormatUint",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     value.Pos(),
								NewText: []byte("strconv.FormatUint(uint64("),
							},
							{
								Pos:     value.End(),
								End:     value.End(),
								NewText: base,
							},
						},
					},
				},
			}
		case isBasicType(valueType, types.Uint64) && oneOf(verb, "%v", "%d", "%x"):
			base := []byte(", 10")
			if verb == "%x" {
				base = []byte(", 16")
			}
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			_, ok := neededPackages[fname]
			if !ok {
				neededPackages[fname] = make(map[string]bool)
			}
			neededPackages[fname]["strconv"] = true
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with faster strconv.FormatUint",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use strconv.FormatUint",
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     value.Pos(),
								NewText: []byte("strconv.FormatUint("),
							},
							{
								Pos:     value.End(),
								End:     value.End(),
								NewText: base,
							},
						},
					},
				},
			}
		case isBasicType(valueType, types.String) && fn == "fmt.Sprintf" && isConcatable(verb):
			var fix string
			if strings.HasSuffix(verb, "%s") {
				fix = strconv.Quote(verb[:len(verb)-2]) + "+" + formatNode(pass.Fset, value)
			} else if strings.HasSuffix(verb, "%[1]s") {
				fix = strconv.Quote(verb[:len(verb)-5]) + "+" + formatNode(pass.Fset, value)
			} else if strings.HasPrefix(verb, "%s") {
				fix = formatNode(pass.Fset, value) + "+" + strconv.Quote(verb[2:])
			} else {
				fix = formatNode(pass.Fset, value) + "+" + strconv.Quote(verb[5:])
			}
			fname := pass.Fset.File(call.Pos()).Name()
			removedFmtUsages[fname]++
			d = &analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: fn + " can be replaced with string concatenation",
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Use string concatenation",
						TextEdits: []analysis.TextEdit{{
							Pos:     call.Pos(),
							End:     call.End(),
							NewText: []byte(fix),
						}},
					},
				},
			}
		}

		if d != nil {
			pass.Report(*d)
		}
	})

	if len(removedFmtUsages) > 0 && n.fiximports {
		for _, pkg := range pass.Pkg.Imports() {
			if pkg.Path() == "fmt" {
				insp = pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
				nodeFilter = []ast.Node{
					(*ast.SelectorExpr)(nil),
				}
				insp.Preorder(nodeFilter, func(node ast.Node) {
					selec := node.(*ast.SelectorExpr)
					selecok, ok := selec.X.(*ast.Ident)
					if ok {
						pkgname, ok := pass.TypesInfo.ObjectOf(selecok).(*types.PkgName)
						if ok && pkgname.Name() == pkg.Name() {
							fname := pass.Fset.File(pkgname.Pos()).Name()
							removedFmtUsages[fname]--
						}
					}
				})
			} else if pkg.Path() == "errors" || pkg.Path() == "strconv" || pkg.Path() == "encoding/hex" {
				insp = pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
				nodeFilter = []ast.Node{
					(*ast.ImportSpec)(nil),
				}
				insp.Preorder(nodeFilter, func(node ast.Node) {
					gd := node.(*ast.ImportSpec)
					if gd.Path.Value == strconv.Quote(pkg.Path()) {
						fname := pass.Fset.File(gd.Pos()).Name()
						_, ok := neededPackages[fname]
						if ok {
							delete(neededPackages[fname], pkg.Path())
						}
					}
				})
			}
		}
		insp = pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
		nodeFilter = []ast.Node{
			(*ast.ImportSpec)(nil),
		}
		insp.Preorder(nodeFilter, func(node ast.Node) {
			gd := node.(*ast.ImportSpec)
			if gd.Path.Value == `"fmt"` {
				fix := ""
				fname := pass.Fset.File(gd.Pos()).Name()
				if removedFmtUsages[fname] < 0 {
					fix += `"fmt"`
					if len(neededPackages[fname]) == 0 {
						return
					}
				}
				keys := make([]string, 0, len(neededPackages[fname]))
				for k := range neededPackages[fname] {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fix = fix + "\n\t\"" + k + `"`
				}
				pass.Report(analysis.Diagnostic{
					Pos:     gd.Pos(),
					End:     gd.End(),
					Message: "Fix imports",
					SuggestedFixes: []analysis.SuggestedFix{
						{
							Message: "Fix imports",
							TextEdits: []analysis.TextEdit{{
								Pos:     gd.Pos(),
								End:     gd.End(),
								NewText: []byte(fix),
							}},
						},
					}})
			}
		})
	}

	return nil, nil
}

var errIface = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

func isBasicType(lhs types.Type, expected ...types.BasicKind) bool {
	for _, rhs := range expected {
		if types.Identical(lhs, types.Typ[rhs]) {
			return true
		}
	}
	return false
}

func formatNode(fset *token.FileSet, node ast.Node) string {
	buf := new(bytes.Buffer)
	if err := format.Node(buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}

func oneOf[T comparable](v T, expected ...T) bool {
	for _, rhs := range expected {
		if v == rhs {
			return true
		}
	}
	return false
}
