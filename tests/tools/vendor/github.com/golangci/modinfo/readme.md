# modinfo

This module contains:
- an analyzer that returns module information.
- methods to find and read `go.mod` file

## Examples

```go
package main

import (
	"fmt"

	"github.com/golangci/modinfo"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzer = &analysis.Analyzer{
    Name: "example",
    Doc:  "Example",
    Run: func(pass *analysis.Pass) (interface{}, error) {
        file, err := modinfo.ReadModuleFileFromPass(pass)
        if err != nil {
          return nil, err
        }

        fmt.Println("go.mod", file)

        // TODO

        return nil, nil
    },
    Requires: []*analysis.Analyzer{
        inspect.Analyzer,
        modinfo.Analyzer,
    },
}
```

```go
package main

import (
	"fmt"

	"github.com/golangci/modinfo"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var Analyzer = &analysis.Analyzer{
    Name: "example",
    Doc:  "Example",
    Run: func(pass *analysis.Pass) (interface{}, error) {
        info, err := modinfo.FindModuleFromPass(pass)
        if err != nil {
          return nil, err
        }

        fmt.Println("Module", info.Dir)

        // TODO

        return nil, nil
    },
    Requires: []*analysis.Analyzer{
        inspect.Analyzer,
        modinfo.Analyzer,
    },
}
```
