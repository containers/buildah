package exhaustive

import (
	"regexp"
)

type regexpFlag struct {
	r *regexp.Regexp
}

func (v *regexpFlag) String() string {
	if v.r != nil {
		return v.r.String()
	}
	return ""
}

func (v *regexpFlag) Set(expr string) error {
	if expr == "" {
		v.r = nil
		return nil
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		return err
	}

	v.r = r
	return nil
}

func (v *regexpFlag) Get() interface{} {
	return v.r
}
