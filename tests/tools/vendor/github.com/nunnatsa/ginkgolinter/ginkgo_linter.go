package ginkgolinter

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	gotypes "go/types"

	"github.com/go-toolsmith/astcopy"
	"golang.org/x/tools/go/analysis"

	"github.com/nunnatsa/ginkgolinter/gomegahandler"
	"github.com/nunnatsa/ginkgolinter/reverseassertion"
	"github.com/nunnatsa/ginkgolinter/types"
)

// The ginkgolinter enforces standards of using ginkgo and gomega.
//
// The current checks are:
// * enforce right length assertion - warn for assertion of len(something):
//
//   This check finds the following patterns and suggests an alternative
//   * Expect(len(something)).To(Equal(number)) ===> Expect(x).To(HaveLen(number))
//   * ExpectWithOffset(1, len(something)).ShouldNot(Equal(0)) ===> ExpectWithOffset(1, something).ShouldNot(BeEmpty())
//   * Ω(len(something)).NotTo(BeZero()) ===> Ω(something).NotTo(BeEmpty())
//   * Expect(len(something)).To(BeNumerically(">", 0)) ===> Expect(something).ToNot(BeEmpty())
//   * Expect(len(something)).To(BeNumerically(">=", 1)) ===> Expect(something).ToNot(BeEmpty())
//   * Expect(len(something)).To(BeNumerically("==", number)) ===> Expect(something).To(HaveLen(number))
//
// * enforce right nil assertion - warn for assertion of x == nil:
//   This check finds the following patterns and suggests an alternative
//   * Expect(x == nil).Should(Equal(true)) ===> Expect(x).Should(BeNil())
//   * Expect(nil == x).Should(BeTrue()) ===> Expect(x).Should(BeNil())
//   * Expect(x != nil).Should(Equal(false)) ===> Expect(x).Should(BeNil())
//   * Expect(nil == x).Should(BeFalse()) ===> Expect(x).Should(BeNil())
//   * Expect(x).Should(Equal(nil) // ===> Expect(x).Should(BeNil())

const (
	linterName                 = "ginkgo-linter"
	wrongLengthWarningTemplate = linterName + ": wrong length assertion; consider using `%s` instead"
	wrongNilWarningTemplate    = linterName + ": wrong nil assertion; consider using `%s` instead"
	wrongBoolWarningTemplate   = linterName + ": wrong boolean assertion; consider using `%s` instead"
	wrongErrWarningTemplate    = linterName + ": wrong error assertion; consider using `%s` instead"
	beEmpty                    = "BeEmpty"
	beNil                      = "BeNil"
	beTrue                     = "BeTrue"
	beFalse                    = "BeFalse"
	equal                      = "Equal"
	not                        = "Not"
	haveLen                    = "HaveLen"
	succeed                    = "Succeed"
	haveOccurred               = "HaveOccurred"
	expect                     = "Expect"
	omega                      = "Ω"
	expectWithOffset           = "ExpectWithOffset"
)

// Analyzer is the interface to go_vet
var Analyzer = NewAnalyzer()

type ginkgoLinter struct {
	suppress *types.Suppress
}

// NewAnalyzer returns an Analyzer - the package interface with nogo
func NewAnalyzer() *analysis.Analyzer {
	linter := ginkgoLinter{
		suppress: &types.Suppress{
			Len: false,
			Nil: false,
			Err: false,
		},
	}

	a := &analysis.Analyzer{
		Name: "ginkgolinter",
		Doc: `enforces standards of using ginkgo and gomega
currently, the linter searches for following:
* wrong length assertions. We want to assert the item rather than its length.
For example:
	Expect(len(x)).Should(Equal(1))
This should be replaced with:
	Expect(x)).Should(HavelLen(1))
	
* wrong nil assertions. We want to assert the item rather than a comparison result.
For example:
	Expect(x == nil).Should(BeTrue())
This should be replaced with:
	Expect(x).Should(BeNil())
	`,
		Run:              linter.run,
		RunDespiteErrors: true,
	}

	a.Flags.Init("ginkgolinter", flag.ExitOnError)
	a.Flags.Var(&linter.suppress.Len, "suppress-len-assertion", "Suppress warning for wrong length assertions")
	a.Flags.Var(&linter.suppress.Nil, "suppress-nil-assertion", "Suppress warning for wrong nil assertions")
	a.Flags.Var(&linter.suppress.Err, "suppress-err-assertion", "Suppress warning for wrong error assertions")

	return a
}

