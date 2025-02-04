package define

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitBuildContext(t *testing.T) {
	t.Parallel()
	// Tests with only repo
	repo, subdir, branch := parseGitBuildContext("https://github.com/containers/repo.git")
	assert.Equal(t, repo, "https://github.com/containers/repo.git")
	assert.Equal(t, subdir, "")
	assert.Equal(t, branch, "")
	// Tests url with branch
	repo, subdir, branch = parseGitBuildContext("https://github.com/containers/repo.git#main")
	assert.Equal(t, repo, "https://github.com/containers/repo.git")
	assert.Equal(t, subdir, "")
	assert.Equal(t, branch, "main")
	// Tests url with no branch and subdir
	repo, subdir, branch = parseGitBuildContext("https://github.com/containers/repo.git#:mydir")
	assert.Equal(t, repo, "https://github.com/containers/repo.git")
	assert.Equal(t, subdir, "mydir")
	assert.Equal(t, branch, "")
	// Tests url with branch and subdir
	repo, subdir, branch = parseGitBuildContext("https://github.com/containers/repo.git#main:mydir")
	assert.Equal(t, repo, "https://github.com/containers/repo.git")
	assert.Equal(t, subdir, "mydir")
	assert.Equal(t, branch, "main")
}
