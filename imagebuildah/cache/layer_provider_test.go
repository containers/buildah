package cache

import (
	"testing"
	"time"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

// Using the same expected layer key assures the symmetry between the key generated
// from the build expectations and the locally cached layers
const expected = "484b8b102aff579c376d76113f5fcf3a09a17e8236f336c9f9abfc621286568f"

func TestCalculateBuildLayerKey(t *testing.T) {
	tm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected,
		CalculateBuildLayerKey("docker",
			true, "98feaed98643f3d36c55aaba7daa1fade5871dd47e35f51d8929093e887e160f",
			"RUN echo hello",
			[]v1.History{
				{
					Created:    &tm,
					CreatedBy:  "echo hello world",
					Author:     "trusted",
					Comment:    "Test layer",
					EmptyLayer: true,
				},
			},
			[]digest.Digest{
				digest.NewDigestFromEncoded(digest.SHA256, "5d5e4b8f920278d500827612ba28787356d2f57f46b6a0f10ed6d59c7311a379"),
			}))
}
