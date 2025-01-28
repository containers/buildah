package define

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullPolicy(t *testing.T) {
	t.Parallel()
	for name, val := range PolicyMap {
		assert.Equal(t, name, val.String())
	}
}
