# exhaustive [![Godoc][godoc-svg]][godoc]

Package exhaustive defines an analyzer that checks exhaustiveness of switch
statements of enum-like constants in Go source code.

For supported flags, the definition of enum, and the definition of
exhaustiveness used by this package, see [pkg.go.dev][godoc-doc]. For a
changelog, see [CHANGELOG][changelog] in the GitHub wiki.

The analyzer can be configured to additionally check exhaustiveness of map
literals whose key type is enum-like.

## Usage

Command line program:

```
go install github.com/nishanths/exhaustive/cmd/exhaustive@latest

exhaustive [flags] [packages]
```

Package:

```
go get github.com/nishanths/exhaustive
```

The `exhaustive.Analyzer` variable follows the guidelines of the
[`golang.org/x/tools/go/analysis`][xanalysis] package. This should make it
possible to integrate `exhaustive` in your own analysis driver program.

## Example

Given an enum:

```go
package token // import "example.org/token"

type Token int

const (
	Add Token = iota
	Subtract
	Multiply
	Quotient
	Remainder
)
```

And code that switches on the enum:

```go
package calc // import "example.org/calc"

import "example.org/token"

func f(t token.Token) {
	switch t {
	case token.Add:
	case token.Subtract:
	case token.Multiply:
	default:
	}
}

var m = map[token.Token]string{
	token.Add:      "add",
	token.Subtract: "subtract",
	token.Multiply: "multiply",
}
```

Running `exhaustive` with default options will report:

```
% exhaustive example.org/calc
calc.go:6:2: missing cases in switch of type token.Token: token.Quotient, token.Remainder
```

Specify the flag `-check=switch,map` to additionally check exhaustiveness of
map literal keys:

```
% exhaustive -check=switch,map example.org/calc
calc.go:6:2: missing cases in switch of type token.Token: token.Quotient, token.Remainder
calc.go:14:9: missing keys in map of key type token.Token: token.Quotient, token.Remainder
```

## Contributing

Issues and changes are welcome. Please discuss substantial changes
in an issue first.

[godoc]: https://pkg.go.dev/github.com/nishanths/exhaustive
[godoc-svg]: https://pkg.go.dev/badge/github.com/nishanths/exhaustive.svg
[godoc-doc]: https://pkg.go.dev/github.com/nishanths/exhaustive#section-documentation
[godoc-flags]: https://pkg.go.dev/github.com/nishanths/exhaustive#hdr-Flags
[xanalysis]: https://pkg.go.dev/golang.org/x/tools/go/analysis
[changelog]: https://github.com/nishanths/exhaustive/wiki/CHANGELOG
[issue-typeparam]: https://github.com/nishanths/exhaustive/issues/31
