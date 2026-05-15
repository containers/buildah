package imagebuildah

import (
	"encoding/json"
	"strconv"
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryEntriesEqual(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		a, b  string
		equal bool
	}{
		{
			a:     `{}`,
			b:     `{}`,
			equal: true,
		},
		{
			a:     `{"created":"2020-06-17T00:22:25.47282687Z"}`,
			b:     `{"created":"2020-06-17T00:22:25.47282687Z"}`,
			equal: true,
		},
		{
			a:     `{"created":"2020-07-16T12:38:26.733333497-04:00"}`,
			b:     `{"created":"2020-07-16T12:38:26.733333497-04:00"}`,
			equal: true,
		},
		{
			a:     `{"created":"2020-07-16T12:38:26.733333497-04:00"}`,
			b:     `{"created":"2020-07-16T12:38:26.733333497Z"}`,
			equal: false,
		},
		{
			a:     `{"created":"2020-07-16T12:38:26.733333497Z"}`,
			b:     `{}`,
			equal: false,
		},
		{
			a:     `{}`,
			b:     `{"created":"2020-07-16T12:38:26.733333497Z"}`,
			equal: false,
		},
		{
			a:     `{"comment":"thing"}`,
			b:     `{"comment":"thing"}`,
			equal: true,
		},
		{
			a:     `{"comment":"thing","ignored-field-for-testing":"ignored"}`,
			b:     `{"comment":"thing"}`,
			equal: true,
		},
		{
			a:     `{"CoMmEnT":"thing"}`,
			b:     `{"comment":"thing"}`,
			equal: true,
		},
		{
			a:     `{"comment":"thing"}`,
			b:     `{"comment":"things"}`,
			equal: false,
		},
		{
			a:     `{"author":"respected"}`,
			b:     `{"author":"respected"}`,
			equal: true,
		},
		{
			a:     `{"author":"respected"}`,
			b:     `{"author":"discredited"}`,
			equal: false,
		},
		{
			a:     `{"created_by":"actions"}`,
			b:     `{"created_by":"actions"}`,
			equal: true,
		},
		{
			a:     `{"created_by":"jiggery"}`,
			b:     `{"created_by":"pokery"}`,
			equal: false,
		},
	}
	for i := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var a, b v1.History
			err := json.Unmarshal([]byte(testCases[i].a), &a)
			require.Nil(t, err, "error unmarshalling history %q: %v", testCases[i].a, err)
			err = json.Unmarshal([]byte(testCases[i].b), &b)
			require.Nil(t, err, "error unmarshalling history %q: %v", testCases[i].b, err)
			equal := historyEntriesEqual(a, b)
			assert.Equal(t, testCases[i].equal, equal, "historyEntriesEqual(%q, %q) != %v", testCases[i].a, testCases[i].b, testCases[i].equal)
		})
	}
}

func TestParseAddUnpackFlag(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		flags    []string
		expected *bool
		wantErr  bool
	}{
		{
			name:     "no flags",
			flags:    []string{},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "bare --unpack",
			flags:    []string{"--unpack"},
			expected: boolPtr(true),
			wantErr:  false,
		},
		{
			name:     "--unpack=true",
			flags:    []string{"--unpack=true"},
			expected: boolPtr(true),
			wantErr:  false,
		},
		{
			name:     "--unpack=false",
			flags:    []string{"--unpack=false"},
			expected: boolPtr(false),
			wantErr:  false,
		},
		{
			name:     "last wins: false then true",
			flags:    []string{"--unpack=false", "--unpack=true"},
			expected: boolPtr(true),
			wantErr:  false,
		},
		{
			name:     "last wins: true then false",
			flags:    []string{"--unpack=true", "--unpack=false"},
			expected: boolPtr(false),
			wantErr:  false,
		},
		{
			name:     "last wins: bare then false",
			flags:    []string{"--unpack", "--unpack=false"},
			expected: boolPtr(false),
			wantErr:  false,
		},
		{
			name:     "invalid empty value",
			flags:    []string{"--unpack="},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid value",
			flags:    []string{"--unpack=maybe"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "other flags ignored",
			flags:    []string{"--chown=1000:1000", "--chmod=755"},
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "unpack with other flags",
			flags:    []string{"--chown=1000:1000", "--unpack=true", "--chmod=755"},
			expected: boolPtr(true),
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseAddUnpackFlag(tc.flags)
			if tc.wantErr {
				assert.Error(t, err, "expected error for flags %v", tc.flags)
			} else {
				assert.NoError(t, err, "unexpected error for flags %v", tc.flags)
				if tc.expected == nil {
					assert.Nil(t, result, "expected nil result for flags %v", tc.flags)
				} else {
					require.NotNil(t, result, "expected non-nil result for flags %v", tc.flags)
					assert.Equal(t, *tc.expected, *result, "unexpected result for flags %v", tc.flags)
				}
			}
		})
	}
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
