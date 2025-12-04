package intervals

import (
	"errors"
	"go/ast"
	"go/constant"
	"go/token"
	gotypes "go/types"
	"strconv"
	"time"

	"golang.org/x/tools/go/analysis"

	"github.com/nunnatsa/ginkgolinter/internal/gomegahandler"
	"github.com/nunnatsa/ginkgolinter/internal/reports"
)

type noDurationIntervalErr struct {
	value string
}

func (err noDurationIntervalErr) Error() string {
	return "only use time.Duration for timeout and polling in Eventually() or Consistently()"
}

func CheckIntervals(pass *analysis.Pass, expr *ast.CallExpr, actualExpr *ast.CallExpr, reportBuilder *reports.Builder, handler gomegahandler.Handler, timePkg string, funcIndex int) {
	var (
		timeout time.Duration
		polling time.Duration
		err     error
	)

	timeoutOffset := funcIndex + 1
	if len(actualExpr.Args) > timeoutOffset {
		timeout, err = getDuration(pass, actualExpr.Args[timeoutOffset], timePkg)
		if err != nil {
			suggestFix := false
			if tryFixIntDuration(expr, err, handler, timePkg, timeoutOffset) {
				suggestFix = true
			}
			reportBuilder.AddIssue(suggestFix, err.Error())
		}
		pollingOffset := funcIndex + 2
		if len(actualExpr.Args) > pollingOffset {
			polling, err = getDuration(pass, actualExpr.Args[pollingOffset], timePkg)
			if err != nil {
				suggestFix := false
				if tryFixIntDuration(expr, err, handler, timePkg, pollingOffset) {
					suggestFix = true
				}
				reportBuilder.AddIssue(suggestFix, err.Error())
			}
		}
	}

	selExp := expr.Fun.(*ast.SelectorExpr)
	for {
		call, ok := selExp.X.(*ast.CallExpr)
		if !ok {
			break
		}

		fun, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		switch fun.Sel.Name {
		case "WithTimeout", "Within":
			if timeout != 0 {
				reportBuilder.AddIssue(false, "timeout defined more than once")
			} else if len(call.Args) == 1 {
				timeout, err = getDurationFromValue(pass, call.Args[0], timePkg)
				if err != nil {
					reportBuilder.AddIssue(false, err.Error())
				}
			}

		case "WithPolling", "ProbeEvery":
			if polling != 0 {
				reportBuilder.AddIssue(false, "polling defined more than once")
			} else if len(call.Args) == 1 {
				polling, err = getDurationFromValue(pass, call.Args[0], timePkg)
				if err != nil {
					reportBuilder.AddIssue(false, err.Error())
				}
			}
		}

		selExp = fun
	}

	if timeout != 0 && polling != 0 && timeout < polling {
		reportBuilder.AddIssue(false, "timeout must not be shorter than the polling interval")
	}
}

func tryFixIntDuration(expr *ast.CallExpr, err error, handler gomegahandler.Handler, timePkg string, offset int) bool {
	suggestFix := false
	var durErr noDurationIntervalErr
	if errors.As(err, &durErr) {
		if len(durErr.value) > 0 {
			actualExpr := handler.GetActualExpr(expr.Fun.(*ast.SelectorExpr))
			var newArg ast.Expr
			second := &ast.SelectorExpr{
				Sel: ast.NewIdent("Second"),
				X:   ast.NewIdent(timePkg),
			}
			if durErr.value == "1" {
				newArg = second
			} else {
				newArg = &ast.BinaryExpr{
					X:  second,
					Op: token.MUL,
					Y:  actualExpr.Args[offset],
				}
			}
			actualExpr.Args[offset] = newArg
			suggestFix = true
		}
	}

	return suggestFix
}

func getDuration(pass *analysis.Pass, interval ast.Expr, timePkg string) (time.Duration, error) {
	argType := pass.TypesInfo.TypeOf(interval)
	if durType, ok := argType.(*gotypes.Named); ok {
		if durType.Obj().Name() == "Duration" && durType.Obj().Pkg().Name() == "time" {
			return getDurationFromValue(pass, interval, timePkg)
		}
	}

	value := ""
	switch val := interval.(type) {
	case *ast.BasicLit:
		if val.Kind == token.INT {
			value = val.Value
		}
	case *ast.Ident:
		i, err := getConstDuration(pass, val, timePkg)
		if err != nil || i == 0 {
			return 0, nil
		}
		value = val.Name
	}

	return 0, noDurationIntervalErr{value: value}
}

