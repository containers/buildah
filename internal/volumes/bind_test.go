package volumes

import (
	"os"
	"testing"

	"github.com/containers/storage/pkg/reexec"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}
