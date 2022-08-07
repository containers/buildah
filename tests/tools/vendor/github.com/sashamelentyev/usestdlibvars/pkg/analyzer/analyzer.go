package analyzer

import (
	"flag"
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
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
		(*ast.BasicLit)(nil),
		(*ast.CallExpr)(nil),
		(*ast.CompositeLit)(nil),
	}

	insp.Preorder(filter, func(node ast.Node) {
		switch n := node.(type) {
		case *ast.CallExpr:
			selectorExpr, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return
			}

			switch selectorExpr.Sel.Name {
			case "WriteHeader":
				if !lookupFlag(pass, HTTPStatusCodeFlag) {
					return
				}

				if basicLit := getBasicLitFromArgs(n.Args, 1, 0, token.INT); basicLit != nil {
					checkHTTPStatusCode(pass, basicLit)
				}

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
					if basicLit := getBasicLitFromElts(n.Elts, "Method"); basicLit != nil {
						checkHTTPMethod(pass, basicLit)
					}
				case "Response":
					if basicLit := getBasicLitFromElts(n.Elts, "StatusCode"); basicLit != nil {
						checkHTTPStatusCode(pass, basicLit)
					}
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

	newVal, ok := httpMethod[currentVal]
	if ok {
		report(pass, basicLit.Pos(), newVal, currentVal)
	}
}

func checkHTTPStatusCode(pass *analysis.Pass, basicLit *ast.BasicLit) {
	currentVal := getBasicLitValue(basicLit)

	newVal, ok := httpStatusCode[currentVal]
	if ok {
		report(pass, basicLit.Pos(), newVal, currentVal)
	}
}

func checkTimeWeekday(pass *analysis.Pass, pos token.Pos, currentVal string) {
	newVal, ok := timeWeekday[currentVal]
	if ok {
		report(pass, pos, newVal, currentVal)
	}
}

func checkTimeMonth(pass *analysis.Pass, pos token.Pos, currentVal string) {
	newVal, ok := timeMonth[currentVal]
	if ok {
		report(pass, pos, newVal, currentVal)
	}
}

func checkTimeLayout(pass *analysis.Pass, pos token.Pos, currentVal string) {
	newVal, ok := timeLayout[currentVal]
	if ok {
		report(pass, pos, newVal, currentVal)
	}
}

func checkCryptoHash(pass *analysis.Pass, pos token.Pos, currentVal string) {
	newVal, ok := cryptoHash[currentVal]
	if ok {
		report(pass, pos, newVal, currentVal)
	}
}

func checkDefaultRPCPath(pass *analysis.Pass, pos token.Pos, currentVal string) {
	newVal, ok := defaultRPCPath[currentVal]
	if ok {
		report(pass, pos, newVal, currentVal)
	}
}

// getBasicLitFromArgs gets the *ast.BasicLit of a function argument.
//
// - count: expected number of argument in function
// - idx: index of the argument to get the *ast.BasicLit
// - typ: argument type
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
// - key: name of key in struct
func getBasicLitFromElts(elts []ast.Expr, key string) *ast.BasicLit {
	for _, e := range elts {
		expr, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		i, ok := expr.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if i.Name != key {
			continue
		}
		basicLit, ok := expr.Value.(*ast.BasicLit)
		if !ok {
			continue
		}
		return basicLit
	}
	return nil
}

func getBasicLitValue(basicLit *ast.BasicLit) string {
	return strings.Trim(basicLit.Value, "\"")
}

func report(pass *analysis.Pass, pos token.Pos, newVal, currentVal string) {
	pass.Reportf(pos, `%q can be replaced by %s`, currentVal, newVal)
}
