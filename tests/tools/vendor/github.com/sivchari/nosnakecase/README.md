# nosnakecase
nosnakecase is a linter that detects snake case of variable naming and function name.

## Instruction

```sh
go install github.com/sivchari/nosnakecase/cmd/nosnakecase@latest
```

## Usage

```go
package sandbox

// global variable name with underscore.
var v_v = 0 // want "v_v is used under score. You should use mixedCap or MixedCap."

// global constant name with underscore.
const c_c = 0 // want "c_c is used under score. You should use mixedCap or MixedCap."

// struct name with underscore.
type S_a struct { // want "S_a is used under score. You should use mixedCap or MixedCap."
	fi int
}

// non-exported struct field name with underscore.
type Sa struct {
	fi_a int // // want "fi_a is used under score. You should use mixedCap or MixedCap."
}

// function as struct field, with parameter name with underscore.
type Sb struct {
	fib func(p_a int) // want "p_a is used under score. You should use mixedCap or MixedCap."
}

// exported struct field with underscore.
type Sc struct {
	Fi_A int // want "Fi_A is used under score. You should use mixedCap or MixedCap."
}

// function as struct field, with return name with underscore.
type Sd struct {
	fib func(p int) (r_a int) // want "r_a is used under score. You should use mixedCap or MixedCap."
}

// interface name with underscore.
type I_a interface { // want "I_a is used under score. You should use mixedCap or MixedCap."
	fn(p int)
}

// interface with parameter name with underscore.
type Ia interface {
	fn(p_a int) // want "p_a is used under score. You should use mixedCap or MixedCap."
}

// interface with parameter name with underscore.
type Ib interface {
	Fn(p_a int) // want "p_a is used under score. You should use mixedCap or MixedCap."
}

// function as struct field, with return name with underscore.
type Ic interface {
	Fn_a() // want "Fn_a is used under score. You should use mixedCap or MixedCap."
}

// interface with return name with underscore.
type Id interface {
	Fn() (r_a int) // want "r_a is used under score. You should use mixedCap or MixedCap."
}

// function name with underscore.
func f_a() {} // want "f_a is used under score. You should use mixedCap or MixedCap."

// function's parameter name with underscore.
func fb(p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."

// named return with underscore.
func fc() (r_b int) { // want "r_b is used under score. You should use mixedCap or MixedCap."
	return 0
}

// local variable (short declaration) with underscore.
func fd(p int) int {
	v_b := p * 2 // want "v_b is used under score. You should use mixedCap or MixedCap."

	return v_b // want "v_b is used under score. You should use mixedCap or MixedCap."
}

// local constant with underscore.
func fe(p int) int {
	const v_b = 2 // want "v_b is used under score. You should use mixedCap or MixedCap."

	return v_b * p // want "v_b is used under score. You should use mixedCap or MixedCap."
}

// local variable with underscore.
func ff(p int) int {
	var v_b = 2 // want "v_b is used under score. You should use mixedCap or MixedCap."

	return v_b * p // want "v_b is used under score. You should use mixedCap or MixedCap."
}

// inner function, parameter name with underscore.
func fg() {
	fgl := func(p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."
	fgl(1)
}

type Foo struct{}

// method name with underscore.
func (f Foo) f_a() {} // want "f_a is used under score. You should use mixedCap or MixedCap."

// method's parameter name with underscore.
func (f Foo) fb(p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."

// named return with underscore.
func (f Foo) fc() (r_b int) { return 0 } // want "r_b is used under score. You should use mixedCap or MixedCap."

// local variable (short declaration) with underscore.
func (f Foo) fd(p int) int {
	v_b := p * 2 // want "v_b is used under score. You should use mixedCap or MixedCap."

	return v_b // want "v_b is used under score. You should use mixedCap or MixedCap."
}

// local constant with underscore.
func (f Foo) fe(p int) int {
	const v_b = 2 // want "v_b is used under score. You should use mixedCap or MixedCap."

	return v_b * p // want "v_b is used under score. You should use mixedCap or MixedCap."
}

// local variable with underscore.
func (f Foo) ff(p int) int {
	var v_b = 2 // want "v_b is used under score. You should use mixedCap or MixedCap."

	return v_b * p // want "v_b is used under score. You should use mixedCap or MixedCap."
}

func fna(a, p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."

func fna1(a string, p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."

func fnb(a, b, p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."

func fnb1(a, b string, p_a int) {} // want "p_a is used under score. You should use mixedCap or MixedCap."

func fnd(
	p_a int, // want "p_a is used under score. You should use mixedCap or MixedCap."
	p_b int, // want "p_b is used under score. You should use mixedCap or MixedCap."
	p_c int, // want "p_c is used under score. You should use mixedCap or MixedCap."
) {
}
```

```console
go vet -vettool=(which nosnakecase) ./...

# command-line-arguments
# a
./a.go:4:5: v_v is used under score. You should use mixedCap or MixedCap.
./a.go:7:7: c_c is used under score. You should use mixedCap or MixedCap.
./a.go:10:6: S_a is used under score. You should use mixedCap or MixedCap.
./a.go:16:2: fi_a is used under score. You should use mixedCap or MixedCap.
./a.go:21:11: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:26:2: Fi_A is used under score. You should use mixedCap or MixedCap.
./a.go:31:19: r_a is used under score. You should use mixedCap or MixedCap.
./a.go:35:6: I_a is used under score. You should use mixedCap or MixedCap.
./a.go:41:5: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:46:5: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:51:2: Fn_a is used under score. You should use mixedCap or MixedCap.
./a.go:56:8: r_a is used under score. You should use mixedCap or MixedCap.
./a.go:60:6: f_a is used under score. You should use mixedCap or MixedCap.
./a.go:63:9: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:66:12: r_b is used under score. You should use mixedCap or MixedCap.
./a.go:72:2: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:74:9: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:79:8: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:81:9: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:86:6: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:88:9: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:93:14: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:100:14: f_a is used under score. You should use mixedCap or MixedCap.
./a.go:103:17: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:106:20: r_b is used under score. You should use mixedCap or MixedCap.
./a.go:110:2: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:112:9: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:117:8: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:119:9: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:124:6: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:126:9: v_b is used under score. You should use mixedCap or MixedCap.
./a.go:129:13: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:131:21: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:133:16: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:135:24: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:138:2: p_a is used under score. You should use mixedCap or MixedCap.
./a.go:139:2: p_b is used under score. You should use mixedCap or MixedCap.
./a.go:140:2: p_c is used under score. You should use mixedCap or MixedCap.
```

## CI

### CircleCI

```yaml
- run:
    name: install nosnakecase
    command: go install github.com/sivchari/nosnakecase/cmd/nosnakecase@latest

- run:
    name: run nosnakecase
    command: go vet -vettool=`which nosnakecase` ./...
```

### GitHub Actions

```yaml
- name: install nosnakecase
  run: go install github.com/sivchari/nosnakecase/cmd/nosnakecase@latest

- name: run nosnakecase
  run: go vet -vettool=`which nosnakecase` ./...
```
