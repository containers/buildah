# zerologlint
![build](https://github.com/ykadowak/zerologlint/actions/workflows/testing.yaml/badge.svg)

`zerologlint` is a linter for [zerolog](https://github.com/rs/zerolog) that can be run with `go vet`.
It detects the wrong usage of `zerolog` that a user forgets to dispatch `zerolog.Event` with `Send` or `Msg` function, in which case nothing will be logged. For more detailed explanations of the cases it detects, see [Example](#Example).

## Install

```bash
go install github.com/ykadowak/zerologlint/cmd/zerologlint@latest
```

## Usage
```bash
go vet -vettool=`which zerologlint` ./...
```

## Example
```go
package main

import (
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func main() {
    // 1. Basic case
    log.Info() // "must be dispatched by Msg or Send method"

    // 2. Nested case
    log.Info(). // "must be dispatched by Msg or Send method"
        Str("foo", "bar").
        Dict("dict", zerolog.Dict().
            Str("bar", "baz").
            Int("n", 1),
        )

    // 3. Reassignment case
    logger := log.Info() // "must be dispatched by Msg or Send method"
    if err != nil {
        logger = log.Error() // "must be dispatched by Msg or Send method"
    }
    logger.Str("foo", "bar")
}
```
