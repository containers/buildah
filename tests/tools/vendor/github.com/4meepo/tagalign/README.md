# Go Tag Align

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/4meepo/tagalign?style=flat-square)
[![codecov](https://codecov.io/github/4meepo/tagalign/branch/main/graph/badge.svg?token=1R1T61UNBQ)](https://codecov.io/github/4meepo/tagalign)
[![GoDoc](https://godoc.org/github.com/4meepo/tagalign?status.svg)](https://pkg.go.dev/github.com/4meepo/tagalign)
[![Go Report Card](https://goreportcard.com/badge/github.com/4meepo/tagalign)](https://goreportcard.com/report/github.com/4meepo/tagalign)

TagAlign is used to align and sort tags in Go struct. It can make the struct more readable and easier to maintain.

For example, this struct

```go
type FooBar struct {
    Foo    int    `json:"foo" validate:"required"`
    Bar    string `json:"bar" validate:"required"`
    FooFoo int8   `json:"foo_foo" validate:"required"`
    BarBar int    `json:"bar_bar" validate:"required"`
    FooBar struct {
    Foo    int    `json:"foo" yaml:"foo" validate:"required"`
    Bar222 string `json:"bar222" validate:"required" yaml:"bar"`
    } `json:"foo_bar" validate:"required"`
    BarFoo    string `json:"bar_foo" validate:"required"`
    BarFooBar string `json:"bar_foo_bar" validate:"required"`
}
```

can be aligned to:

```go
type FooBar struct {
    Foo    int    `json:"foo"     validate:"required"`
    Bar    string `json:"bar"     validate:"required"`
    FooFoo int8   `json:"foo_foo" validate:"required"`
    BarBar int    `json:"bar_bar" validate:"required"`
    FooBar struct {
        Foo    int    `json:"foo"    yaml:"foo"          validate:"required"`
        Bar222 string `json:"bar222" validate:"required" yaml:"bar"`
    } `json:"foo_bar" validate:"required"`
    BarFoo    string `json:"bar_foo"     validate:"required"`
    BarFooBar string `json:"bar_foo_bar" validate:"required"`
}
```

In addition to alignment, it can also sort tags with fixed order. If we enable sort with fixed order `json,xml`, the following code

```go
type SortExample struct {
    Foo    int `json:"foo,omitempty" yaml:"bar" xml:"baz" binding:"required" gorm:"column:foo" zip:"foo" validate:"required"`
    Bar    int `validate:"required"  yaml:"foo" xml:"bar" binding:"required" json:"bar,omitempty" gorm:"column:bar" zip:"bar" `
    FooBar int `gorm:"column:bar" validate:"required"   xml:"bar" binding:"required" json:"bar,omitempty"  zip:"bar" yaml:"foo"`
}
```

will be sorted and aligned to:

```go
type SortExample struct {
    Foo    int `json:"foo,omitempty" xml:"baz" binding:"required" gorm:"column:foo" validate:"required" yaml:"bar" zip:"foo"`
    Bar    int `json:"bar,omitempty" xml:"bar" binding:"required" gorm:"column:bar" validate:"required" yaml:"foo" zip:"bar"`
    FooBar int `json:"bar,omitempty" xml:"bar" binding:"required" gorm:"column:bar" validate:"required" yaml:"foo" zip:"bar"`
}
```

The fixed order is `json,xml`, so the tags `json` and `xml` will be sorted and aligned first, and the rest tags will be sorted and aligned in the dictionary order.

## Install

```bash
go install github.com/4meepo/tagalign/cmd/tagalign
```

## Usage

By default tagalign will only align tags, but not sort them. But alignment and sort can work together or separately.

If you don't want to align tags, you can use `-noalign` to disable alignment.

You can use `-sort` to enable sort and `-order` to set the fixed order of tags.

```bash
# Only align tags.
tagalign -fix {package path}
# Only sort tags with fixed order.
tagalign -fix -noalign -sort -order "json,xml" {package path}
# Align and sort together.
tagalign -fix -sort -order "json,xml" {package path}
```

TODO: integrate with golangci-lint

## Reference

[Golang AST Visualizer](http://goast.yuroyoro.net/)

[Create New Golang CI Linter](https://golangci-lint.run/contributing/new-linters/)

[Autofix Example](https://github.com/golangci/golangci-lint/pull/2450/files)

[Integrating](https://disaev.me/p/writing-useful-go-analysis-linter/#integrating)
