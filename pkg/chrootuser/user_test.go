package chrootuser

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testGroupData = `# comment
  # indented comment
wheel:*:0:root
daemon:*:1:
kmem:*:2:
`

func TestParseStripComments(t *testing.T) {
	// Test reading group file, ignoring comment lines
	rc := bufio.NewScanner(strings.NewReader(testGroupData))
	line, ok := scanWithoutComments(rc)
	assert.Equal(t, ok, true)
	assert.Equal(t, line, "wheel:*:0:root")
}

func TestParseNextGroup(t *testing.T) {
	// Test parsing group file
	rc := bufio.NewScanner(strings.NewReader(testGroupData))
	expected := []lookupGroupEntry{
		lookupGroupEntry{"wheel", 0, "root"},
		lookupGroupEntry{"daemon", 1, ""},
		lookupGroupEntry{"kmem", 2, ""},
	}
	for _, exp := range expected {
		grp := parseNextGroup(rc)
		assert.NotNil(t, grp)
		assert.Equal(t, *grp, exp)
	}
	assert.Nil(t, parseNextGroup(rc))
}
