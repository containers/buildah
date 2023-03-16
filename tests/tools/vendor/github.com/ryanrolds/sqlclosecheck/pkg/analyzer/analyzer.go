package analyzer

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

const (
	rowsName    = "Rows"
	stmtName    = "Stmt"
	closeMethod = "Close"
)

type action uint8

const (
	actionUnhandled action = iota
	actionHandled
	actionReturned
	actionPassed
	actionClosed
	actionUnvaluedCall
	actionUnvaluedDefer
	actionNoOp
)

var (
	sqlPackages = []string{
		"database/sql",
		"github.com/jmoiron/sqlx",
	}
)

func NewAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "sqlclosecheck",
		Doc:  "Checks that sql.Rows and sql.Stmt are closed.",
		Run:  run,
		Requires: []*analysis.Analyzer{
			buildssa.Analyzer,
		},
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	pssa, ok := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !ok {
		return nil, nil
	}

	// Build list of types we are looking for
	targetTypes := getTargetTypes(pssa, sqlPackages)

	// If non of the types are found, skip
	if len(targetTypes) == 0 {
		return nil, nil
	}

	funcs := pssa.SrcFuncs
	for _, f := range funcs {
		for _, b := range f.Blocks {
			for i := range b.Instrs {
				// Check if instruction is call that returns a target type
				targetValues := getTargetTypesValues(b, i, targetTypes)
				if len(targetValues) == 0 {
					continue
				}

				// log.Printf("%s", f.Name())

				// For each found target check if they are closed and deferred
				for _, targetValue := range targetValues {
					refs := (*targetValue.value).Referrers()
					isClosed := checkClosed(refs, targetTypes)
					if !isClosed {
						pass.Reportf((targetValue.instr).Pos(), "Rows/Stmt was not closed")
					}

					checkDeferred(pass, refs, targetTypes, false)
				}
			}
		}
	}

	return nil, nil
}

func getTargetTypes(pssa *buildssa.SSA, targetPackages []string) []*types.Pointer {
	targets := []*types.Pointer{}

	for _, sqlPkg := range targetPackages {
		pkg := pssa.Pkg.Prog.ImportedPackage(sqlPkg)
		if pkg == nil {
			// the SQL package being checked isn't imported
			return targets
		}

		rowsType := getTypePointerFromName(pkg, rowsName)
		if rowsType != nil {
			targets = append(targets, rowsType)
		}

		stmtType := getTypePointerFromName(pkg, stmtName)
		if stmtType != nil {
			targets = append(targets, stmtType)
		}
	}

	return targets
}

func getTypePointerFromName(pkg *ssa.Package, name string) *types.Pointer {
	pkgType := pkg.Type(name)
	if pkgType == nil {
		// this package does not use Rows/Stmt
		return nil
	}

	obj := pkgType.Object()
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil
	}

	return types.NewPointer(named)
}

type targetValue struct {
	value *ssa.Value
	instr ssa.Instruction
}

func getTargetTypesValues(b *ssa.BasicBlock, i int, targetTypes []*types.Pointer) []targetValue {
	targetValues := []targetValue{}

	instr := b.Instrs[i]
	call, ok := instr.(*ssa.Call)
	if !ok {
		return targetValues
	}

	signature := call.Call.Signature()
	results := signature.Results()
	for i := 0; i < results.Len(); i++ {
		v := results.At(i)
		varType := v.Type()

		for _, targetType := range targetTypes {
			if !types.Identical(varType, targetType) {
				continue
			}

			for _, cRef := range *call.Referrers() {
				switch instr := cRef.(type) {
				case *ssa.Call:
					if len(instr.Call.Args) >= 1 && types.Identical(instr.Call.Args[0].Type(), targetType) {
						targetValues = append(targetValues, targetValue{
							value: &instr.Call.Args[0],
							instr: call,
						})
					}
				case ssa.Value:
					if types.Identical(instr.Type(), targetType) {
						targetValues = append(targetValues, targetValue{
							value: &instr,
							instr: call,
						})
					}
				}
			}
		}
	}

	return targetValues
}

