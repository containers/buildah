package buildah

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chrootarchive"
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

// Add copies the contents of the specified sources into the container's root
// filesystem, optionally extracting contents of local files that look like
// non-empty archives.
func (b *Builder) Add(destination string, extract bool, source ...string) error {
	mountPoint, err := b.Mount("")
	if err != nil {
		return err
	}
	defer func() {
		if err2 := b.Unmount(); err2 != nil {
			logrus.Errorf("error unmounting container: %v", err2)
		}
	}()
	dest := mountPoint
	if destination != "" && filepath.IsAbs(destination) {
		dest = filepath.Join(dest, destination)
	} else {
		if err := os.MkdirAll(filepath.Join(dest, b.Workdir), 0755); err != nil {
			return fmt.Errorf("error ensuring directory %q exists: %v)", b.Workdir, err)
		}
		dest = filepath.Join(dest, b.Workdir, destination)
	}
	// Make sure the destination's parent directory is usable.
	if fi, err := os.Stat(filepath.Dir(dest)); err == nil && !fi.Mode().IsDir() {
		return fmt.Errorf("%q already exists, but is not a subdirectory)", filepath.Dir(dest))
	}
	// Now look at the destination itself.
	destfi, err := os.Stat(dest)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("couldn't determine what %q is: %v)", dest, err)
		}
		destfi = nil
	}
	if len(source) > 1 && (destfi == nil || !destfi.Mode().IsDir()) {
		return fmt.Errorf("destination %q is not a directory", dest)
	}
	for _, src := range source {
		if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
			// We assume that source is a file, and we're copying
			// it to the destination.  Compute a filename and save
			// the contents.
			url, err := url.Parse(src)
			if err != nil {
				return fmt.Errorf("error parsing URL %q: %v", src, err)
			}
			d := dest
			if destfi != nil && destfi.Mode().IsDir() {
				d = filepath.Join(dest, path.Base(url.Path))
			}
			if err := addUrl(d, src); err != nil {
				return err
			}
			continue
		}
		srcfi, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("error reading %q: %v", src, err)
		}
		if srcfi.Mode().IsDir() {
			// The source is a directory, so we're either creating
			// the destination or a subdirectory of the
			// destination.  Try to create it first, so that we can
			// detect if there's already something there.
			d := dest
			if destfi != nil && destfi.Mode().IsDir() {
				d = filepath.Join(dest, filepath.Base(src))
			}
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("error ensuring directory %q exists: %v)", dest, err)
			}
			logrus.Debugf("copying %q to %q", src+string(os.PathSeparator)+"*", d+string(os.PathSeparator)+"*")
			if err := chrootarchive.CopyWithTar(src, d); err != nil {
				return fmt.Errorf("error copying %q to %q: %v", src, d, err)
			}
			continue
		}
		if !extract || !archive.IsArchivePath(src) {
			// This source is a file, and either it's not an
			// archive, or we don't care whether or not it's an
			// archive.
			d := dest
			if destfi != nil && destfi.Mode().IsDir() {
				d = filepath.Join(dest, filepath.Base(src))
			}
			// Copy the file, preserving attributes.
			logrus.Debugf("copying %q to %q", src, d)
			if err := chrootarchive.CopyFileWithTar(src, d); err != nil {
				return fmt.Errorf("error copying %q to %q: %v", src, d, err)
			}
			continue
		}
		// We're extracting an archive into the destination directory.
		logrus.Debugf("extracting contents of %q into %q", src, dest)
		if err := chrootarchive.UntarPath(src, dest); err != nil {
			return fmt.Errorf("error extracting %q into %q: %v", src, dest, err)
		}
	}
	return nil
}
