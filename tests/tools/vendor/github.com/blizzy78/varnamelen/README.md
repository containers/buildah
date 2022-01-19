[![Build Status](https://api.travis-ci.com/blizzy78/varnamelen.svg?branch=master)](https://app.travis-ci.com/github/blizzy78/varnamelen) [![Coverage Status](https://coveralls.io/repos/github/blizzy78/varnamelen/badge.svg?branch=master)](https://coveralls.io/github/blizzy78/varnamelen?branch=master) [![GoDoc](https://pkg.go.dev/badge/github.com/blizzy78/varnamelen)](https://pkg.go.dev/github.com/blizzy78/varnamelen)


varnamelen
==========

A Go Analyzer checking that the length of a variable's name matches its usage scope.

A variable with a short name can be hard to use if the variable is used over a longer span of lines of code.
A longer variable name may be easier to comprehend.

The analyzer can check variable names, method receiver names, as well as named return values.

Conventional Go parameters such as `ctx context.Context` or `t *testing.T` will always be ignored.

**Example output**

```
test.go:4:2: variable name 'x' is too short for the scope of its usage (varnamelen)
        x := 123
        ^
test.go:6:2: variable name 'i' is too short for the scope of its usage (varnamelen)
        i := 10
        ^
```


Standalone Usage
----------------

The `cmd/` folder provides a standalone command line utility. You can build it like this:

```
go build -o varnamelen ./cmd/
```

**Usage**

```
varnamelen: checks that the length of a variable's name matches its scope

Usage: varnamelen [-flag] [package]

A variable with a short name can be hard to use if the variable is used
over a longer span of lines of code. A longer variable name may be easier
to comprehend.

Flags:
  -V	print version and exit
  -all
    	no effect (deprecated)
  -c int
    	display offending line with this many lines of context (default -1)
  -checkReceiver
    	check method receiver names
  -checkReturn
    	check named return values
  -cpuprofile string
    	write CPU profile to this file
  -debug string
    	debug flags, any subset of "fpstv"
  -fix
    	apply all suggested fixes
  -flags
    	print analyzer flags in JSON
  -ignoreNames value
    	comma-separated list of ignored variable names
  -json
    	emit JSON output
  -maxDistance int
    	maximum number of lines of variable usage scope considered 'short' (default 5)
  -memprofile string
    	write memory profile to this file
  -minNameLength int
    	minimum length of variable name considered 'long' (default 3)
  -source
    	no effect (deprecated)
  -tags string
    	no effect (deprecated)
  -trace string
    	write trace log to this file
  -v	no effect (deprecated)
```


License
-------

This package is licensed under the MIT license.
