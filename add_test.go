package buildah

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.podman.io/image/v5/types"
)

func TestDirCopyContentsToKeepDirectoryNames(t *testing.T) {
	for _, tc := range []struct {
		name               string
		dirCopyContents    types.OptionalBool
		keepDirectoryNames bool
	}{
		{
			name:               "undefined defaults to not keeping directory names",
			dirCopyContents:    types.OptionalBoolUndefined,
			keepDirectoryNames: false,
		},
		{
			name:               "true means copy contents, don't keep directory names",
			dirCopyContents:    types.OptionalBoolTrue,
			keepDirectoryNames: false,
		},
		{
			name:               "false means keep directory names",
			dirCopyContents:    types.OptionalBoolFalse,
			keepDirectoryNames: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := AddAndCopyOptions{DirCopyContents: tc.dirCopyContents}
			got := options.DirCopyContents == types.OptionalBoolFalse
			assert.Equal(t, tc.keepDirectoryNames, got)
		})
	}
}
