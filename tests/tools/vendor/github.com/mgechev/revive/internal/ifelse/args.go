package ifelse

// PreserveScope is a configuration argument that prevents suggestions
// that would enlarge variable scope
const PreserveScope = "preserveScope"

// Args contains arguments common to the early-return, indent-error-flow
// and superfluous-else rules (currently just preserveScope)
type Args struct {
	PreserveScope bool
}
