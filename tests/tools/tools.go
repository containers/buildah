// +build tools

package tools

// Importing the packages here will allow to vendor those via
// `go mod vendor`.

import (
	_ "github.com/cpuguy83/go-md2man"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/vbatts/git-validation"
)