// main assertion function
func (l *ginkgoLinter) run(pass *analysis.Pass) (interface{}, error) {
	if l.suppress.AllTrue() {
		return nil, nil
	}

	for _, file := range pass.Files {
		fileSuppress := l.suppress.Clone()

		cm := ast.NewCommentMap(pass.Fset, file, file.Comments)

		fileSuppress.UpdateFromFile(cm)
		if fileSuppress.AllTrue() {
			continue
		}

		handler := gomegahandler.GetGomegaHandler(file)
		if handler == nil { // no gomega import => no use in gomega in this file; nothing to do here
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {

			stmt, ok := n.(*ast.ExprStmt)
			if !ok {
				return true
			}

			exprSuppress := fileSuppress.Clone()

			if comments, ok := cm[stmt]; ok {
				exprSuppress.UpdateFromComment(comments)
			}

			// search for function calls
			assertionExp, ok := stmt.X.(*ast.CallExpr)
			if !ok {
				return true
			}

			assertionFunc, ok := assertionExp.Fun.(*ast.SelectorExpr)
			if !ok || !isAssertionFunc(assertionFunc.Sel.Name) {
				return true
			}

			actualArg := getActualArg(assertionFunc, handler)
			if actualArg == nil {
				return true
			}

			return checkExpression(pass, exprSuppress, actualArg, assertionExp, handler)

		})
	}
	return nil, nil
}

func checkExpression(pass *analysis.Pass, exprSuppress types.Suppress, actualArg ast.Expr, assertionExp *ast.CallExpr, handler gomegahandler.Handler) bool {
	assertionExp = astcopy.CallExpr(assertionExp)
	oldExpr := goFmt(pass.Fset, assertionExp)
	if !bool(exprSuppress.Len) && isActualIsLenFunc(actualArg) {

		return checkLengthMatcher(assertionExp, pass, handler, oldExpr)
	} else {
		if nilable, compOp := getNilableFromComparison(actualArg); nilable != nil {
			if isExprError(pass, nilable) {
				if exprSuppress.Err {
					return true
				}
			} else if exprSuppress.Nil {
				return true
			}

			return checkNilMatcher(assertionExp, pass, nilable, handler, compOp == token.NEQ, oldExpr)

		} else if isExprError(pass, actualArg) {
			return bool(exprSuppress.Err) || checkNilError(pass, assertionExp, handler, actualArg, oldExpr)

		} else {
			return simplifyEqual(pass, exprSuppress, assertionExp, handler, actualArg, oldExpr)
		}
	}
}

// Check if the "actual" argument is a call to the golang built-in len() function
func isActualIsLenFunc(actualArg ast.Expr) bool {
	lenArgExp, ok := actualArg.(*ast.CallExpr)
	if !ok {
		return false
	}

	lenFunc, ok := lenArgExp.Fun.(*ast.Ident)
	return ok && lenFunc.Name == "len"
}

// Check if matcher function is in one of the patterns we want to avoid
func checkLengthMatcher(exp *ast.CallExpr, pass *analysis.Pass, handler gomegahandler.Handler, oldExp string) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	matcherFuncName, ok := handler.GetActualFuncName(matcher)
	if !ok {
		return true
	}

	switch matcherFuncName {
	case equal:
		handleEqualMatcher(matcher, pass, exp, handler, oldExp)
		return false

	case "BeZero":
		handleBeZero(pass, exp, handler, oldExp)
		return false

	case "BeNumerically":
		return handleBeNumerically(matcher, pass, exp, handler, oldExp)

	case not:
		reverseAssertionFuncLogic(exp)
		exp.Args[0] = exp.Args[0].(*ast.CallExpr).Args[0]
		return checkLengthMatcher(exp, pass, handler, oldExp)

	default:
		return true
	}
}