func getDurationFromValue(pass *analysis.Pass, interval ast.Expr, timePkg string) (time.Duration, error) {
	switch dur := interval.(type) {
	case *ast.SelectorExpr:
		ident, ok := dur.X.(*ast.Ident)
		if ok {
			if ident.Name == timePkg {
				return getTimeDurationValue(dur)
			}
			return getDurationFromValue(pass, dur.Sel, timePkg)
		}
	case *ast.BinaryExpr:
		return getBinaryExprDuration(pass, dur, timePkg)

	case *ast.Ident:
		return getConstDuration(pass, dur, timePkg)
	}

	return 0, nil
}

func getConstDuration(pass *analysis.Pass, ident *ast.Ident, timePkg string) (time.Duration, error) {
	o := pass.TypesInfo.ObjectOf(ident)
	if o != nil {
		if c, ok := o.(*gotypes.Const); ok {
			if c.Val().Kind() == constant.Int {
				i, err := strconv.Atoi(c.Val().String())
				if err != nil {
					return 0, nil
				}
				return time.Duration(i), nil
			}
		}
	}

	if ident.Obj != nil && ident.Obj.Kind == ast.Con && ident.Obj.Decl != nil {
		if vals, ok := ident.Obj.Decl.(*ast.ValueSpec); ok {
			if len(vals.Values) == 1 {
				switch val := vals.Values[0].(type) {
				case *ast.BasicLit:
					if val.Kind == token.INT {
						i, err := strconv.Atoi(val.Value)
						if err != nil {
							return 0, nil
						}
						return time.Duration(i), nil
					}
					return 0, nil
				case *ast.BinaryExpr:
					return getBinaryExprDuration(pass, val, timePkg)
				}
			}
		}
	}

	return 0, nil
}

func getTimeDurationValue(dur *ast.SelectorExpr) (time.Duration, error) {
	switch dur.Sel.Name {
	case "Nanosecond":
		return time.Nanosecond, nil
	case "Microsecond":
		return time.Microsecond, nil
	case "Millisecond":
		return time.Millisecond, nil
	case "Second":
		return time.Second, nil
	case "Minute":
		return time.Minute, nil
	case "Hour":
		return time.Hour, nil
	default:
		return 0, errors.New("unknown duration value") // should never happen
	}
}

func getBinaryExprDuration(pass *analysis.Pass, expr *ast.BinaryExpr, timePkg string) (time.Duration, error) {
	x, err := getBinaryDurValue(pass, expr.X, timePkg)
	if err != nil || x == 0 {
		return 0, nil
	}
	y, err := getBinaryDurValue(pass, expr.Y, timePkg)
	if err != nil || y == 0 {
		return 0, nil
	}

	switch expr.Op {
	case token.ADD:
		return x + y, nil
	case token.SUB:
		val := x - y
		if val > 0 {
			return val, nil
		}
		return 0, nil
	case token.MUL:
		return x * y, nil
	case token.QUO:
		if y == 0 {
			return 0, nil
		}
		return x / y, nil
	case token.REM:
		if y == 0 {
			return 0, nil
		}
		return x % y, nil
	default:
		return 0, nil
	}
}

func getBinaryDurValue(pass *analysis.Pass, expr ast.Expr, timePkg string) (time.Duration, error) {
	switch x := expr.(type) {
	case *ast.SelectorExpr:
		return getDurationFromValue(pass, x, timePkg)
	case *ast.BinaryExpr:
		return getBinaryExprDuration(pass, x, timePkg)
	case *ast.BasicLit:
		if x.Kind == token.INT {
			val, err := strconv.Atoi(x.Value)
			if err != nil {
				return 0, err
			}
			return time.Duration(val), nil
		}
	case *ast.ParenExpr:
		return getBinaryDurValue(pass, x.X, timePkg)

	case *ast.Ident:
		return getConstDuration(pass, x, timePkg)
	}

	return 0, nil
}
