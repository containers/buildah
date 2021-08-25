package cli

import (
	"testing"

	"github.com/containers/common/pkg/completion"
	"github.com/spf13/pflag"
)

func testFlagCompletion(t *testing.T, flags pflag.FlagSet, flagCompletions completion.FlagCompletions) {
	// lookup if for each flag a flag completion function exists
	flags.VisitAll(func(f *pflag.Flag) {
		// skip hidden, deprecated and boolean flags
		if f.Hidden || len(f.Deprecated) > 0 || f.Value.Type() == "bool" {
			return
		}
		if _, ok := flagCompletions[f.Name]; !ok {
			t.Errorf("Flag %q has no shell completion function set.", f.Name)
		}
	})

	// make sure no unnecessary flag completion functions are defined
	for name := range flagCompletions {
		if flag := flags.Lookup(name); flag == nil {
			t.Errorf("Flag %q does not exist but has a shell completion function set.", name)
		}
	}
}

func TestUserNsFlagsCompletion(t *testing.T) {
	flags := GetUserNSFlags(&UserNSResults{})
	flagCompletions := GetUserNSFlagsCompletions()
	testFlagCompletion(t, flags, flagCompletions)
}

func TestNameSpaceFlagsCompletion(t *testing.T) {
	flags := GetNameSpaceFlags(&NameSpaceResults{})
	flagCompletions := GetNameSpaceFlagsCompletions()
	testFlagCompletion(t, flags, flagCompletions)
}

func TestBudFlagsCompletion(t *testing.T) {
	flags := GetBudFlags(&BudResults{})
	flagCompletions := GetBudFlagsCompletions()
	testFlagCompletion(t, flags, flagCompletions)
}

func TestFromAndBudFlagsCompletions(t *testing.T) {
	flags, err := GetFromAndBudFlags(&FromAndBudResults{}, &UserNSResults{}, &NameSpaceResults{})
	if err != nil {
		t.Error("Could load the from and build flags.")
	}
	flagCompletions := GetFromAndBudFlagsCompletions()
	testFlagCompletion(t, flags, flagCompletions)
}