// Check if matcher function is in one of the patterns we want to avoid
func checkNilMatcher(exp *ast.CallExpr, pass *analysis.Pass, nilable ast.Expr, handler gomegahandler.Handler, notEqual bool, oldExp string) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	matcherFuncName, ok := handler.GetActualFuncName(matcher)
	if !ok {
		return true
	}

	switch matcherFuncName {
	case equal:
		handleEqualNilMatcher(matcher, pass, exp, handler, nilable, notEqual, oldExp)

	case beTrue:
		handleNilBeBoolMatcher(pass, exp, handler, nilable, notEqual, oldExp)

	case beFalse:
		reverseAssertionFuncLogic(exp)
		handleNilBeBoolMatcher(pass, exp, handler, nilable, notEqual, oldExp)

	case not:
		reverseAssertionFuncLogic(exp)
		exp.Args[0] = exp.Args[0].(*ast.CallExpr).Args[0]
		return checkNilMatcher(exp, pass, nilable, handler, notEqual, oldExp)

	default:
		return true
	}
	return false
}

func checkNilError(pass *analysis.Pass, assertionExp *ast.CallExpr, handler gomegahandler.Handler, actualArg ast.Expr, oldExpr string) bool {
	if len(assertionExp.Args) == 0 {
		return true
	}

	equalFuncExpr, ok := assertionExp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	funcName, ok := handler.GetActualFuncName(equalFuncExpr)
	if !ok {
		return true
	}

	switch funcName {
	case beNil: // no additional processing needed.
	case equal:

		if len(equalFuncExpr.Args) == 0 {
			return true
		}

		nilable, ok := equalFuncExpr.Args[0].(*ast.Ident)
		if !ok || nilable.Name != "nil" {
			return true
		}

	case not:
		reverseAssertionFuncLogic(assertionExp)
		assertionExp.Args[0] = assertionExp.Args[0].(*ast.CallExpr).Args[0]
		return checkNilError(pass, assertionExp, handler, actualArg, oldExpr)
	default:
		return true
	}

	var newFuncName string
	if _, ok := actualArg.(*ast.CallExpr); ok {
		newFuncName = succeed
	} else {
		reverseAssertionFuncLogic(assertionExp)
		newFuncName = haveOccurred
	}

	handler.ReplaceFunction(equalFuncExpr, ast.NewIdent(newFuncName))
	equalFuncExpr.Args = nil

	report(pass, assertionExp, wrongErrWarningTemplate, oldExpr)
	return false
}

// handle Equal(nil), Equal(true) and Equal(false)
func simplifyEqual(pass *analysis.Pass, exprSuppress types.Suppress, assertionExp *ast.CallExpr, handler gomegahandler.Handler, actualArg ast.Expr, oldExpr string) bool {
	if len(assertionExp.Args) == 0 {
		return true
	}

	equalFuncExpr, ok := assertionExp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	funcName, ok := handler.GetActualFuncName(equalFuncExpr)
	if !ok {
		return true
	}

	switch funcName {
	case equal:
		if len(equalFuncExpr.Args) == 0 {
			return true
		}

		token, ok := equalFuncExpr.Args[0].(*ast.Ident)
		if !ok {
			return true
		}

		var replacement string
		var template string
		switch token.Name {
		case "nil":
			if exprSuppress.Nil {
				return true
			}
			replacement = beNil
			template = wrongNilWarningTemplate
		case "true":
			replacement = beTrue
			template = wrongBoolWarningTemplate
		case "false":
			replacement = beFalse
			template = wrongBoolWarningTemplate
		default:
			return true
		}

		handler.ReplaceFunction(equalFuncExpr, ast.NewIdent(replacement))
		equalFuncExpr.Args = nil

		report(pass, assertionExp, template, oldExpr)

		return false

	case not:
		reverseAssertionFuncLogic(assertionExp)
		assertionExp.Args[0] = assertionExp.Args[0].(*ast.CallExpr).Args[0]
		return simplifyEqual(pass, exprSuppress, assertionExp, handler, actualArg, oldExpr)
	default:
		return true
	}
}

// checks that the function is an assertion's actual function and return the "actual" parameter. If the function
// is not assertion's actual function, return nil.
func getActualArg(assertionFunc *ast.SelectorExpr, handler gomegahandler.Handler) ast.Expr {
	actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	funcName, ok := handler.GetActualFuncName(actualExpr)
	if !ok {
		return nil
	}

	switch funcName {
	case expect, omega:
		return actualExpr.Args[0]
	case expectWithOffset:
		return actualExpr.Args[1]
	default:
		return nil
	}
}

