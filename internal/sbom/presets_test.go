package sbom

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreset(t *testing.T) {
	for presetName, expectToFind := range map[string]bool{
		"":                true,
		"syft":            true,
		"syft-cyclonedx":  true,
		"syft-spdx":       true,
		"trivy":           true,
		"trivy-cyclonedx": true,
		"trivy-spdx":      true,
		"rpc":             false,
		"justmakestuffup": false,
	} {
		desc := presetName
		if desc == "" {
			desc = "(blank)"
		}
		t.Run(desc, func(t *testing.T) {
			settings, err := Preset(presetName)
			require.NoError(t, err)
			if expectToFind {
				assert.NotNil(t, settings)
				assert.NotEmpty(t, settings.Commands)
			} else {
				assert.Nil(t, settings)
			}
		})
	}
}
