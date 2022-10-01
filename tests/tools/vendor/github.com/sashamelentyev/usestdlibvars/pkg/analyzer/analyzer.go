package analyzer

import (
	"flag"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/sashamelentyev/usestdlibvars/pkg/analyzer/internal/mapping"
)

const (
	TimeWeekdayFlag    = "time-weekday"
	TimeMonthFlag      = "time-month"
	TimeLayoutFlag     = "time-layout"
	CryptoHashFlag     = "crypto-hash"
	HTTPMethodFlag     = "http-method"
	HTTPStatusCodeFlag = "http-status-code"
	DefaultRPCPathFlag = "default-rpc-path"
)

// New returns new usestdlibvars analyzer.
func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "usestdlibvars",
		Doc:      "A linter that detect the possibility to use variables/constants from the Go standard library.",
		Run:      run,
		Flags:    flags(),
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

func flags() flag.FlagSet {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Bool(HTTPMethodFlag, true, "suggest the use of http.MethodXX")
	flags.Bool(HTTPStatusCodeFlag, true, "suggest the use of http.StatusXX")
	flags.Bool(TimeWeekdayFlag, false, "suggest the use of time.Weekday")
	flags.Bool(TimeMonthFlag, false, "suggest the use of time.Month")
	flags.Bool(TimeLayoutFlag, false, "suggest the use of time.Layout")
	flags.Bool(CryptoHashFlag, false, "suggest the use of crypto.Hash")
	flags.Bool(DefaultRPCPathFlag, false, "suggest the use of rpc.DefaultXXPath")
	return *flags
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	filter := []ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.BasicLit)(nil),
		(*ast.CompositeLit)(nil),
		(*ast.IfStmt)(nil),
		(*ast.SwitchStmt)(nil),
	}

	insp.Preorder(filter, func(node ast.Node) {
		switch n := node.(type) {
		case *ast.CallExpr:
			selectorExpr, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return
			}

			ident, ok := selectorExpr.X.(*ast.Ident)
			if !ok {
				return
			}

			switch ident.Name {
			case "http":
				switch selectorExpr.Sel.Name {
				case "NewRequest":
					if !lookupFlag(pass, HTTPMethodFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 3, 0, token.STRING); basicLit != nil {
						checkHTTPMethod(pass, basicLit)
					}

				case "NewRequestWithContext":
					if !lookupFlag(pass, HTTPMethodFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 4, 1, token.STRING); basicLit != nil {
						checkHTTPMethod(pass, basicLit)
					}

				case "Error":
					if !lookupFlag(pass, HTTPStatusCodeFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 3, 2, token.INT); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}

				case "StatusText":
					if !lookupFlag(pass, HTTPStatusCodeFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 1, 0, token.INT); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}

				case "Redirect":
					if !lookupFlag(pass, HTTPStatusCodeFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 4, 3, token.INT); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}

				case "RedirectHandler":
					if !lookupFlag(pass, HTTPStatusCodeFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 2, 1, token.INT); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}
				}
			default:
				if selectorExpr.Sel.Name == "WriteHeader" {
					if !lookupFlag(pass, HTTPStatusCodeFlag) {
						return
					}

					if basicLit := getBasicLitFromArgs(n.Args, 1, 0, token.INT); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}
				}
			}

		case *ast.BasicLit:
			currentVal := getBasicLitValue(n)

			if lookupFlag(pass, TimeWeekdayFlag) {
				checkTimeWeekday(pass, n.Pos(), currentVal)
			}

			if lookupFlag(pass, TimeMonthFlag) {
				checkTimeMonth(pass, n.Pos(), currentVal)
			}

			if lookupFlag(pass, TimeLayoutFlag) {
				checkTimeLayout(pass, n.Pos(), currentVal)
			}

			if lookupFlag(pass, CryptoHashFlag) {
				checkCryptoHash(pass, n.Pos(), currentVal)
			}

			if lookupFlag(pass, DefaultRPCPathFlag) {
				checkDefaultRPCPath(pass, n.Pos(), currentVal)
			}

		case *ast.CompositeLit:
			selectorExpr, ok := n.Type.(*ast.SelectorExpr)
			if !ok {
				return
			}

			ident, ok := selectorExpr.X.(*ast.Ident)
			if !ok {
				return
			}

			if ident.Name == "http" {
				switch selectorExpr.Sel.Name {
				case "Request":
					if !lookupFlag(pass, HTTPMethodFlag) {
						return
					}

					if basicLit := getBasicLitFromElts(n.Elts, "Method"); basicLit != nil {
						checkHTTPMethod(pass, basicLit)
					}

				case "Response":
					if !lookupFlag(pass, HTTPStatusCodeFlag) {
						return
					}

					if basicLit := getBasicLitFromElts(n.Elts, "StatusCode"); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}
				}
			}

		case *ast.IfStmt:
			binaryExpr, ok := n.Cond.(*ast.BinaryExpr)
			if !ok {
				return
			}

			selectorExpr, ok := binaryExpr.X.(*ast.SelectorExpr)
			if !ok {
				return
			}

			basicLit, ok := binaryExpr.Y.(*ast.BasicLit)
			if !ok {
				return
			}

			switch selectorExpr.Sel.Name {
			case "StatusCode":
				if !lookupFlag(pass, HTTPStatusCodeFlag) {
					return
				}

				checkHTTPStatusCode(pass, basicLit)
			case "Method":
				if !lookupFlag(pass, HTTPMethodFlag) {
					return
				}

				checkHTTPMethod(pass, basicLit)
			}

		case *ast.SwitchStmt:
			selectorExpr, ok := n.Tag.(*ast.SelectorExpr)
			if !ok {
				return
			}

			var checkFunc func(pass *analysis.Pass, basicLit *ast.BasicLit)

			switch selectorExpr.Sel.Name {
			case "StatusCode":
				if !lookupFlag(pass, HTTPStatusCodeFlag) {
					return
				}
				checkFunc = checkHTTPStatusCode
			case "Method":
				if !lookupFlag(pass, HTTPMethodFlag) {
					return
				}
				checkFunc = checkHTTPMethod
			default:
				return
			}

			for _, stmt := range n.Body.List {
				caseClause, ok := stmt.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, expr := range caseClause.List {
					basicLit, ok := expr.(*ast.BasicLit)
					if !ok {
						continue
					}
					checkFunc(pass, basicLit)
				}
			}
		}
	})

	return nil, nil
}

