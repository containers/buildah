package contextcheck

import (
	"go/ast"
	"go/types"
	"strconv"
	"strings"
	"sync"

	"github.com/gostaticanalysis/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

type Configuration struct {
	DisableFact bool
}

func NewAnalyzer(cfg Configuration) *analysis.Analyzer {
	analyzer := &analysis.Analyzer{
		Name: "contextcheck",
		Doc:  "check the function whether use a non-inherited context",
		Run:  NewRun(nil, cfg.DisableFact),
		Requires: []*analysis.Analyzer{
			buildssa.Analyzer,
		},
	}

	if !cfg.DisableFact {
		analyzer.FactTypes = append(analyzer.FactTypes, (*ctxFact)(nil))
	}

	return analyzer
}

const (
	ctxPkg  = "context"
	ctxName = "Context"

	httpPkg = "net/http"
	httpRes = "ResponseWriter"
	httpReq = "Request"
)

const (
	CtxIn      int = 1 << iota // ctx in function's param
	CtxOut                     // ctx in function's results
	CtxInField                 // ctx in function's field param
)

type entryType int

const (
	EntryNone            entryType = iota
	EntryWithCtx                   // has ctx in
	EntryWithHttpHandler           // is http handler
)

var (
	pkgFactMap = make(map[*types.Package]ctxFact)
	pkgFactMu  sync.RWMutex
)

type resInfo struct {
	Valid bool
	Funcs []string
}

type ctxFact map[string]resInfo

func (*ctxFact) String() string { return "ctxCheck" }
func (*ctxFact) AFact()         {}

type runner struct {
	pass     *analysis.Pass
	ctxTyp   *types.Named
	ctxPTyp  *types.Pointer
	skipFile map[*ast.File]bool

	httpResTyps []types.Type
	httpReqTyps []types.Type

	currentFact ctxFact
	disableFact bool
}

func NewRun(pkgs []*packages.Package, disableFact bool) func(pass *analysis.Pass) (interface{}, error) {
	m := make(map[string]bool)
	for _, pkg := range pkgs {
		m[strings.Split(pkg.PkgPath, "/")[0]] = true
	}
	return func(pass *analysis.Pass) (interface{}, error) {
		// skip different repo
		if !m[strings.Split(pass.Pkg.Path(), "/")[0]] {
			return nil, nil
		}

		r := &runner{disableFact: disableFact}
		r.run(pass)
		return nil, nil
	}
}

func (r *runner) run(pass *analysis.Pass) {
	r.pass = pass
	pssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	funcs := pssa.SrcFuncs

	// collect ctx obj
	var ok bool
	r.ctxTyp, r.ctxPTyp, ok = r.getRequiedType(pssa, ctxPkg, ctxName)
	if !ok {
		return
	}

	// collect http obj
	r.collectHttpTyps(pssa)

	r.skipFile = make(map[*ast.File]bool)
	r.currentFact = make(ctxFact)

	type entryInfo struct {
		f  *ssa.Function // entryfunc
		tp entryType     // entrytype
	}
	var tmpFuncs []entryInfo
	for _, f := range funcs {
		// skip checked function
		key := f.RelString(nil)
		if _, ok := r.currentFact[key]; ok {
			continue
		}

		if entryType := r.checkIsEntry(f); entryType == EntryNone {
			// record the result of nomal function
			checkingMap := make(map[string]bool)
			checkingMap[key] = true
			r.setFact(key, r.checkFuncWithoutCtx(f, checkingMap), f.Name())
			continue
		} else {
			tmpFuncs = append(tmpFuncs, entryInfo{f: f, tp: entryType})
		}
	}

	for _, v := range tmpFuncs {
		r.checkFuncWithCtx(v.f, v.tp)
	}

	if len(r.currentFact) > 0 {
		if r.disableFact {
			setPkgFact(pass.Pkg, r.currentFact)
		} else {
			pass.ExportPackageFact(&r.currentFact)
		}
	}
}

func (r *runner) getRequiedType(pssa *buildssa.SSA, path, name string) (obj *types.Named, pobj *types.Pointer, ok bool) {
	pkg := pssa.Pkg.Prog.ImportedPackage(path)
	if pkg == nil {
		return
	}

	objTyp := pkg.Type(name)
	if objTyp == nil {
		return
	}
	obj, ok = objTyp.Object().Type().(*types.Named)
	if !ok {
		return
	}
	pobj = types.NewPointer(obj)

	return
}

func (r *runner) collectHttpTyps(pssa *buildssa.SSA) {
	objRes, pobjRes, ok := r.getRequiedType(pssa, httpPkg, httpRes)
	if ok {
		r.httpResTyps = append(r.httpResTyps, objRes, pobjRes)
	}

	objReq, pobjReq, ok := r.getRequiedType(pssa, httpPkg, httpReq)
	if ok {
		r.httpReqTyps = append(r.httpReqTyps, objReq, pobjReq, types.NewPointer(pobjReq))
	}
}

func (r *runner) noImportedContextAndHttp(f *ssa.Function) (ret bool) {
	if !f.Pos().IsValid() {
		return false
	}

	file := analysisutil.File(r.pass, f.Pos())
	if file == nil {
		return false
	}

	if skip, has := r.skipFile[file]; has {
		return skip
	}
	defer func() {
		r.skipFile[file] = ret
	}()

	for _, impt := range file.Imports {
		path, err := strconv.Unquote(impt.Path.Value)
		if err != nil {
			continue
		}
		path = analysisutil.RemoveVendor(path)
		if path == ctxPkg || path == httpPkg {
			return false
		}
	}

	return true
}

func (r *runner) checkIsEntry(f *ssa.Function) entryType {
	if r.noImportedContextAndHttp(f) {
		return EntryNone
	}

	ctxIn, ctxOut := r.checkIsCtx(f)
	if ctxOut {
		// skip the function which generate ctx
		return EntryNone
	} else if ctxIn {
		// has ctx in, ignore *http.Request.Context()
		return EntryWithCtx
	}

	// check is `func handler(w http.ResponseWriter, r *http.Request) {}`
	if r.checkIsHttpHandler(f) {
		return EntryWithHttpHandler
	}

	return EntryNone
}

func (r *runner) checkIsCtx(f *ssa.Function) (in, out bool) {
	// check params
	tuple := f.Signature.Params()
	for i := 0; i < tuple.Len(); i++ {
		if r.isCtxType(tuple.At(i).Type()) {
			in = true
			break
		}
	}

	// check freevars
	for _, param := range f.FreeVars {
		if r.isCtxType(param.Type()) {
			in = true
			break
		}
	}

	// check results
	tuple = f.Signature.Results()
	for i := 0; i < tuple.Len(); i++ {
		if r.isCtxType(tuple.At(i).Type()) {
			out = true
			break
		}
	}
	return
}

func (r *runner) checkIsHttpHandler(f *ssa.Function) bool {
	// must has no result
	if f.Signature.Results().Len() > 0 {
		return false
	}

	// must be `func f(w http.ResponseWriter, r *http.Request) {}`
	tuple := f.Signature.Params()
	if tuple.Len() != 2 {
		return false
	}
	return r.isHttpResType(tuple.At(0).Type()) && r.isHttpReqType(tuple.At(1).Type())
}

func (r *runner) collectCtxRef(f *ssa.Function, isHttpHandler bool) (refMap map[ssa.Instruction]bool, ok bool) {
	ok = true
	refMap = make(map[ssa.Instruction]bool)
	checkedRefMap := make(map[ssa.Value]bool)
	storeInstrs := make(map[*ssa.Store]bool)
	phiInstrs := make(map[*ssa.Phi]bool)

	var checkRefs func(val ssa.Value, fromAddr bool)
	var checkInstr func(instr ssa.Instruction, fromAddr bool)

	checkRefs = func(val ssa.Value, fromAddr bool) {
		if val == nil || val.Referrers() == nil {
			return
		}

		if checkedRefMap[val] {
			return
		}
		checkedRefMap[val] = true

		for _, instr := range *val.Referrers() {
			checkInstr(instr, fromAddr)
		}
	}

	checkInstr = func(instr ssa.Instruction, fromAddr bool) {
		switch i := instr.(type) {
		case ssa.CallInstruction:
			refMap[i] = true
			tp := r.getCallInstrCtxType(i)
			if tp&CtxOut != 0 {
				// collect referrers of the results
				checkRefs(i.Value(), false)
				return
			}
		case *ssa.Store:
			if fromAddr {
				// collect all store to judge whether it's right value is valid
				storeInstrs[i] = true
			} else {
				checkRefs(i.Addr, true)
			}
		case *ssa.UnOp:
			checkRefs(i, false)
		case *ssa.MakeClosure:
			for _, param := range i.Bindings {
				if r.isCtxType(param.Type()) {
					refMap[i] = true
					break
				}
			}
		case *ssa.Extract:
			// only care about ctx
			if r.isCtxType(i.Type()) {
				checkRefs(i, false)
			}
		case *ssa.Phi:
			phiInstrs[i] = true
			checkRefs(i, false)
		case *ssa.TypeAssert:
			// ctx.(*bm.Context)
		}
	}

	if isHttpHandler {
		for _, v := range r.getHttpReqCtx(f) {
			checkRefs(v, false)
		}
	} else {
		for _, param := range f.Params {
			if r.isCtxType(param.Type()) {
				checkRefs(param, false)
			}
		}

		for _, param := range f.FreeVars {
			if r.isCtxType(param.Type()) {
				checkRefs(param, false)
			}
		}
	}

	for instr := range storeInstrs {
		if !checkedRefMap[instr.Val] {
			r.pass.Reportf(instr.Pos(), "Non-inherited new context, use function like `context.WithXXX` instead")
			ok = false
		}
	}

	for instr := range phiInstrs {
		for _, v := range instr.Edges {
			if !checkedRefMap[v] {
				r.pass.Reportf(instr.Pos(), "Non-inherited new context, use function like `context.WithXXX` instead")
				ok = false
			}
		}
	}

	return
}

func (r *runner) getHttpReqCtx(f *ssa.Function) (rets []ssa.Value) {
	checkedRefMap := make(map[ssa.Value]bool)

	var checkRefs func(val ssa.Value, fromAddr bool)
	var checkInstr func(instr ssa.Instruction, fromAddr bool)

	checkRefs = func(val ssa.Value, fromAddr bool) {
		if val == nil || val.Referrers() == nil {
			return
		}

		if checkedRefMap[val] {
			return
		}
		checkedRefMap[val] = true

		for _, instr := range *val.Referrers() {
			checkInstr(instr, fromAddr)
		}
	}

	checkInstr = func(instr ssa.Instruction, fromAddr bool) {
		switch i := instr.(type) {
		case ssa.CallInstruction:
			// r.Context() only has one recv
			if len(i.Common().Args) != 1 {
				break
			}

			// find r.Context()
			if r.getCallInstrCtxType(i)&CtxOut != CtxOut {
				break
			}

			// check is r.Context
			f := r.getFunction(instr)
			if f == nil || f.Name() != ctxName {
				break
			}
			if f.Signature.Recv() != nil {
				// collect the return of r.Context
				rets = append(rets, i.Value())
			}
		case *ssa.Store:
			if !fromAddr {
				checkRefs(i.Addr, true)
			}
		case *ssa.UnOp:
			checkRefs(i, false)
		case *ssa.Phi:
			checkRefs(i, false)
		case *ssa.MakeClosure:
		case *ssa.Extract:
			// http.Request can only be input
		}
	}

	for _, param := range f.Params {
		if r.isHttpReqType(param.Type()) {
			checkRefs(param, false)
			break
		}
	}

	return
}

func (r *runner) checkFuncWithCtx(f *ssa.Function, tp entryType) {
	isHttpHandler := tp == EntryWithHttpHandler
	refMap, ok := r.collectCtxRef(f, isHttpHandler)
	if !ok {
		return
	}

	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			tp, ok := r.getCtxType(instr)
			if !ok {
				continue
			}

			// checked in collectCtxRef, skipped
			if tp&CtxOut != 0 {
				continue
			}

			if tp&CtxIn != 0 {
				if !refMap[instr] {
					if isHttpHandler {
						r.pass.Reportf(instr.Pos(), "Non-inherited new context, use function like `context.WithXXX` or `r.Context` instead")
					} else {
						r.pass.Reportf(instr.Pos(), "Non-inherited new context, use function like `context.WithXXX` instead")
					}
				}
			}

			ff := r.getFunction(instr)
			if ff == nil {
				continue
			}

			key := ff.RelString(nil)
			res, ok := r.getValue(key, ff)
			if ok {
				if !res.Valid {
					r.pass.Reportf(instr.Pos(), "Function `%s` should pass the context parameter", strings.Join(reverse(res.Funcs), "->"))
				}
			}
		}
	}
}

