[![Go Report Card](https://goreportcard.com/badge/github.com/nunnatsa/ginkgolinter)](https://goreportcard.com/report/github.com/nunnatsa/ginkgolinter)
[![Coverage Status](https://coveralls.io/repos/github/nunnatsa/ginkgolinter/badge.svg?branch=main)](https://coveralls.io/github/nunnatsa/ginkgolinter?branch=main)
![Build Status](https://github.com/nunnatsa/ginkgolinter/workflows/CI/badge.svg)
[![License](https://img.shields.io/github/license/nunnatsa/ginkgolinter)](/LICENSE)
[![Release](https://img.shields.io/github/release/nunnatsa/ginkgolinter.svg)](https://github.com/nunnatsa/ginkgolinter/releases/latest)
[![GitHub Releases Stats of ginkgolinter](https://img.shields.io/github/downloads/nunnatsa/ginkgolinter/total.svg?logo=github)](https://somsubhra.github.io/github-release-stats/?username=nunnatsa&repository=ginkgolinter)

# ginkgo-linter
[ginkgo](https://onsi.github.io/ginkgo/) is a popular testing framework and [gomega](https://onsi.github.io/gomega/) is its assertion package.

This is a golang linter to enforce some standards while using the ginkgo and gomega packages.

## Install the CLI
Download the right executable from the latest release, according to your OS.

Another option is to use go:
```shell
go install github.com/nunnatsa/ginkgolinter/cmd/ginkgolinter@latest
```
Then add the new executable to your PATH.

## usage
```shell
ginkgolinter [-fix] ./...
```

Use the `-fix` flag to apply the fix suggestions to the source code.

### Use ginkgolinter with golangci-lint
The ginkgolinter is now part of the popular [golangci-lint](https://golangci-lint.run/), starting from version `v0.51.1`.

It is not enabled by default, though. There are two ways to run ginkgolinter with golangci-lint:

* From command line:
  ```shell
  golangci-lint run -E ginkgolinter ./...
  ```
* From configuration:
  
  Add ginkgolinter to the enabled linters list in .golangci.reference.yml file in your project. For more details, see
  the [golangci-lint documentation](https://golangci-lint.run/usage/configuration/); e.g.
   ```yaml
   linters:
     enable:
       - ginkgolinter
   ```
## Linter Checks
The linter checks the gomega assertions in golang test code. Gomega may be used together with ginkgo tests, For example:
```go
It("should test something", func() { // It is ginkgo test case function
	Expect("abcd").To(HaveLen(4), "the string should have a length of 4") // Expect is the gomega assertion
})
```
or within a classic golang test code, like this:
```go
func TestWithGomega(t *testing.T) {
	g := NewWithT(t)
	g.Expect("abcd").To(HaveLen(4), "the string should have a length of 4")
}
```

In some cases, the gomega will be passed as a variable to function by ginkgo, for example:
```go
Eventually(func(g Gomega) error {
	g.Expect("abcd").To(HaveLen(4), "the string should have a length of 4")
	return nil
}).Should(Succeed())
```

The linter checks the `Expect`, `ExpectWithOffset` and the `Ω` "actual" functions, with the `Should`, `ShouldNot`, `To`, `ToNot` and `NotTo` assertion functions.

It also supports the embedded `Not()` matcher

### Wrong Length Assertion
The linter finds assertion of the golang built-in `len` function, with all kind of matchers, while there are already gomega matchers for these usecases; We want to assert the item, rather than its length.

There are several wrong patterns:
```go
Expect(len(x)).To(Equal(0)) // should be: Expect(x).To(BeEmpty())
Expect(len(x)).To(BeZero()) // should be: Expect(x).To(BeEmpty())
Expect(len(x)).To(BeNumeric(">", 0)) // should be: Expect(x).ToNot(BeEmpty())
Expect(len(x)).To(BeNumeric(">=", 1)) // should be: Expect(x).ToNot(BeEmpty())
Expect(len(x)).To(BeNumeric("==", 0)) // should be: Expect(x).To(BeEmpty())
Expect(len(x)).To(BeNumeric("!=", 0)) // should be: Expect(x).ToNot(BeEmpty())

Expect(len(x)).To(Equal(1)) // should be: Expect(x).To(HaveLen(1))
Expect(len(x)).To(BeNumeric("==", 2)) // should be: Expect(x).To(HaveLen(2))
Expect(len(x)).To(BeNumeric("!=", 3)) // should be: Expect(x).ToNot(HaveLen(3))
```

It also supports the embedded `Not()` matcher; e.g.

`Ω(len(x)).Should(Not(Equal(4)))` => `Ω(x).ShouldNot(HaveLen(4))`

Or even (double negative):

`Ω(len(x)).To(Not(BeNumeric(">", 0)))` => `Ω(x).To(BeEmpty())`

The output of the linter,when finding issues, looks like this:
```
./testdata/src/a/a.go:14:5: ginkgo-linter: wrong length assertion; consider using `Expect("abcd").Should(HaveLen(4))` instead
./testdata/src/a/a.go:18:5: ginkgo-linter: wrong length assertion; consider using `Expect("").Should(BeEmpty())` instead
./testdata/src/a/a.go:22:5: ginkgo-linter: wrong length assertion; consider using `Expect("").Should(BeEmpty())` instead
```
#### use the `HaveLen(0)` matcher. 
The linter will also warn about the `HaveLen(0)` matcher, and will suggest to replace it with `BeEmpty()`

### Wrong `nil` Assertion
The linter finds assertion of the comparison to nil, with all kind of matchers, instead of using the existing `BeNil()` matcher; We want to assert the item, rather than a comparison result.

There are several wrong patterns:

```go
Expect(x == nil).To(Equal(true)) // should be: Expect(x).To(BeNil())
Expect(nil == x).To(Equal(true)) // should be: Expect(x).To(BeNil())
Expect(x != nil).To(Equal(true)) // should be: Expect(x).ToNot(BeNil())
Expect(nil != nil).To(Equal(true)) // should be: Expect(x).ToNot(BeNil())

Expect(x == nil).To(BeTrue()) // should be: Expect(x).To(BeNil())
Expect(x == nil).To(BeFalse()) // should be: Expect(x).ToNot(BeNil())
```
It also supports the embedded `Not()` matcher; e.g.

`Ω(x == nil).Should(Not(BeTrue()))` => `Ω(x).ShouldNot(BeNil())`

Or even (double negative):

`Ω(x != nil).Should(Not(BeTrue()))` => `Ω(x).Should(BeNil())`

### Wrong boolean Assertion
The linter finds assertion using the `Equal` method, with the values of to `true` or `false`, instead
of using the existing `BeTrue()` or `BeFalse()` matcher.

There are several wrong patterns:

```go
Expect(x).To(Equal(true)) // should be: Expect(x).To(BeTrue())
Expect(x).To(Equal(false)) // should be: Expect(x).To(BeFalse())
```
It also supports the embedded `Not()` matcher; e.g.

`Ω(x).Should(Not(Equal(True)))` => `Ω(x).ShouldNot(BeTrue())`

### Wrong Error Assertion
The linter finds assertion of errors compared with nil, or to be equal nil, or to be nil. The linter suggests to use `Succeed` for functions or `HaveOccurred` for error values..

There are several wrong patterns:

```go
Expect(err).To(BeNil()) // should be: Expect(err).ToNot(HaveOccurred())
Expect(err == nil).To(Equal(true)) // should be: Expect(err).ToNot(HaveOccurred())
Expect(err == nil).To(BeFalse()) // should be: Expect(err).To(HaveOccurred())
Expect(err != nil).To(BeTrue()) // should be: Expect(err).To(HaveOccurred())
Expect(funcReturnsError()).To(BeNil()) // should be: Expect(funcReturnsError()).To(Succeed())

and so on
```
It also supports the embedded `Not()` matcher; e.g.

`Ω(err == nil).Should(Not(BeTrue()))` => `Ω(x).Should(HaveOccurred())`

### Wrong Comparison Assertion
The linter finds assertion of boolean comparisons, which are already supported by existing gomega matchers. 

The linter assumes that when compared something to literals or constants, these values should be used for the assertion,
and it will do its best to suggest the right assertion expression accordingly. 

There are several wrong patterns:
```go
var x = 10
var s = "abcd"

...

Expect(x == 10).Should(BeTrue()) // should be Expect(x).Should(Equal(10))
Expect(10 == x).Should(BeTrue()) // should be Expect(x).Should(Equal(10))
Expect(x != 5).Should(Equal(true)) // should be Expect(x).ShouldNot(Equal(5))
Expect(x != 0).Should(Equal(true)) // should be Expect(x).ShouldNot(BeZero())

Expect(s != "abcd").Should(BeFalse()) // should be Expect(s).Should(Equal("abcd"))
Expect("abcd" != s).Should(BeFalse()) // should be Expect(s).Should(Equal("abcd"))
```
Or non-equal comparisons:
```go
Expect(x > 10).To(BeTrue()) // ==> Expect(x).To(BeNumerically(">", 10))
Expect(x >= 15).To(BeTrue()) // ==> Expect(x).To(BeNumerically(">=", 15))
Expect(3 > y).To(BeTrue()) // ==> Expect(y).To(BeNumerically("<", 3))
// and so on ...
```

This check included a limited support in constant values. For example:
```go
const c1 = 5

...

Expect(x1 == c1).Should(BeTrue()) // ==> Expect(x1).Should(Equal(c1))
Expect(c1 == x1).Should(BeTrue()) // ==> Expect(x1).Should(Equal(c1))
```

### Using a function call in async assertion
This rule finds an actual bug in tests, where asserting a function call in an async function; e.g. `Eventually`. For
example:
```go
func slowInt(int val) int {
	time.Sleep(time.Second)
	return val
}

...

It("should test that slowInt returns 42, eventually", func() {
	Eventually(slowInt(42)).WithPolling(time.Millisecond * 100).WithTimeout(time.Second * 2).Equal(42)
})
```
The problem with the above code is that it **should** poll - call the function - until it returns 42, but what actually 
happens is that first the function is called, and we pass `42` to `Eventually` - not the function. This is not what we 
tried to do here.

The linter will suggest replacing this code by:
```go
It("should test that slowInt returns 42, eventually", func() {
	Eventually(slowInt).WithArguments(42).WithPolling(time.Millisecond * 100).WithTimeout(time.Second * 2).Equal(42)
})
```

The linter suggested replacing the function call by the function name. 

If function arguments are used, the linter will add the `WithArguments()` method to pass them.

Please notice that `WithArguments()` is only supported from gomenga v1.22.0.

When using an older version of gomega, change the code manually. For example:

```go
It("should test that slowInt returns 42, eventually", func() {
	Eventually(func() int {
		slowint(42)		
	}).WithPolling(time.Millisecond * 100).WithTimeout(time.Second * 2).Equal(42)
})
```

### Comparing a pointer with a value
The linter warns when comparing a pointer with a value.
These comparisons are always wrong and will always fail.

In case of a positive assertion (`To()` or `Should()`), the test will just fail.

But the main concern is for false positive tests, when using a negative assertion (`NotTo()`, `ToNot()`, `ShouldNot()`,
`Should(Not())` etc.); e.g.
```go
num := 5
...
pNum := &num
...
Expect(pNum).ShouldNot(Equal(6))
```
This assertion will pass, but for the wrong reasons: pNum is not equal 6, not because num == 5, but because pNum is
a pointer, while `6` is an `int`.

In the case above, the linter will suggest `Expect(pNum).ShouldNot(HaveValue(Equal(6)))`

This is also right for additional matchers: `BeTrue()` and `BeFalse()`, `BeIdenticalTo()`, `BeEquivalentTo()` 
and `BeNumerically`.

### Missing Assertion Method
The linter warns when calling an "actual" method (e.g. `Expect()`, `Eventually()` etc.), without an assertion method (e.g 
`Should()`, `NotTo()` etc.)

For example:
```go
// no assertion for the result
Eventually(doSomething).WithTimeout(time.Seconds * 5).WithPolling(time.Milliseconds * 100)
```

The linter will not suggest a fix for this warning.

This rule cannot be suppressed.

## Suppress the linter
### Suppress warning from command line
* Use the `--suppress-len-assertion=true` flag to suppress the wrong length assertion warning
* Use the `--suppress-nil-assertion=true` flag to suppress the wrong nil assertion warning
* Use the `--suppress-err-assertion=true` flag to suppress the wrong error assertion warning
* Use the `--suppress-compare-assertion=true` flag to suppress the wrong comparison assertion warning
* Use the `--suppress-async-assertion=true` flag to suppress the function call in async assertion warning
* Use the `--allow-havelen-0=true` flag to avoid warnings about `HaveLen(0)`; Note: this parameter is only supported from
  command line, and not from a comment.

### Suppress warning from the code
To suppress the wrong length assertion warning, add a comment with (only)

`ginkgo-linter:ignore-len-assert-warning`. 

To suppress the wrong nil assertion warning, add a comment with (only)

`ginkgo-linter:ignore-nil-assert-warning`. 

To suppress the wrong error assertion warning, add a comment with (only)

`ginkgo-linter:ignore-err-assert-warning`. 

To suppress the wrong comparison assertion warning, add a comment with (only)

`ginkgo-linter:ignore-compare-assert-warning`. 

To suppress the wrong async assertion warning, add a comment with (only)

`ginkgo-linter:ignore-async-assert-warning`. 

There are two options to use these comments:
1. If the comment is at the top of the file, supress the warning for the whole file; e.g.:
   ```go
   package mypackage
   
   // ginkgo-linter:ignore-len-assert-warning
   
   import (
       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
   )
   
   var _ = Describe("my test", func() {
        It("should do something", func() {
            Expect(len("abc")).Should(Equal(3)) // nothing in this file will trigger the warning
        })
   })
   ```
   
2. If the comment is before a wrong length check expression, the warning is suppressed for this expression only; for example:
   ```golang
   It("should test something", func() {
       // ginkgo-linter:ignore-nil-assert-warning
       Expect(x == nil).Should(BeTrue()) // this line will not trigger the warning
       Expect(x == nil).Should(BeTrue()) // this line will trigger the warning
   }
   ```
