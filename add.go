package buildah

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/archive"
)

// addUrl copies the contents of the source URL to the destination.  This is
// its own function so that deferred closes happen after we're done pulling
// down each item of potentially many.
func addUrl(destination, srcurl string) error {
	logrus.Debugf("saving %q to %q", srcurl, destination)
	resp, err := http.Get(srcurl)
	if err != nil {
		return fmt.Errorf("error getting %q: %v", srcurl, err)
	}
	defer resp.Body.Close()
	f, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("error creating %q: %v", destination, err)
	}
	defer f.Close()
	n, err := io.Copy(f, resp.Body)
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return fmt.Errorf("error reading contents for %q: wrong length (%d != %d)", destination, n, resp.ContentLength)
	}
	if err := f.Chmod(0755); err != nil {
		return fmt.Errorf("error setting permissions on %q: %v", destination, err)
	}
	return nil
}

// Add copies contents into the container's root filesystem, optionally
// extracting contents of local files that look like non-empty archives.
func (b *Builder) Add(destination string, extract bool, source ...string) error {
	if b.MountPoint == "" {
		return fmt.Errorf("build container is not mounted")
	}
	dest := b.MountPoint
	if destination != "" && filepath.IsAbs(destination) {
		dest = filepath.Join(dest, destination)
	} else {
		dest = filepath.Join(dest, b.Workdir, destination)
	}
	for _, src := range source {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			// We assume that source is a file, and we're copying
			// it to the destination.  If we're copying multiple
			// things, or the destination already exists as a
			// directory, construct the new file's path using the
			// source's basename.  Otherwise guess that the
			// destination is a file.
			d := dest
			if len(source) > 1 {
				d = filepath.Join(dest, path.Base(src))
			} else if fi, err := os.Stat(dest); err == nil && fi.Mode().IsDir() {
				d = filepath.Join(dest, path.Base(src))
			}
			// Make sure the destination's directory exists.
			if err := os.MkdirAll(filepath.Dir(d), 0755); err != nil {
				return fmt.Errorf("error ensuring directory %q exists: %v)", filepath.Dir(d), err)
			}
			// Make sure we're not overwriting anything.
			if _, err := os.Stat(d); err == nil {
				return fmt.Errorf("%q already exists)", d)
			}
			// Save the contents.
			if err := addUrl(d, src); err != nil {
				return err
			}
			continue
		}
		fi, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("error reading %q: %v", src, err)
		}
		if fi.Mode().IsDir() {
			// The source is a directory, so we're creating a
			// subdirectory of the destination, which we expect
			// will also be a directory.  Make sure it exists.
			if err := os.MkdirAll(dest, 0755); err != nil {
				return fmt.Errorf("error ensuring directory %q exists: %v)", dest, err)
			}
			d := filepath.Join(dest, filepath.Base(src))
			logrus.Debugf("copying %q to %q", src+string(os.PathSeparator)+"*", d+string(os.PathSeparator)+"*")
			if err := archive.CopyWithTar(src, d); err != nil {
				return fmt.Errorf("error copying %q to %q: %v", src, d, err)
			}
			continue
		}
		if !extract || !archive.IsArchivePath(src) {
			// This source is a file, and either it's not an
			// archive, or we don't care whether or not it's an
			// archive.  If we're copying multiple things, or the
			// destination already exists as a directory, construct
			// the new file's path using the source's basename.
			// Otherwise guess that the destination is a file.
			d := dest
			if len(source) > 1 {
				d = filepath.Join(dest, filepath.Base(src))
			} else if fi, err := os.Stat(dest); err == nil && fi.Mode().IsDir() {
				d = filepath.Join(dest, filepath.Base(src))
			}
			// Make sure the destination's directory exists.
			if err := os.MkdirAll(filepath.Dir(d), 0755); err != nil {
				return fmt.Errorf("error ensuring directory %q exists: %v)", filepath.Dir(d), err)
			}
			// Make sure we're not overwriting anything.
			if _, err := os.Stat(d); err == nil {
				return fmt.Errorf("%q already exists)", d)
			}
			// Copy the file, preserving attributes.
			logrus.Debugf("copying %q to %q", src, d)
			if err := archive.CopyFileWithTar(src, d); err != nil {
				return fmt.Errorf("error copying %q to %q: %v", src, d, err)
			}
			continue
		}
		// We're extracting an archive, so the destination has to be a
		// directory.  Make sure it exists.
		if err := os.MkdirAll(dest, 0755); err != nil {
			return fmt.Errorf("error ensuring directory %q exists: %v)", dest, err)
		}
		logrus.Debugf("extracting contents of %q into %q", src, dest)
		if err := archive.UntarPath(src, dest); err != nil {
			return fmt.Errorf("error extracting %q into %q: %v", src, dest, err)
		}
	}
	return nil
}
