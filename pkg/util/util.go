package util //nolint:revive,nolintlint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/buildah/define"
	"github.com/containers/buildah/pkg/parse"
)

// Mirrors path to a tmpfile if path points to a
// file descriptor instead of actual file on filesystem
// reason: operations with file descriptors are can lead
// to edge cases where content on FD is not in a consumable
// state after first consumption.
// returns path as string and bool to confirm if temp file
// was created and needs to be cleaned up.
func MirrorToTempFileIfPathIsDescriptor(file string) (string, bool) {
	// one use-case is discussed here
	// https://github.com/containers/buildah/issues/3070
	if !strings.HasPrefix(file, "/dev/fd/") {
		return file, false
	}
	b, err := os.ReadFile(file)
	if err != nil {
		// if anything goes wrong return original path
		return file, false
	}
	tmpfile, err := os.CreateTemp(parse.GetTempDir(), "buildah-temp-file")
	if err != nil {
		return file, false
	}
	defer tmpfile.Close()
	if _, err := tmpfile.Write(b); err != nil {
		// if anything goes wrong return original path
		return file, false
	}

	return tmpfile.Name(), true
}

// filename search order
var containerFileNames = []string{
	"Containerfile",
	"Dockerfile",
}

func discoverTemplateContainerfile(path string, suffix string) (string, error) {
	var filePath string
	var file os.FileInfo
	var err error

	for _, name := range containerFileNames {
		if suffix != "" {
			filePath = filepath.Join(path, name+"."+suffix)
		} else {
			filePath = filepath.Join(path, name)
		}

		file, err = os.Stat(filePath)
		if err != nil {
			continue
		}

		// The file exists, now verify the correct mode
		if mode := file.Mode(); mode.IsRegular() {
			return filePath, nil
		}
	}

	return "", errors.New("cannot find Containerfile")
}

func discoverContainerfile(path string) (string, error) {
	return discoverTemplateContainerfile(path, "")
}

// DiscoverContainerfileEx searches for a Containerfile or Dockerfile at the given path.
// When path is a directory, it searches for regular container definition files
// ("Containerfile", "Dockerfile") and optionally for template files
// ("Containerfile.suffix", "Dockerfile.suffix") according to the templateLookupPolicy.
//
// The templateSuffixOrder defines the list of suffixes to try when searching for
// template files. The templateLookupPolicy controls whether to try template files
// before (TemplateLookupFirst), after (TemplateLookupLast), or not at all (TemplateLookupNever).
//
// If path is a regular file, it is returned directly without further searching.
func DiscoverContainerfileEx(path string, templateLookupPolicy define.TemplateLookupPolicy, templateSuffixOrder []string) (string, error) {
	// Test for existence of the file
	target, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("discovering Containerfile: %v", err)
	}

	switch mode := target.Mode(); {
	case mode.IsDir():
		// If the path is a real directory, we assume a Containerfile or a Dockerfile within it

		if templateLookupPolicy == define.TemplateLookupFirst {
			for _, suffix := range templateSuffixOrder {
				filePath, err := discoverTemplateContainerfile(path, suffix)
				if err == nil {
					return filePath, nil
				}
			}
		}

		filePath, err := discoverContainerfile(path)
		if err == nil {
			return filePath, nil
		}

		if templateLookupPolicy == define.TemplateLookupLast {
			for _, suffix := range templateSuffixOrder {
				filePath, err := discoverTemplateContainerfile(path, suffix)
				if err == nil {
					return filePath, nil
				}
			}
		}

		missingNames := strings.Join(containerFileNames, " or ")
		if len(containerFileNames) > 1 {
			missingNames = "either " + missingNames
		}
		return "", fmt.Errorf("cannot find %s in context directory", missingNames)

	case mode.IsRegular():
		// If the context dir is a file, we assume this as Containerfile
		return path, nil
	}

	return "", fmt.Errorf("path is not a file or directory: %s", path)
}

// DiscoverContainerfile tries to find a Containerfile or a Dockerfile within the provided `path`.
func DiscoverContainerfile(path string) (string, error) {
	return DiscoverContainerfileEx(path, define.TemplateLookupNever, nil)
}
