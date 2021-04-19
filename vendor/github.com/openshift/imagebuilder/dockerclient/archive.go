package dockerclient

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/fileutils"
	"github.com/containers/storage/pkg/idtools"
	"k8s.io/klog"
)

var isArchivePath = archive.IsArchivePath

// TransformFileFunc is given a chance to transform an arbitrary input file.
type TransformFileFunc func(h *tar.Header, r io.Reader) (data []byte, update bool, skip bool, err error)

// FilterArchive transforms the provided input archive to a new archive,
// giving the fn a chance to transform arbitrary files.
func FilterArchive(r io.Reader, w io.Writer, fn TransformFileFunc) error {
	tr := tar.NewReader(r)
	tw := tar.NewWriter(w)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			return tw.Close()
		}
		if err != nil {
			return err
		}

		var body io.Reader = tr
		name := h.Name
		data, ok, skip, err := fn(h, tr)
		klog.V(6).Infof("Transform %s -> %s: data=%t ok=%t skip=%t err=%v", name, h.Name, data != nil, ok, skip, err)
		if err != nil {
			return err
		}
		if skip {
			continue
		}
		if ok {
			h.Size = int64(len(data))
			body = bytes.NewBuffer(data)
		}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if _, err := io.Copy(tw, body); err != nil {
			return err
		}
	}
}

type CreateFileFunc func() (*tar.Header, io.ReadCloser, bool, error)

func NewLazyArchive(fn CreateFileFunc) io.ReadCloser {
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)
	go func() {
		for {
			h, r, more, err := fn()
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if h == nil {
				tw.Flush()
				pw.Close()
				return
			}
			if err := tw.WriteHeader(h); err != nil {
				r.Close()
				pw.CloseWithError(err)
				return
			}
			n, err := io.Copy(tw, &io.LimitedReader{R: r, N: h.Size})
			r.Close()
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if n != h.Size {
				pw.CloseWithError(fmt.Errorf("short read for %s", h.Name))
				return
			}
			if !more {
				tw.Flush()
				pw.Close()
				return
			}
		}
	}()
	return pr
}

func archiveFromURL(src, dst, tempDir string, check DirectoryCheck) (io.Reader, io.Closer, error) {
	// get filename from URL
	u, err := url.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	base := path.Base(u.Path)
	if base == "." {
		return nil, nil, fmt.Errorf("cannot determine filename from url: %s", u)
	}
	resp, err := http.Get(src)
	if err != nil {
		return nil, nil, err
	}
	archive := NewLazyArchive(func() (*tar.Header, io.ReadCloser, bool, error) {
		if resp.StatusCode >= 400 {
			return nil, nil, false, fmt.Errorf("server returned a status code >= 400: %s", resp.Status)
		}

		header := &tar.Header{
			Name: sourceToDestinationName(path.Base(u.Path), dst, false),
			Mode: 0600,
		}
		r := resp.Body
		if resp.ContentLength == -1 {
			f, err := ioutil.TempFile(tempDir, "url")
			if err != nil {
				return nil, nil, false, fmt.Errorf("unable to create temporary file for source URL: %v", err)
			}
			n, err := io.Copy(f, resp.Body)
			if err != nil {
				f.Close()
				return nil, nil, false, fmt.Errorf("unable to download source URL: %v", err)
			}
			if err := f.Close(); err != nil {
				return nil, nil, false, fmt.Errorf("unable to write source URL: %v", err)
			}
			f, err = os.Open(f.Name())
			if err != nil {
				return nil, nil, false, fmt.Errorf("unable to open downloaded source URL: %v", err)
			}
			r = f
			header.Size = n
		} else {
			header.Size = resp.ContentLength
		}
		return header, r, false, nil
	})
	return archive, closers{resp.Body.Close, archive.Close}, nil
}

