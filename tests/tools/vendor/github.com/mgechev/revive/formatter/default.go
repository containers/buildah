package formatter

import (
	"fmt"

	"github.com/mgechev/revive/lint"
)

// Default is an implementation of the Formatter interface
// which formats the errors to text.
type Default struct {
	Metadata lint.FormatterMetadata
}

// Name returns the name of the formatter
func (*Default) Name() string {
	return "default"
}

// Format formats the failures gotten from the lint.
func (*Default) Format(failures <-chan lint.Failure, _ lint.Config) (string, error) {
	for failure := range failures {
		fmt.Printf("%v: %s\n", failure.Position.Start, failure.Failure)
	}
	return "", nil
}
