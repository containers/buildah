[![CircleCI](https://circleci.com/gh/sylvia7788/contextcheck.svg?style=svg)](https://circleci.com/gh/sylvia7788/contextcheck)


# contextcheck

`contextcheck` is a static analysis tool, it is used to check the function whether use a non-inherited context, which will result in a broken call link.

For example:

```go
func call1(ctx context.Context) {
    ...

    ctx = getNewCtx(ctx)
    call2(ctx) // OK

    call2(context.Background()) // Non-inherited new context, use function like `context.WithXXX` instead

    call3() // Function `call3` should pass the context parameter
    call4() // Function `call4->call3` should pass the context parameter
    ...
}

func call2(ctx context.Context) {
    ...
}

func call3() {
    ctx := context.TODO()
    call2(ctx)
}

func call4() {
    call3()
}


// if you want none-inherit ctx, use this function
func getNewCtx(ctx context.Context) (newCtx context.Context) {
    ...
    return
}

/* ---------- check net/http.HandleFunc ---------- */

func call5(ctx context.Context, w http.ResponseWriter, r *http.Request) {
}

func call6(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	call5(ctx, w, r)
	call5(context.Background(), w, r) // Non-inherited new context, use function like `context.WithXXX` or `r.Context` instead
}

func call7(in bool, w http.ResponseWriter, r *http.Request) {
	call5(r.Context(), w, r)
	call5(context.Background(), w, r)
}

func call8() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		call5(r.Context(), w, r)
		call5(context.Background(), w, r) // Non-inherited new context, use function like `context.WithXXX` or `r.Context` instead

		call6(w, r)

		// call7 should be like `func call7(ctx context.Context, in bool, w http.ResponseWriter, r *http.Request)`
		call7(true, w, r) // Function `call7` should pass the context parameter
	})
}
```

## Tips

You can break ctx inheritance by this way, eg: [issue](https://github.com/sylvia7788/contextcheck/issues/2).

```go
func call1(ctx context.Context) {
    ...

    newCtx, cancel := NoInheritCancel(ctx)
    defer cancel()

    call2(newCtx)
    ...
}

func call2(ctx context.Context) {
    ...
}

func NoInheritCancel(_ context.Context) (context.Context,context.CancelFunc) {
  return context.WithCancel(context.Background())
}
```

## Installation

You can get `contextcheck` by `go get` command.

```bash
$ go get -u github.com/sylvia7788/contextcheck
```

or build yourself.

```bash
$ make build
$ make install
```

## Usage

Invoke `contextcheck` with your package name

```bash
$ contextcheck ./...
$ # or
$ go vet -vettool=`which contextcheck` ./...
```
