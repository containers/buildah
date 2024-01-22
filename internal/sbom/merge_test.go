package sbom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeJSON(t *testing.T) {
	tmp := t.TempDir()
	map1 := map[string]any{
		"string": "yeah",
		"number": 1,
		"struct": map[string]any{
			"string": "yep",
			"number": 2,
		},
	}
	err := encodeJSON(filepath.Join(tmp, "1.json"), map1)
	require.NoError(t, err)
	st1, err := os.Stat(filepath.Join(tmp, "1.json"))
	require.NoError(t, err)
	assert.NotZero(t, st1.Size())

	map2 := struct {
		String string `json:"string"`
		Number int    `json:"number"`
		Struct struct {
			String string `json:"string"`
			Number int    `json:"number"`
		} `json:"struct"`
	}{
		String: "yeah",
		Number: 1,
		Struct: struct {
			String string `json:"string"`
			Number int    `json:"number"`
		}{
			String: "yep",
			Number: 2,
		},
	}
	err = encodeJSON(filepath.Join(tmp, "2.json"), map2)
	require.NoError(t, err)
	st2, err := os.Stat(filepath.Join(tmp, "2.json"))
	require.NoError(t, err)
	assert.NotZero(t, st2.Size())
	c1, err := os.ReadFile(filepath.Join(tmp, "1.json"))
	require.NoError(t, err)
	c2, err := os.ReadFile(filepath.Join(tmp, "2.json"))
	require.NoError(t, err)
	assert.Equalf(t, len(c2), len(c1), "length of %q is not the same as length of %q", string(c1), string(c2))
}

func TestDecodeJSON(t *testing.T) {
	tmp := t.TempDir()
	var map1, map2, map3 map[string]any
	err := os.WriteFile(filepath.Join(tmp, "1.json"), []byte(`
	{
	"string":"yeah",
	"number":1,
	"struct":{"string":"yep",
	"number":2
	}}
	`), 0o666)
	require.NoError(t, err)

	err = decodeJSON(filepath.Join(tmp, "1.json"), &map1)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmp, "2.json"), []byte(`
	{"string":"yeah",
	   "number":1,
	"struct":{"string":"yep",   "number":2}
	}
	`), 0o666)
	require.NoError(t, err)

	err = decodeJSON(filepath.Join(tmp, "2.json"), &map2)
	require.NoError(t, err)
	assert.Equal(t, map2, map1)

	err = os.WriteFile(filepath.Join(tmp, "3.txt"), []byte(`
	what a lovely, lovely day
	`), 0o666)
	require.NoError(t, err)

	err = decodeJSON(filepath.Join(tmp, "3.txt"), &map3)
	require.Error(t, err)
}

func TestGetComponentNameVersionPurl(t *testing.T) {
	input := map[string]any{
		"name":    "alice",
		"version": "1.0",
		"purl":    "purl://...",
	}
	s, purl, err := getComponentNameVersionPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice@1.0", s)
	assert.Equal(t, "purl://...", purl)

	input = map[string]any{
		"name": "alice",
		"purl": "pkg:/...",
	}
	s, purl, err = getComponentNameVersionPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice", s)
	assert.Equal(t, "pkg:/...", purl)

	input = map[string]any{
		"name":    "alice",
		"version": "2.0",
	}
	s, purl, err = getComponentNameVersionPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice@2.0", s)
	assert.Empty(t, purl)
}

func TestGetLicenseID(t *testing.T) {
	input := map[string]any{
		"licenseId": "driver",
	}
	s, err := getLicenseID(input)
	require.NoError(t, err)
	assert.Equal(t, "driver", s)
}

func TestGetPackageNameVersionInfoPurl(t *testing.T) {
	input := map[string]any{
		"name":        "alice",
		"versionInfo": "1.0",
		"externalRefs": []any{
			map[string]any{
				"referenceCategory": "PACKAGE-MANAGER",
				"referenceType":     "purl",
				"referenceLocator":  "pkg://....",
			},
		},
	}
	s, purl, err := getPackageNameVersionInfoPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice@1.0", s)
	assert.Equal(t, "pkg://....", purl)

	input = map[string]any{
		"name": "alice",
		"externalRefs": []any{
			map[string]any{
				"referenceCategory": "PACKAGE-MANAGER",
				"referenceType":     "purl",
				"referenceLocator":  "pkg:///...",
			},
		},
	}
	s, purl, err = getPackageNameVersionInfoPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice", s)
	assert.Equal(t, "pkg:///...", purl)

	input = map[string]any{
		"name": "alice",
		"externalRefs": []any{
			map[string]any{
				"referenceCategory": "NOT-THE-PACKAGE-MANAGER",
				"referenceType":     "obscure",
				"referenceLocator":  "beep:...",
			},
		},
	}
	s, purl, err = getPackageNameVersionInfoPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice", s)
	assert.Empty(t, purl)

	input = map[string]any{
		"name": "alice",
	}
	s, purl, err = getPackageNameVersionInfoPurl(input)
	require.NoError(t, err)
	assert.Equal(t, "alice", s)
	assert.Empty(t, purl)

	input = map[string]any{
		"not-name": "alice",
	}
	_, _, err = getPackageNameVersionInfoPurl(input)
	require.Error(t, err)
}

func TestMergeSlicesWithoutDuplicatesFixed(t *testing.T) {
	base := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	merge := map[string]any{
		"array": []any{
			map[string]any{"second": 2},
		},
	}
	expected := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	err := mergeSlicesWithoutDuplicates(base, merge, "array", func(record any) (string, error) {
		return "fixed", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, base)
}

func TestMergeSlicesWithoutDuplicatesDynamic(t *testing.T) {
	base := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	merge := map[string]any{
		"array": []any{
			map[string]any{"second": 2},
		},
	}
	expected := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
			map[string]any{"second": 2},
		},
	}
	err := mergeSlicesWithoutDuplicates(base, merge, "array", func(record any) (string, error) {
		if m, ok := record.(map[string]any); ok {
			for key := range m {
				return key, nil
			}
		}
		return "broken", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, base)
}

func TestMergeSlicesWithoutDuplicatesNoop(t *testing.T) {
	base := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	expected := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	err := mergeSlicesWithoutDuplicates(base, nil, "array", func(record any) (string, error) {
		if m, ok := record.(map[string]any); ok {
			for key := range m {
				return key, nil
			}
		}
		return "broken", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, base)
}

func TestMergeSlicesWithoutDuplicatesMissing(t *testing.T) {
	base := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	merge := map[string]any{
		"array": []any{
			map[string]any{"second": 2},
		},
	}
	expected := map[string]any{
		"array": []any{
			map[string]any{"first": 1},
		},
	}
	err := mergeSlicesWithoutDuplicates(base, merge, "otherarray", func(record any) (string, error) {
		if m, ok := record.(map[string]any); ok {
			for key := range m {
				return key, nil
			}
		}
		return "broken", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, base)
}
