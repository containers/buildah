package volumes

import (
	"os"
	"testing"

	"go.podman.io/storage/pkg/reexec"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}
