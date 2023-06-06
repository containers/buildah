# `mirror` [![Code Coverage](https://coveralls.io/repos/github/butuzov/mirror/badge.svg?branch=main)](https://coveralls.io/github/butuzov/mirror?branch=main) [![build status](https://github.com/butuzov/mirror/actions/workflows/main.yaml/badge.svg?branch=main)]()

`mirror` suggests use of alternative functions/methods in order to gain performance boosts by avoiding unnecessary `[]byte/string` conversion calls. See [MIRROR_FUNCS.md](MIRROR_FUNCS.md) list of mirror functions you can use in go's stdlib.

## 🇺🇦 PLEASE HELP ME 🇺🇦
Fundrise for scout drone **DJI Matrice 30T** for my squad (Ukrainian Forces). See more details at [butuzov/README.md](https://github.com/butuzov/butuzov/)

## Linter Use Cases

### `github.com/argoproj/argo-cd`

```go
// Before
func IsValidHostname(hostname string, fqdn bool) bool {
  if !fqdn {
    return validHostNameRegexp.Match([]byte(hostname)) || validIPv6Regexp.Match([]byte(hostname))
  } else {
    return validFQDNRegexp.Match([]byte(hostname))
  }
}

// After: With alternative method (and lost `else` case)
func IsValidHostname(hostname string, fqdn bool) bool {
  if !fqdn {
    return validHostNameRegexp.MatchString(hostname) || validIPv6Regexp.MatchString(hostname)
  }

  return validFQDNRegexp.MatchString(hostname)
}
```

## Install

```
go install github.com/butuzov/mirror/cmd/mirror@latest
```

## How to use

You run `mirror` with [`go vet`](https://pkg.go.dev/cmd/vet):

```
go vet -vettool=$(which mirror) ./...
# github.com/jcmoraisjr/haproxy-ingress/pkg/common/net/ssl
pkg/common/net/ssl/ssl.go:64:11: avoid allocations with (*os.File).WriteString
pkg/common/net/ssl/ssl.go:161:12: avoid allocations with (*os.File).WriteString
pkg/common/net/ssl/ssl.go:166:3: avoid allocations with (*os.File).WriteString
```

Can be called directly:
```
mirror ./...
# https://github.com/cosmtrek/air
/air/runner/util.go:149:6: avoid allocations with (*regexp.Regexp).MatchString
/air/runner/util.go:173:14: avoid allocations with (*os.File).WriteString
```

## Command line

- You can add checks for `_test.go` files with cli option `--with-tests`
