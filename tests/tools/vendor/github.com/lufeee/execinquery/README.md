# execinquery - a simple query string checker in Query function
[![Go Matrix](https://github.com/lufeee/execinquery/actions/workflows/go-cross.yml/badge.svg?branch=main)](https://github.com/lufeee/execinquery/actions/workflows/go-cross.yml)
[![Go lint](https://github.com/lufeee/execinquery/actions/workflows/lint.yml/badge.svg?branch=main)](https://github.com/lufeee/execinquery/actions/workflows/lint.yml)
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat)](LICENSE)
## About

execinquery is a linter about query string checker in Query function which reads your Go src files and
warnings it finds.

## Installation

```sh
go install github.com/lufeee/execinquery/cmd/execinquery
```

## Usage
```go
package main

import (
        "database/sql"
        "log"
)

func main() {
        db, err := sql.Open("mysql", "test:test@tcp(test:3306)/test")
        if err != nil {
                log.Fatal("Database Connect Error: ", err)
        }
        defer db.Close()

        test := "a"
        _, err = db.Query("Update * FROM hoge where id = ?", test)
        if err != nil {
                log.Fatal("Query Error: ", err)
        }

}
```

```console
go vet -vettool=$(which execinquery) ./...

# command-line-arguments
./a.go:16:11: Use Exec instead of Query to execute `UPDATE` query
```

## CI

### CircleCI

```yaml
- run:
    name: install execinquery
    command: go install github.com/lufeee/execinquery

- run:
    name: run execinquery
    command: go vet -vettool=`which execinquery` ./...
```

### GitHub Actions

```yaml
- name: install execinquery
  run: go install github.com/lufeee/execinquery

- name: run execinquery
  run: go vet -vettool=`which execinquery` ./...
```

### License

MIT license.

<hr>