func checkClosed(refs *[]ssa.Instruction, targetTypes []*types.Pointer) bool {
	numInstrs := len(*refs)
	for idx, ref := range *refs {
		// log.Printf("%T - %s", ref, ref)

		action := getAction(ref, targetTypes)
		switch action {
		case actionClosed:
			return true
		case actionPassed:
			// Passed and not used after
			if numInstrs == idx+1 {
				return true
			}
		case actionReturned:
			return true
		case actionHandled:
			return true
		default:
			// log.Printf(action)
		}
	}

	return false
}

func getAction(instr ssa.Instruction, targetTypes []*types.Pointer) action {
	switch instr := instr.(type) {
	case *ssa.Defer:
		if instr.Call.Value == nil {
			return actionUnvaluedDefer
		}

		name := instr.Call.Value.Name()
		if name == closeMethod {
			return actionClosed
		}
	case *ssa.Call:
		if instr.Call.Value == nil {
			return actionUnvaluedCall
		}

		isTarget := false
		staticCallee := instr.Call.StaticCallee()
		if staticCallee != nil {
			receiver := instr.Call.StaticCallee().Signature.Recv()
			if receiver != nil {
				isTarget = isTargetType(receiver.Type(), targetTypes)
			}
		}

		name := instr.Call.Value.Name()
		if isTarget && name == closeMethod {
			return actionClosed
		}

		if !isTarget {
			return actionPassed
		}
	case *ssa.Phi:
		return actionPassed
	case *ssa.MakeInterface:
		return actionPassed
	case *ssa.Store:
		// A Row/Stmt is stored in a struct, which may be closed later
		// by a different flow.
		if _, ok := instr.Addr.(*ssa.FieldAddr); ok {
			return actionReturned
		}

		if len(*instr.Addr.Referrers()) == 0 {
			return actionNoOp
		}

		for _, aRef := range *instr.Addr.Referrers() {
			if c, ok := aRef.(*ssa.MakeClosure); ok {
				if f, ok := c.Fn.(*ssa.Function); ok {
					for _, b := range f.Blocks {
						if checkClosed(&b.Instrs, targetTypes) {
							return actionHandled
						}
					}
				}
			}
		}
	case *ssa.UnOp:
		instrType := instr.Type()
		for _, targetType := range targetTypes {
			if types.Identical(instrType, targetType) {
				if checkClosed(instr.Referrers(), targetTypes) {
					return actionHandled
				}
			}
		}
	case *ssa.FieldAddr:
		if checkClosed(instr.Referrers(), targetTypes) {
			return actionHandled
		}
	case *ssa.Return:
		return actionReturned
	default:
		// log.Printf("%s", instr)
	}

	return actionUnhandled
}

func checkDeferred(pass *analysis.Pass, instrs *[]ssa.Instruction, targetTypes []*types.Pointer, inDefer bool) {
	for _, instr := range *instrs {
		switch instr := instr.(type) {
		case *ssa.Defer:
			if instr.Call.Value != nil && instr.Call.Value.Name() == closeMethod {
				return
			}
		case *ssa.Call:
			if instr.Call.Value != nil && instr.Call.Value.Name() == closeMethod {
				if !inDefer {
					pass.Reportf(instr.Pos(), "Close should use defer")
				}

				return
			}
		case *ssa.Store:
			if len(*instr.Addr.Referrers()) == 0 {
				return
			}

			for _, aRef := range *instr.Addr.Referrers() {
				if c, ok := aRef.(*ssa.MakeClosure); ok {
					if f, ok := c.Fn.(*ssa.Function); ok {
						for _, b := range f.Blocks {
							checkDeferred(pass, &b.Instrs, targetTypes, true)
						}
					}
				}
			}
		case *ssa.UnOp:
			instrType := instr.Type()
			for _, targetType := range targetTypes {
				if types.Identical(instrType, targetType) {
					checkDeferred(pass, instr.Referrers(), targetTypes, inDefer)
				}
			}
		case *ssa.FieldAddr:
			checkDeferred(pass, instr.Referrers(), targetTypes, inDefer)
		}
	}
}

func isTargetType(t types.Type, targetTypes []*types.Pointer) bool {
	for _, targetType := range targetTypes {
		if types.Identical(t, targetType) {
			return true
		}
	}

	return false
}
