package tmpdir

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"go.podman.io/common/pkg/config"
)

// GetTempDir returns the path of the preferred temporary directory on the host.
func GetTempDir() string {
	if tmpdir, ok := os.LookupEnv("TMPDIR"); ok {
		abs, err := filepath.Abs(tmpdir)
		if err == nil {
			return abs
		}
		logrus.Warnf("ignoring TMPDIR from environment, evaluating it: %v", err)
	}
	if containerConfig, err := config.Default(); err == nil {
		if tmpdir, err := containerConfig.ImageCopyTmpDir(); err == nil {
			// Only the storage-private directory belongs to Buildah.
			// TMPDIR and explicitly configured paths are managed by callers.
			if containerConfig.Engine.ImageCopyTmpDir == "storage" {
				if err := os.MkdirAll(tmpdir, 0o700); err != nil {
					logrus.Warnf("creating storage temporary directory %q: %v", tmpdir, err)
				}
			}
			return tmpdir
		}
	}
	return "/var/tmp"
}
