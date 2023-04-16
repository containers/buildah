package cli

import (
	"testing"

	"github.com/containers/common/pkg/completion"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func testFlagCompletion(t *testing.T, flags pflag.FlagSet, flagCompletions completion.FlagCompletions) {
	// lookup if for each flag a flag completion function exists
	flags.VisitAll(func(f *pflag.Flag) {
		// skip hidden and deprecated flags
		if f.Hidden || len(f.Deprecated) > 0 {
			return
		}
		if _, ok := flagCompletions[f.Name]; !ok && f.Value.Type() != "bool" {
			t.Errorf("Flag %q has no shell completion function set.", f.Name)
		} else if ok && f.Value.Type() == "bool" {
			// make sure bool flags don't have a completion function
			t.Errorf(`Flag %q is a bool flag but has a shell completion function set.
	You have to remove this shell completion function.`, f.Name)
			return

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

func TestLookupEnvVarReferences(t *testing.T) {
	t.Run("EmptyInput", func(t *testing.T) {
		assert.Empty(t, LookupEnvVarReferences(nil, nil))
		assert.Empty(t, LookupEnvVarReferences([]string{}, nil))
	})

	t.Run("EmptyEnvironment", func(t *testing.T) {
		assert.Equal(t, []string{"a=b"}, LookupEnvVarReferences([]string{"a=b"}, nil))
		assert.Equal(t, []string{"a="}, LookupEnvVarReferences([]string{"a="}, nil))
		assert.Equal(t, []string{}, LookupEnvVarReferences([]string{"a"}, nil))
		assert.Equal(t, []string{}, LookupEnvVarReferences([]string{"*"}, nil))
	})

	t.Run("MissingEnvironment", func(t *testing.T) {
		assert.Equal(t,
			[]string{"a=b", "c="},
			LookupEnvVarReferences([]string{"a=b", "c="}, []string{"x=y"}))

		assert.Equal(t,
			[]string{"a=b"},
			LookupEnvVarReferences([]string{"a=b", "c"}, []string{"x=y"}))

		assert.Equal(t,
			[]string{"a=b"},
			LookupEnvVarReferences([]string{"a=b", "c*"}, []string{"x=y"}))
	})

	t.Run("MatchingEnvironment", func(t *testing.T) {
		assert.Equal(t,
			[]string{"a=b", "c="},
			LookupEnvVarReferences([]string{"a=b", "c="}, []string{"c=d", "x=y"}))

		assert.Equal(t,
			[]string{"a=b", "c=d"},
			LookupEnvVarReferences([]string{"a=b", "c"}, []string{"c=d", "x=y"}))

		assert.Equal(t,
			[]string{"a=b", "c=d"},
			LookupEnvVarReferences([]string{"a=b", "c*"}, []string{"c=d", "x=y"}))

		assert.Equal(t,
			[]string{"a=b", "c=d", "cg=i"},
			LookupEnvVarReferences([]string{"a=b", "c*"}, []string{"c=d", "x=y", "cg=i"}))
	})

	t.Run("MultipleMatches", func(t *testing.T) {
		assert.Equal(t,
			[]string{"a=b", "c=d", "cg=i", "c=d", "x=y", "cg=i", "cg=i"},
			LookupEnvVarReferences([]string{"a=b", "c*", "*", "cg*"}, []string{"c=d", "x=y", "cg=i"}))
	})
}
