# Depguard

A Go linter that checks package imports are in a list of acceptable packages.
This allows you to allow imports from a whole organization or only
allow specific packages within a repository. 

## Install

```bash
go get github.com/OpenPeeDeeP/depguard/v2
```

## Config

The Depguard binary looks for a file named `^\.?depguard\.(yaml|yml|json|toml)$` in the current
current working directory. Examples include (`.depguard.yml` or `depguard.toml`).

The following is an example configuration file.

```json
{
  "main": {
    "files": [
      "$all",
      "!$test"
    ],
    "allow": [
      "$gostd",
      "github.com/OpenPeeDeeP"
    ],
    "deny": {
      "reflect": "Who needs reflection",
    }
  },
  "tests": {
    "files": [
      "$test"
    ],
    "deny": {
      "github.com/stretchr/testify": "Please use standard library for tests"
    }
  }
}
```

- The top level is a map of lists. The key of the map is a name that shows up in 
the linter's output.
- `files` - list of file globs that will match this list of settings to compare against
- `allow` - list of allowed packages
- `deny` - map of packages that are not allowed where the value is a suggestion

Files are matched using [Globs](https://github.com/gobwas/glob). If the files 
list is empty, then all files will match that list. Prefixing a file
with an exclamation mark `!` will put that glob in a "don't match" list. A file
will match a list if it is allowed and not denied.

> Should always prefix a file glob with `**/` as files are matched against absolute paths.

Allow is a prefix of packages to allow. A dollar sign `$` can be used at the end
of a package to specify it must be exact match only.

Deny is a map where the key is a prefix of the package to deny, and the value
is a suggestion on what to use instead. A dollar sign `$` can be used at the end
of a package to specify it must be exact match only.

A Prefix List just means that a package will match a value, if the value is a 
prefix of the package. Example `github.com/OpenPeeDeeP/depguard` package will match
a value of `github.com/OpenPeeDeeP` but won't match `github.com/OpenPeeDeeP/depguard/v2`.

### Variables

There are variable replacements for each type of list (file or package). This is
to reduce repetition and tedious behaviors.

#### File Variables

> you can still use and exclamation mark `!` in front of a variable to say not to 
use it. Example `!$test` will match any file that is not a go test file.

- `$all` - matches all go files
- `$test` - matches all go test files

#### Package Variables

- `$gostd` - matches all of go's standard library (Pulled from GOROOT)

### Example Configs

Below:

- non-test go files will match `Main` and test go files will match `Test`.
- both allow all of go standard library except for the `reflect` package which will
tell the user "Please don't use reflect package".
- go test files are also allowed to use https://github.com/stretchr/testify package
and any sub-package of it.

```yaml
Main:
  files:
  - $all
  - "!$test"
  allow:
  - $gostd
  deny:
    reflect: Please don't use reflect package
Test:
  files:
  - $test
  allow:
  - $gostd
  - github.com/stretchr/testify
  deny:
    reflect: Please don't use reflect package
```

Below:

- All go files will match `Main`
- Go files in internal will match both `Main` and `Internal`

```yaml
Main:
  files:
  - $all
Internal:
  files:
  - "**/internal/**/*.go"
```

Below:

- All packages are allowed except for `github.com/OpenPeeDeeP/depguard`. Though
`github.com/OpenPeeDeeP/depguard/v2` and `github.com/OpenPeeDeeP/depguard/somepackage`
would be allowed.

```yaml
Main:
  deny:
  - github.com/OpenPeeDeeP/depguard$
```

## Golangci-lint

This linter was built with
[Golangci-lint](https://github.com/golangci/golangci-lint) in mind. It is compatible
and read their docs to see how to implement all their linters, including this one.