func lookupFlag(pass *analysis.Pass, name string) bool {
	return pass.Analyzer.Flags.Lookup(name).Value.(flag.Getter).Get().(bool)
}

func checkHTTPMethod(pass *analysis.Pass, basicLit *ast.BasicLit) {
	currentVal := getBasicLitValue(basicLit)

	if newVal, ok := mapping.HTTPMethod[strings.ToUpper(currentVal)]; ok {
		report(pass, basicLit.Pos(), currentVal, newVal)
	}
}

func checkHTTPStatusCode(pass *analysis.Pass, basicLit *ast.BasicLit) {
	currentVal := getBasicLitValue(basicLit)

	if newVal, ok := mapping.HTTPStatusCode[currentVal]; ok {
		report(pass, basicLit.Pos(), currentVal, newVal)
	}
}

func checkTimeWeekday(pass *analysis.Pass, pos token.Pos, currentVal string) {
	if newVal, ok := mapping.TimeWeekday[currentVal]; ok {
		report(pass, pos, currentVal, newVal)
	}
}

func checkTimeMonth(pass *analysis.Pass, pos token.Pos, currentVal string) {
	if newVal, ok := mapping.TimeMonth[currentVal]; ok {
		report(pass, pos, currentVal, newVal)
	}
}

func checkTimeLayout(pass *analysis.Pass, pos token.Pos, currentVal string) {
	if newVal, ok := mapping.TimeLayout[currentVal]; ok {
		report(pass, pos, currentVal, newVal)
	}
}

func checkCryptoHash(pass *analysis.Pass, pos token.Pos, currentVal string) {
	if newVal, ok := mapping.CryptoHash[currentVal]; ok {
		report(pass, pos, currentVal, newVal)
	}
}

func checkDefaultRPCPath(pass *analysis.Pass, pos token.Pos, currentVal string) {
	if newVal, ok := mapping.DefaultRPCPath[currentVal]; ok {
		report(pass, pos, currentVal, newVal)
	}
}

// getBasicLitFromArgs gets the *ast.BasicLit of a function argument.
//
// Arguments:
//   - count - expected number of argument in function
//   - idx - index of the argument to get the *ast.BasicLit
//   - typ - argument type
func getBasicLitFromArgs(args []ast.Expr, count, idx int, typ token.Token) *ast.BasicLit {
	if len(args) != count {
		return nil
	}

	basicLit, ok := args[idx].(*ast.BasicLit)
	if !ok {
		return nil
	}

	if basicLit.Kind != typ {
		return nil
	}

	return basicLit
}

// getBasicLitFromElts gets the *ast.BasicLit of a struct elements.
//
// Arguments:
//   - key: name of key in struct
func getBasicLitFromElts(elts []ast.Expr, key string) *ast.BasicLit {
	for _, e := range elts {
		expr, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		ident, ok := expr.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name != key {
			continue
		}
		if basicLit, ok := expr.Value.(*ast.BasicLit); ok {
			return basicLit
		}
	}
	return nil
}

// getBasicLitValue returns BasicLit value as string without quotes
func getBasicLitValue(basicLit *ast.BasicLit) string {
	return strings.Trim(basicLit.Value, "\"")
}

func report(pass *analysis.Pass, pos token.Pos, currentVal, newVal string) {
	pass.Reportf(pos, `%q can be replaced by %s`, currentVal, newVal)
}