// Replace the len function call by its parameter, to create a fix suggestion
func replaceLenActualArg(actualExpr *ast.CallExpr, handler gomegahandler.Handler) {
	name, ok := handler.GetActualFuncName(actualExpr)
	if !ok {
		return
	}

	switch name {
	case expect, omega:
		arg := actualExpr.Args[0]
		if isActualIsLenFunc(arg) {
			// replace the len function call by its parameter, to create a fix suggestion
			actualExpr.Args[0] = arg.(*ast.CallExpr).Args[0]
		}
	case expectWithOffset:
		arg := actualExpr.Args[1]
		if isActualIsLenFunc(arg) {
			// replace the len function call by its parameter, to create a fix suggestion
			actualExpr.Args[1] = arg.(*ast.CallExpr).Args[0]
		}
	}
}

// Replace the nil comparison with the compared object, to create a fix suggestion
func replaceNilActualArg(actualExpr *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr) bool {
	actualFuncName, ok := handler.GetActualFuncName(actualExpr)
	if !ok {
		return false
	}

	switch actualFuncName {
	case expect, omega:
		actualExpr.Args[0] = nilable
		return true

	case expectWithOffset:
		actualExpr.Args[1] = nilable
		return true

	default:
		return false
	}
}

// For the BeNumerically matcher, we want to avoid the assertion of length to be > 0 or >= 1, or just == number
func handleBeNumerically(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, oldExp string) bool {
	opExp, ok1 := matcher.Args[0].(*ast.BasicLit)
	valExp, ok2 := matcher.Args[1].(*ast.BasicLit)

	if ok1 && ok2 {
		op := opExp.Value
		val := valExp.Value

		if (op == `">"` && val == "0") || (op == `">="` && val == "1") {
			reverseAssertionFuncLogic(exp)
			handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(beEmpty))
			exp.Args[0].(*ast.CallExpr).Args = nil
			reportLengthAssertion(pass, exp, handler, oldExp)
			return false
		} else if op == `"=="` {
			chooseNumericMatcher(exp, handler, valExp)
			reportLengthAssertion(pass, exp, handler, oldExp)

			return false
		} else if op == `"!="` {
			reverseAssertionFuncLogic(exp)
			chooseNumericMatcher(exp, handler, valExp)
			reportLengthAssertion(pass, exp, handler, oldExp)

			return false
		}
	}
	return true
}

func chooseNumericMatcher(exp *ast.CallExpr, handler gomegahandler.Handler, valExp *ast.BasicLit) {
	caller := exp.Args[0].(*ast.CallExpr)
	if valExp.Value == "0" {
		handler.ReplaceFunction(caller, ast.NewIdent(beEmpty))
		exp.Args[0].(*ast.CallExpr).Args = nil
	} else {
		handler.ReplaceFunction(caller, ast.NewIdent(haveLen))
		exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{valExp}
	}
}

func reverseAssertionFuncLogic(exp *ast.CallExpr) {
	assertionFunc := exp.Fun.(*ast.SelectorExpr).Sel
	assertionFunc.Name = reverseassertion.ChangeAssertionLogic(assertionFunc.Name)
}

func handleEqualMatcher(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, oldExp string) {
	equalTo, ok := matcher.Args[0].(*ast.BasicLit)
	if ok {
		chooseNumericMatcher(exp, handler, equalTo)
	} else {
		handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(haveLen))
		exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{matcher.Args[0]}
	}
	reportLengthAssertion(pass, exp, handler, oldExp)
}

func handleBeZero(pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, oldExp string) {
	exp.Args[0].(*ast.CallExpr).Args = nil
	handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(beEmpty))
	reportLengthAssertion(pass, exp, handler, oldExp)
}

func handleEqualNilMatcher(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr, notEqual bool, oldExp string) {
	equalTo, ok := matcher.Args[0].(*ast.Ident)
	if !ok {
		return
	}

	if equalTo.Name == "false" {
		reverseAssertionFuncLogic(exp)
	} else if equalTo.Name != "true" {
		return
	}

	newFuncName, isItError := handleNilComparisonErr(pass, exp, nilable)

	handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(newFuncName))
	exp.Args[0].(*ast.CallExpr).Args = nil

	reportNilAssertion(pass, exp, handler, nilable, notEqual, oldExp, isItError)
}

