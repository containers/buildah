package modinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
)

type ModInfo struct {
	Path      string `json:"Path"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
	Main      bool   `json:"Main"`
}

var (
	once        sync.Once
	information []ModInfo
	errInfo     error
)

var Analyzer = &analysis.Analyzer{
	Name:       "modinfo",
	Doc:        "Module information",
	URL:        "https://github.com/golangci/modinfo",
	Run:        runOnce,
	ResultType: reflect.TypeOf([]ModInfo(nil)),
}

func runOnce(pass *analysis.Pass) (any, error) {
	_, ok := os.LookupEnv("MODINFO_DEBUG_DISABLE_ONCE")
	if ok {
		return GetModuleInfo(pass)
	}

	once.Do(func() {
		information, errInfo = GetModuleInfo(pass)
	})

	return information, errInfo
}

// GetModuleInfo gets modules information.
// Always returns 1 element except for workspace (returns all the modules of the workspace).
// Based on `go list -m -json` behavior.
func GetModuleInfo(pass *analysis.Pass) ([]ModInfo, error) {
	// https://github.com/golang/go/issues/44753#issuecomment-790089020
	cmd := exec.Command("go", "list", "-m", "-json")
	for _, file := range pass.Files {
		name := pass.Fset.File(file.Pos()).Name()
		if filepath.Ext(name) != ".go" {
			continue
		}

		cmd.Dir = filepath.Dir(name)
		break
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("command go list: %w: %s", err, string(out))
	}

	var infos []ModInfo

	for dec := json.NewDecoder(bytes.NewBuffer(out)); dec.More(); {
		var v ModInfo
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("unmarshaling error: %w: %s", err, string(out))
		}

		if v.GoMod == "" {
			return nil, errors.New("working directory is not part of a module")
		}

		if !v.Main || v.Dir == "" {
			continue
		}

		infos = append(infos, v)
	}

	if len(infos) == 0 {
		return nil, errors.New("go.mod file not found")
	}

	sort.Slice(infos, func(i, j int) bool {
		return len(infos[i].Path) > len(infos[j].Path)
	})

	return infos, nil
}

// FindModuleFromPass finds the module related to the files of the pass.
func FindModuleFromPass(pass *analysis.Pass) (ModInfo, error) {
	infos, ok := pass.ResultOf[Analyzer].([]ModInfo)
	if !ok {
		return ModInfo{}, errors.New("no modinfo analyzer result")
	}

	var name string
	for _, file := range pass.Files {
		f := pass.Fset.File(file.Pos()).Name()
		if filepath.Ext(f) != ".go" {
			continue
		}

		name = f
		break
	}

	// no Go file found in analysis pass
	if name == "" {
		name, _ = os.Getwd()
	}

	for _, info := range infos {
		if !strings.HasPrefix(name, info.Dir) {
			continue
		}
		return info, nil
	}

	return ModInfo{}, errors.New("module information not found")
}

// ReadModuleFileFromPass read the `go.mod` file from the pass result.
func ReadModuleFileFromPass(pass *analysis.Pass) (*modfile.File, error) {
	info, err := FindModuleFromPass(pass)
	if err != nil {
		return nil, err
	}

	return ReadModuleFile(info)
}

// ReadModuleFile read the `go.mod` file.
func ReadModuleFile(info ModInfo) (*modfile.File, error) {
	raw, err := os.ReadFile(info.GoMod)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod file: %w", err)
	}

	return modfile.Parse("go.mod", raw, nil)
}