func archiveFromDisk(directory string, src, dst string, allowDownload bool, excludes []string, check DirectoryCheck) (io.Reader, io.Closer, error) {
	var err error
	if filepath.IsAbs(src) {
		src, err = filepath.Rel(directory, filepath.Join(directory, src))
		if err != nil {
			return nil, nil, err
		}
	}

	infos, err := CalcCopyInfo(src, directory, true)
	if err != nil {
		return nil, nil, err
	}

	// special case when we are archiving a single file at the root
	if len(infos) == 1 && !infos[0].FileInfo.IsDir() && (infos[0].Path == "." || infos[0].Path == "/") {
		klog.V(5).Infof("Archiving a file instead of a directory from %s", directory)
		infos[0].Path = filepath.Base(directory)
		infos[0].FromDir = false
		directory = filepath.Dir(directory)
	}

	options, err := archiveOptionsFor(infos, dst, excludes, check)
	if err != nil {
		return nil, nil, err
	}

	klog.V(4).Infof("Tar of %s %#v", directory, options)
	rc, err := archive.TarWithOptions(directory, options)
	return rc, rc, err
}

func archiveFromFile(file string, src, dst string, excludes []string, check DirectoryCheck) (io.Reader, io.Closer, error) {
	var err error
	if filepath.IsAbs(src) {
		src, err = filepath.Rel(filepath.Dir(src), src)
		if err != nil {
			return nil, nil, err
		}
	}

	mapper, _, err := newArchiveMapper(src, dst, excludes, true, check)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}

	r, err := transformArchive(f, true, mapper.Filter)
	cc := newCloser(func() error {
		err := f.Close()
		if !mapper.foundItems {
			return makeNotExistError(src)
		}
		return err
	})
	return r, cc, err
}

func archiveFromContainer(in io.Reader, src, dst string, excludes []string, check DirectoryCheck) (io.ReadCloser, string, error) {
	mapper, archiveRoot, err := newArchiveMapper(src, dst, excludes, false, check)
	if err != nil {
		return nil, "", err
	}

	r, err := transformArchive(in, false, mapper.Filter)
	rc := readCloser{Reader: r, Closer: newCloser(func() error {
		if !mapper.foundItems {
			return makeNotExistError(src)
		}
		return nil
	})}
	return rc, archiveRoot, err
}

func transformArchive(r io.Reader, compressed bool, fn TransformFileFunc) (io.Reader, error) {
	pr, pw := io.Pipe()
	go func() {
		if compressed {
			in, err := archive.DecompressStream(r)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			r = in
		}
		err := FilterArchive(r, pw, fn)
		pw.CloseWithError(err)
	}()
	return pr, nil
}

// * -> test
// a (dir)  -> test
// a (file) -> test
// a (dir)  -> test/
// a (file) -> test/
//
func archivePathMapper(src, dst string, isDestDir bool) (fn func(name string, isDir bool) (string, bool)) {
	srcPattern := filepath.Clean(src)
	if srcPattern == "." {
		srcPattern = "*"
	}
	pattern := filepath.Base(srcPattern)

	klog.V(6).Infof("creating mapper for srcPattern=%s pattern=%s dst=%s isDestDir=%t", srcPattern, pattern, dst, isDestDir)

	// no wildcards
	if !containsWildcards(pattern) {
		return func(name string, isDir bool) (string, bool) {
			// when extracting from the working directory, Docker prefaces with ./
			if strings.HasPrefix(name, "."+string(filepath.Separator)) {
				name = name[2:]
			}
			if name == srcPattern {
				if isDir {
					return "", false
				}
				if isDestDir {
					return filepath.Join(dst, filepath.Base(name)), true
				}
				return dst, true
			}

			remainder := strings.TrimPrefix(name, srcPattern+string(filepath.Separator))
			if remainder == name {
				return "", false
			}
			return filepath.Join(dst, remainder), true
		}
	}

	// root with pattern
	prefix := filepath.Dir(srcPattern)
	if prefix == "." {
		return func(name string, isDir bool) (string, bool) {
			// match only on the first segment under the prefix
			var firstSegment = name
			if i := strings.Index(name, string(filepath.Separator)); i != -1 {
				firstSegment = name[:i]
			}
			ok, _ := filepath.Match(pattern, firstSegment)
			if !ok {
				return "", false
			}
			return filepath.Join(dst, name), true
		}
	}
	prefix += string(filepath.Separator)

	// nested with pattern
	return func(name string, isDir bool) (string, bool) {
		remainder := strings.TrimPrefix(name, prefix)
		if remainder == name {
			return "", false
		}
		// match only on the first segment under the prefix
		var firstSegment = remainder
		if i := strings.Index(remainder, string(filepath.Separator)); i != -1 {
			firstSegment = remainder[:i]
		}
		ok, _ := filepath.Match(pattern, firstSegment)
		if !ok {
			return "", false
		}
		return filepath.Join(dst, remainder), true
	}
}

