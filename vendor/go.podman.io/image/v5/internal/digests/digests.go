package digests

import (
	"fmt"

	"github.com/opencontainers/go-digest"
)

// Options records users’ preferences for used digest algorithm usage.
// It is a value type and can be copied using ordinary assignment.
type Options struct {
	// FIXME: construction should ensure that if an algorithm is set, it is .Available().

	MustUse digest.Algorithm // If not "", written digests must use this algorithm.
	Prefer  digest.Algorithm // If not "", use this algorithm whenever possible.
	Default digest.Algorithm // If not "", use this algorithm if there is no reason to use anything else
}

// Situation records the context in which a digest is being chosen.
type Situation struct {
	Preexisting           digest.Digest // If not "", a pre-existing digest value (frequently one which is cheaper to use than others)
	CannotChangeAlgorithm bool          // If true, (Preexisting != "") and that value must not be replaced.
}

// Choose chooses a digest algorithm based on the options and the situation.
func (o Options) Choose(s Situation) (digest.Algorithm, error) {
	// FIXME: unit tests
	if s.CannotChangeAlgorithm && s.Preexisting == "" {
		return "", fmt.Errorf("internal error: digests.Situation.CannotChangeAlgorithm is true but Preexisting is empty")
	}

	var choice digest.Algorithm // = what we want to use
	switch {
	case o.MustUse != "":
		choice = o.MustUse
	case s.CannotChangeAlgorithm:
		choice = s.Preexisting.Algorithm()
		if !choice.Available() {
			return "", fmt.Errorf("existing digest uses unimplemented algorithm %s", choice)
		}
	case o.Prefer != "":
		choice = o.Prefer
	case s.Preexisting != "" && s.Preexisting.Algorithm().Available():
		choice = s.Preexisting.Algorithm()
	case o.Default != "":
		choice = o.Default
	default:
		choice = digest.Canonical
	}

	if s.CannotChangeAlgorithm && choice != s.Preexisting.Algorithm() {
		return "", fmt.Errorf("requested to always use digest algorithm %s but we cannot replace existing digest algorithm %s", choice, s.Preexisting.Algorithm())
	}

	return choice, nil
}
