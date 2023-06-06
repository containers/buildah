# musttag

[![checks](https://github.com/tmzane/musttag/actions/workflows/checks.yml/badge.svg)](https://github.com/tmzane/musttag/actions/workflows/checks.yml)
[![pkg.go.dev](https://pkg.go.dev/badge/go.tmz.dev/musttag.svg)](https://pkg.go.dev/go.tmz.dev/musttag)
[![goreportcard](https://goreportcard.com/badge/go.tmz.dev/musttag)](https://goreportcard.com/report/go.tmz.dev/musttag)
[![codecov](https://codecov.io/gh/tmzane/musttag/branch/main/graph/badge.svg)](https://codecov.io/gh/tmzane/musttag)

A Go linter that enforces field tags in (un)marshaled structs

## 📌 About

`musttag` checks that exported fields of a struct passed to a `Marshal`-like function are annotated with the relevant tag:

```go
// BAD:
var user struct {
	Name string
}
data, err := json.Marshal(user)

// GOOD:
var user struct {
	Name string `json:"name"`
}
data, err := json.Marshal(user)
```

The rational from [Uber Style Guide][1]:

> The serialized form of the structure is a contract between different systems.
> Changes to the structure of the serialized form, including field names, break this contract.
> Specifying field names inside tags makes the contract explicit,
> and it guards against accidentally breaking the contract by refactoring or renaming fields.

## 🚀 Features

The following packages are supported out of the box:

* [`encoding/json`][2]
* [`encoding/xml`][3]
* [`gopkg.in/yaml.v3`][4]
* [`github.com/BurntSushi/toml`][5]
* [`github.com/mitchellh/mapstructure`][6]
* [`github.com/jmoiron/sqlx`][7]

In addition, any [custom package](#custom-packages) can be added to the list.

## 📋 Usage

`musttag` is already integrated into `golangci-lint`, and this is the recommended way to use it.

To enable the linter, add the following lines to `.golangci.yml`:

```yaml
linters:
  enable:
    - musttag
```

If you'd rather prefer to use `musttag` standalone, you can install it via `brew`...

```shell
brew install tmzane/tap/musttag
```

...or download a prebuilt binary from the [Releases][9] page.

Then run it either directly or as a `go vet` tool:

```shell
go vet -vettool=$(which musttag) ./...
```

### Custom packages

To enable reporting a custom function, you need to add its description to `.golangci.yml`.

The following is an example of adding support for the `hclsimple.DecodeFile` function from [`github.com/hashicorp/hcl`][8]:

```yaml
linters-settings:
  musttag:
    functions:
        # The full name of the function, including the package.
      - name: github.com/hashicorp/hcl/v2/hclsimple.DecodeFile
        # The struct tag whose presence should be ensured.
        tag: hcl
        # The position of the argument to check.
        arg-pos: 2
```

The same can be done via the `-fn=name:tag:arg-pos` flag when using `musttag` standalone:

```shell
musttag -fn="github.com/hashicorp/hcl/v2/hclsimple.DecodeFile:hcl:2" ./...
```

[1]: https://github.com/uber-go/guide/blob/master/style.md#use-field-tags-in-marshaled-structs
[2]: https://pkg.go.dev/encoding/json
[3]: https://pkg.go.dev/encoding/xml
[4]: https://github.com/go-yaml/yaml
[5]: https://github.com/BurntSushi/toml
[6]: https://github.com/mitchellh/mapstructure
[7]: https://github.com/jmoiron/sqlx
[8]: https://github.com/hashicorp/hcl
[9]: https://github.com/tmzane/musttag/releases
