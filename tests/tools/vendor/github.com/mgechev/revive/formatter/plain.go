package formatter

import (
	"bytes"
	"fmt"

	"github.com/mgechev/revive/lint"
)

// Plain is an implementation of the Formatter interface
// which formats the errors to JSON.
type Plain struct {
	Metadata lint.FormatterMetadata
}

// Name returns the name of the formatter
func (*Plain) Name() string {
	return "plain"
}

// Format formats the failures gotten from the lint.
func (*Plain) Format(failures <-chan lint.Failure, _ lint.Config) (string, error) {
	var buf bytes.Buffer
	for failure := range failures {
		fmt.Fprintf(&buf, "%v: %s %s\n", failure.Position.Start, failure.Failure, "https://revive.run/r#"+failure.RuleName)
	}
	return buf.String(), nil
}