func (r *runner) checkFuncWithoutCtx(f *ssa.Function, checkingMap map[string]bool) (ret bool) {
	ret = true
	orgKey := f.RelString(nil)
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			tp, ok := r.getCtxType(instr)
			if !ok {
				continue
			}

			if tp&CtxOut != 0 {
				continue
			}

			// it is considered illegal as long as ctx is in the input and not in *struct X
			if tp&CtxIn != 0 {
				if tp&CtxInField == 0 {
					ret = false
				}
			}

			ff := r.getFunction(instr)
			if ff == nil {
				continue
			}

			key := ff.RelString(nil)
			res, ok := r.getValue(key, ff)
			if ok {
				if !res.Valid {
					ret = false

					// save the call link
					r.setFact(orgKey, res.Valid, res.Funcs...)
				}
				continue
			}

			// check is thunk or bound
			if strings.HasSuffix(key, "$thunk") || strings.HasSuffix(key, "$bound") {
				continue
			}

			if entryType := r.checkIsEntry(ff); entryType == EntryNone {
				// cannot get info from fact, skip
				if ff.Blocks == nil {
					continue
				}

				// handler cycle call
				if checkingMap[key] {
					continue
				}
				checkingMap[key] = true

				valid := r.checkFuncWithoutCtx(ff, checkingMap)
				r.setFact(orgKey, valid, ff.Name())
				if !valid {
					ret = false
				}
			}
		}
	}
	return ret
}

