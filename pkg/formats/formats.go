package formats

import (
	"github.com/containers/common/pkg/formats"
)

const (
	// JSONString const to save on duplicate variable names
	JSONString = formats.JSONString
	// IDString const to save on duplicates for Go templates
	IDString = formats.IDString
)

// Writer interface for outputs
type Writer = formats.Writer

// JSONStructArray for JSON output
type JSONStructArray = formats.JSONStructArray

// StdoutTemplateArray for Go template output
type StdoutTemplateArray = formats.StdoutTemplateArray

// JSONStruct for JSON output
type JSONStruct = formats.JSONStruct

// StdoutTemplate for Go template output
type StdoutTemplate = formats.StdoutTemplate

// YAMLStruct for YAML output
type YAMLStruct = formats.YAMLStruct
