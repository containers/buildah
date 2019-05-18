package overlay

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/storage"
	"github.com/containers/storage/pkg/idtools"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// MountTemp creates a subdir of the contentDir based on the source directory
// from the source system.  It then mounds up the source directory on to the
// generated mount point and returns the mount point to the caller.
func MountTemp(store storage.Store, containerId, source, dest string, rootUID, rootGID int) (specs.Mount, string, error) {
	mount := specs.Mount{}

	contentDir, err := store.ContainerDirectory(containerId)
	if err != nil {
		return mount, "", err
	}
	upperDir := filepath.Join(contentDir, "upper")
	workDir := filepath.Join(contentDir, "work")
	if err := idtools.MkdirAllAs(upperDir, 0700, rootUID, rootGID); err != nil {
		return mount, "", errors.Wrapf(err, "failed to create the overlay %s directory", upperDir)
	}
	if err := idtools.MkdirAllAs(workDir, 0700, rootUID, rootGID); err != nil {
		return mount, "", errors.Wrapf(err, "failed to create the overlay %s directory", workDir)
	}

	mount.Source = "overlay"
	mount.Destination = dest
	mount.Type = "overlay"
	mount.Options = strings.Split(fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s,private", source, upperDir, workDir), ",")

	return mount, contentDir, nil
}

// RemoveTemp removes temporary mountpoint and all content from its parent
// directory
func RemoveTemp(contentDir string) error {
	return os.RemoveAll(contentDir)
}