func (r *runner) getCtxType(instr ssa.Instruction) (tp int, ok bool) {
	switch i := instr.(type) {
	case ssa.CallInstruction:
		tp = r.getCallInstrCtxType(i)
		ok = true
	case *ssa.MakeClosure:
		tp = r.getMakeClosureCtxType(i)
		ok = true
	}
	return
}

func (r *runner) getCallInstrCtxType(c ssa.CallInstruction) (tp int) {
	// check params
	for _, v := range c.Common().Args {
		if r.isCtxType(v.Type()) {
			if vv, ok := v.(*ssa.UnOp); ok {
				if _, ok := vv.X.(*ssa.FieldAddr); ok {
					tp |= CtxInField
				}
			}

			tp |= CtxIn
			break
		}
	}

	// check results
	if v := c.Value(); v != nil {
		if r.isCtxType(v.Type()) {
			tp |= CtxOut
		} else {
			tuple, ok := v.Type().(*types.Tuple)
			if !ok {
				return
			}
			for i := 0; i < tuple.Len(); i++ {
				if r.isCtxType(tuple.At(i).Type()) {
					tp |= CtxOut
					break
				}
			}
		}
	}

	return
}

func (r *runner) getMakeClosureCtxType(c *ssa.MakeClosure) (tp int) {
	for _, v := range c.Bindings {
		if r.isCtxType(v.Type()) {
			if vv, ok := v.(*ssa.UnOp); ok {
				if _, ok := vv.X.(*ssa.FieldAddr); ok {
					tp |= CtxInField
				}
			}

			tp |= CtxIn
			break
		}
	}
	return
}