type archiveMapper struct {
	exclude     *fileutils.PatternMatcher
	rename      func(name string, isDir bool) (string, bool)
	prefix      string
	resetOwners bool
	foundItems  bool
}

func newArchiveMapper(src, dst string, excludes []string, resetOwners bool, check DirectoryCheck) (*archiveMapper, string, error) {
	ex, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return nil, "", err
	}

	isDestDir := strings.HasSuffix(dst, "/") || path.Base(dst) == "."
	dst = path.Clean(dst)
	if !isDestDir && check != nil {
		isDir, err := check.IsDirectory(dst)
		if err != nil {
			return nil, "", err
		}
		isDestDir = isDir
	}

	var prefix string
	archiveRoot := src
	srcPattern := "*"
	switch {
	case src == "":
		return nil, "", fmt.Errorf("source may not be empty")
	case src == ".", src == "/":
		// no transformation necessary
	case strings.HasSuffix(src, "/"), strings.HasSuffix(src, "/."):
		src = path.Clean(src)
		archiveRoot = src
		if archiveRoot != "/" && archiveRoot != "." {
			prefix = path.Base(archiveRoot)
		}
	default:
		src = path.Clean(src)
		srcPattern = path.Base(src)
		archiveRoot = path.Dir(src)
		if archiveRoot != "/" && archiveRoot != "." {
			prefix = path.Base(archiveRoot)
		}
	}
	if !strings.HasSuffix(archiveRoot, "/") {
		archiveRoot += "/"
	}

	mapperFn := archivePathMapper(srcPattern, dst, isDestDir)

	return &archiveMapper{
		exclude:     ex,
		rename:      mapperFn,
		prefix:      prefix,
		resetOwners: resetOwners,
	}, archiveRoot, nil
}

func (m *archiveMapper) Filter(h *tar.Header, r io.Reader) ([]byte, bool, bool, error) {
	if m.resetOwners {
		h.Uid, h.Gid = 0, 0
	}
	// Trim a leading path, the prefix segment (which has no leading or trailing slashes), and
	// the final leader segment. Depending on the segment, Docker could return /prefix/ or prefix/.
	h.Name = strings.TrimPrefix(h.Name, "/")
	if !strings.HasPrefix(h.Name, m.prefix) {
		return nil, false, true, nil
	}
	h.Name = strings.TrimPrefix(strings.TrimPrefix(h.Name, m.prefix), "/")

	// skip a file if it doesn't match the src
	isDir := h.Typeflag == tar.TypeDir
	newName, ok := m.rename(h.Name, isDir)
	if !ok {
		return nil, false, true, nil
	}
	if newName == "." {
		return nil, false, true, nil
	}
	// skip based on excludes
	if ok, _ := m.exclude.Matches(h.Name); ok {
		return nil, false, true, nil
	}

	m.foundItems = true

	h.Name = newName

	if h.Typeflag == tar.TypeLink {
		// run the link target name through the same mapping the Name
		// in the target's entry would have gotten
		linkName := strings.TrimPrefix(h.Linkname, "/")
		if !strings.HasPrefix(linkName, m.prefix) {
			klog.V(6).Infof("No prefix %q in link target %q", m.prefix, h.Linkname)
			return nil, false, true, nil
		}
		linkName = strings.TrimPrefix(strings.TrimPrefix(linkName, m.prefix), "/")
		newTarget, ok := m.rename(linkName, false)
		if !ok {
			klog.V(6).Infof("Transform link target %s -> %s: ok=%t skip=%t", h.Linkname, newTarget, ok, true)
			return nil, false, true, nil
		}
		klog.V(6).Infof("Transform link target %s -> %s: ok=%t", h.Linkname, newTarget, ok)
		h.Linkname = newTarget
	}

	// include all files
	return nil, false, false, nil
}