func handleNilBeBoolMatcher(pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr, notEqual bool, oldExp string) {
	newFuncName, isItError := handleNilComparisonErr(pass, exp, nilable)
	handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(newFuncName))
	exp.Args[0].(*ast.CallExpr).Args = nil

	reportNilAssertion(pass, exp, handler, nilable, notEqual, oldExp, isItError)
}

func handleNilComparisonErr(pass *analysis.Pass, exp *ast.CallExpr, nilable ast.Expr) (string, bool) {
	newFuncName := beNil
	isItError := isExprError(pass, nilable)
	if isItError {
		if _, ok := nilable.(*ast.CallExpr); ok {
			newFuncName = succeed
		} else {
			reverseAssertionFuncLogic(exp)
			newFuncName = haveOccurred
		}
	}

	return newFuncName, isItError
}
func isAssertionFunc(name string) bool {
	switch name {
	case "To", "ToNot", "NotTo", "Should", "ShouldNot":
		return true
	}
	return false
}

func reportLengthAssertion(pass *analysis.Pass, expr *ast.CallExpr, handler gomegahandler.Handler, oldExpr string) {
	replaceLenActualArg(expr.Fun.(*ast.SelectorExpr).X.(*ast.CallExpr), handler)

	report(pass, expr, wrongLengthWarningTemplate, oldExpr)
}

func reportNilAssertion(pass *analysis.Pass, expr *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr, notEqual bool, oldExpr string, isItError bool) {
	changed := replaceNilActualArg(expr.Fun.(*ast.SelectorExpr).X.(*ast.CallExpr), handler, nilable)
	if !changed {
		return
	}

	if notEqual {
		reverseAssertionFuncLogic(expr)
	}
	template := wrongNilWarningTemplate
	if isItError {
		template = wrongErrWarningTemplate
	}

	report(pass, expr, template, oldExpr)
}

func report(pass *analysis.Pass, expr *ast.CallExpr, messageTemplate, oldExpr string) {
	newExp := goFmt(pass.Fset, expr)
	pass.Report(analysis.Diagnostic{
		Pos:     expr.Pos(),
		Message: fmt.Sprintf(messageTemplate, newExp),
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message: fmt.Sprintf("should replace %s with %s", oldExpr, newExp),
				TextEdits: []analysis.TextEdit{
					{
						Pos:     expr.Pos(),
						End:     expr.End(),
						NewText: []byte(newExp),
					},
				},
			},
		},
	})
}

func getNilableFromComparison(actualArg ast.Expr) (ast.Expr, token.Token) {
	bin, ok := actualArg.(*ast.BinaryExpr)
	if !ok {
		return nil, token.ILLEGAL
	}

	if bin.Op == token.EQL || bin.Op == token.NEQ {
		if isNil(bin.Y) {
			return bin.X, bin.Op
		} else if isNil(bin.X) {
			return bin.Y, bin.Op
		}
	}

	return nil, token.ILLEGAL
}

func isNil(expr ast.Expr) bool {
	nilObject, ok := expr.(*ast.Ident)
	return ok && nilObject.Name == "nil" && nilObject.Obj == nil
}

func goFmt(fset *token.FileSet, x ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, x)
	return b.String()
}

var errorType *gotypes.Interface

func init() {
	errorType = gotypes.Universe.Lookup("error").Type().Underlying().(*gotypes.Interface)
}

func isError(t gotypes.Type) bool {
	return gotypes.Implements(t, errorType)
}

func isExprError(pass *analysis.Pass, expr ast.Expr) bool {
	actualArgType := pass.TypesInfo.TypeOf(expr)
	switch t := actualArgType.(type) {
	case *gotypes.Named:
		if isError(actualArgType) {
			return true
		}
	case *gotypes.Tuple:
		if t.Len() > 0 {
			switch t0 := t.At(0).Type().(type) {
			case *gotypes.Named, *gotypes.Pointer:
				if isError(t0) {
					return true
				}
			}
		}
	}
	return false
}