func (r *runner) getFunction(instr ssa.Instruction) (f *ssa.Function) {
	switch i := instr.(type) {
	case ssa.CallInstruction:
		if i.Common().IsInvoke() {
			return
		}

		switch c := i.Common().Value.(type) {
		case *ssa.Function:
			f = c
		case *ssa.MakeClosure:
			// captured in the outer layer
		case *ssa.Builtin, *ssa.UnOp, *ssa.Lookup, *ssa.Phi:
			// skipped
		case *ssa.Extract, *ssa.Call:
			// function is a result of a call, skipped
		case *ssa.Parameter:
			// function is a param, skipped
		}
	case *ssa.MakeClosure:
		f = i.Fn.(*ssa.Function)
	}
	return
}

func (r *runner) isCtxType(tp types.Type) bool {
	return types.Identical(tp, r.ctxTyp) || types.Identical(tp, r.ctxPTyp)
}

func (r *runner) isHttpResType(tp types.Type) bool {
	for _, v := range r.httpResTyps {
		if ok := types.Identical(v, v); ok {
			return true
		}
	}
	return false
}

func (r *runner) isHttpReqType(tp types.Type) bool {
	for _, v := range r.httpReqTyps {
		if ok := types.Identical(tp, v); ok {
			return true
		}
	}
	return false
}

func (r *runner) getValue(key string, f *ssa.Function) (res resInfo, ok bool) {
	res, ok = r.currentFact[key]
	if ok {
		return
	}

	if f.Pkg == nil {
		return
	}

	var fact ctxFact
	var got bool
	if r.disableFact {
		fact, got = getPkgFact(f.Pkg.Pkg)
	} else {
		got = r.pass.ImportPackageFact(f.Pkg.Pkg, &fact)
	}
	if got {
		res, ok = fact[key]
	}
	return
}

func (r *runner) setFact(key string, valid bool, funcs ...string) {
	r.currentFact[key] = resInfo{
		Valid: valid,
		Funcs: append(r.currentFact[key].Funcs, funcs...),
	}
}

// setPkgFact save fact to mem
func setPkgFact(pkg *types.Package, fact ctxFact) {
	pkgFactMu.Lock()
	pkgFactMap[pkg] = fact
	pkgFactMu.Unlock()
}

// getPkgFact get fact from mem
func getPkgFact(pkg *types.Package) (fact ctxFact, ok bool) {
	pkgFactMu.RLock()
	fact, ok = pkgFactMap[pkg]
	pkgFactMu.RUnlock()
	return
}

func reverse(arr1 []string) (arr2 []string) {
	l := len(arr1)
	if l == 0 {
		return
	}
	arr2 = make([]string, l)
	for i := 0; i <= l/2; i++ {
		arr2[i] = arr1[l-1-i]
		arr2[l-1-i] = arr1[i]
	}
	return
}