func archiveOptionsFor(infos []CopyInfo, dst string, excludes []string, check DirectoryCheck) (*archive.TarOptions, error) {
	dst = trimLeadingPath(dst)
	dstIsDir := strings.HasSuffix(dst, "/") || dst == "." || dst == "/" || strings.HasSuffix(dst, "/.")
	dst = trimTrailingSlash(dst)
	dstIsRoot := dst == "." || dst == "/"

	if !dstIsDir && check != nil {
		isDir, err := check.IsDirectory(dst)
		if err != nil {
			return nil, fmt.Errorf("unable to check whether %s is a directory: %v", dst, err)
		}
		dstIsDir = isDir
	}

	options := &archive.TarOptions{
		ChownOpts: &idtools.IDPair{UID: 0, GID: 0},
	}

	pm, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return options, nil
	}

	for _, info := range infos {
		if ok, _ := pm.Matches(info.Path); ok {
			continue
		}

		srcIsDir := strings.HasSuffix(info.Path, "/") || info.Path == "." || info.Path == "/" || strings.HasSuffix(info.Path, "/.")
		infoPath := trimTrailingSlash(info.Path)

		options.IncludeFiles = append(options.IncludeFiles, infoPath)
		if len(dst) == 0 {
			continue
		}
		if options.RebaseNames == nil {
			options.RebaseNames = make(map[string]string)
		}

		klog.V(6).Infof("len=%d info.FromDir=%t info.IsDir=%t dstIsRoot=%t dstIsDir=%t srcIsDir=%t", len(infos), info.FromDir, info.IsDir(), dstIsRoot, dstIsDir, srcIsDir)
		switch {
		case len(infos) > 1 && dstIsRoot:
			// copying multiple things into root, no rename necessary ([Dockerfile, dir] -> [Dockerfile, dir])
		case len(infos) > 1:
			// put each input into the target, which is assumed to be a directory ([Dockerfile, dir] -> [a/Dockerfile, a/dir])
			options.RebaseNames[infoPath] = path.Join(dst, path.Base(infoPath))
		case info.FileInfo.IsDir():
			// mapping a directory to a destination, explicit or not ([dir] -> [a])
			options.RebaseNames[infoPath] = dst
		case info.FromDir:
			// this is a file that was part of an explicit directory request, no transformation
			options.RebaseNames[infoPath] = path.Join(dst, path.Base(infoPath))
		case dstIsDir:
			// mapping what is probably a file to a non-root directory ([Dockerfile] -> [dir/Dockerfile])
			options.RebaseNames[infoPath] = path.Join(dst, path.Base(infoPath))
		default:
			// a single file mapped to another single file ([Dockerfile] -> [Dockerfile.2])
			options.RebaseNames[infoPath] = dst
		}
	}

	options.ExcludePatterns = excludes
	return options, nil
}

func sourceToDestinationName(src, dst string, forceDir bool) string {
	switch {
	case forceDir, strings.HasSuffix(dst, "/"), path.Base(dst) == ".":
		return path.Join(dst, src)
	default:
		return dst
	}
}

// logArchiveOutput prints log info about the provided tar file as it is streamed. If an
// error occurs the remainder of the pipe is read to prevent blocking.
func logArchiveOutput(r io.Reader, prefix string) {
	pr, pw := io.Pipe()
	r = ioutil.NopCloser(io.TeeReader(r, pw))
	go func() {
		err := func() error {
			tr := tar.NewReader(pr)
			for {
				h, err := tr.Next()
				if err != nil {
					return err
				}
				klog.Infof("%s %s (%d %s)", prefix, h.Name, h.Size, h.FileInfo().Mode())
				if _, err := io.Copy(ioutil.Discard, tr); err != nil {
					return err
				}
			}
		}()
		if err != io.EOF {
			klog.Infof("%s: unable to log archive output: %v", prefix, err)
			io.Copy(ioutil.Discard, pr)
		}
	}()
}

type closer struct {
	closefn func() error
}

func newCloser(closeFunction func() error) *closer {
	return &closer{closefn: closeFunction}
}

func (r *closer) Close() error {
	return r.closefn()
}

type readCloser struct {
	io.Reader
	io.Closer
}
