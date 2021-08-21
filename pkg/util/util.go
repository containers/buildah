package util

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// DiscoverContainerfile tries to find a Containerfile or a Dockerfile within the provided `path`.
func DiscoverContainerfile(path string) (foundCtrFile string, err error) {
	// Test for existence of the file
	target, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "discovering Containerfile")
	}

	switch mode := target.Mode(); {
	case mode.IsDir():
		// If the path is a real directory, we assume a Containerfile or a Dockerfile within it
		ctrfile := filepath.Join(path, "Containerfile")

		// Test for existence of the Containerfile file
		file, err := os.Stat(ctrfile)
		if err != nil {
			// See if we have a Dockerfile within it
			ctrfile = filepath.Join(path, "Dockerfile")

			// Test for existence of the Dockerfile file
			file, err = os.Stat(ctrfile)
			if err != nil {
				return "", errors.Wrap(err, "cannot find Containerfile or Dockerfile in context directory")
			}
		}

		// The file exists, now verify the correct mode
		if mode := file.Mode(); mode.IsRegular() {
			foundCtrFile = ctrfile
		} else {
			return "", errors.Errorf("assumed Containerfile %q is not a file", ctrfile)
		}

	case mode.IsRegular():
		// If the context dir is a file, we assume this as Containerfile
		foundCtrFile = path
	}

	return foundCtrFile, nil
}
