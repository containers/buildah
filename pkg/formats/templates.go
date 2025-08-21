package formats

import (
	"text/template"

	"go.podman.io/common/pkg/formats"
)

// Parse creates a new anonymous template with the basic functions
// and parses the given format.
func Parse(format string) (*template.Template, error) {
	return formats.Parse(format)
}

// NewParse creates a new tagged template with the basic functions
// and parses the given format.
func NewParse(tag, format string) (*template.Template, error) {
	return formats.NewParse(tag, format)
}
