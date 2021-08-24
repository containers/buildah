package copier

import (
	"archive/tar"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/reexec"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	flag.Parse()
	if testing.Verbose() {
		logrus.SetLevel(logrus.DebugLevel)
	}
	os.Exit(m.Run())
}

// makeFileContents creates contents for a file of a specified size
func makeContents(length int64) io.ReadCloser {
	pipeReader, pipeWriter := io.Pipe()
	buffered := bufio.NewWriter(pipeWriter)
	go func() {
		count := int64(0)
		for count < length {
			if _, err := buffered.Write([]byte{"0123456789abcdef"[count%16]}); err != nil {
				buffered.Flush()
				pipeWriter.CloseWithError(err) // nolint:errcheck
				return
			}
			count++
		}
		buffered.Flush()
		pipeWriter.Close()
	}()
	return pipeReader
}

// makeArchiveSlice creates an archive from the set of headers and returns a byte slice.
func makeArchiveSlice(headers []tar.Header) []byte {
	rc := makeArchive(headers, nil)
	defer rc.Close()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, rc); err != nil {
		panic("error creating in-memory archive")
	}
	return buf.Bytes()
}

// makeArchive creates an archive from the set of headers.
func makeArchive(headers []tar.Header, contents map[string][]byte) io.ReadCloser {
	if contents == nil {
		contents = make(map[string][]byte)
	}
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		var err error
		buffered := bufio.NewWriter(pipeWriter)
		tw := tar.NewWriter(buffered)
		for _, header := range headers {
			var fileContent []byte
			switch header.Typeflag {
			case tar.TypeLink, tar.TypeSymlink:
				header.Size = 0
			case tar.TypeReg:
				fileContent = contents[header.Name]
				if len(fileContent) != 0 {
					header.Size = int64(len(fileContent))
				}
			}
			if err = tw.WriteHeader(&header); err != nil {
				break
			}
			if header.Typeflag == tar.TypeReg && header.Size > 0 {
				var fileContents io.Reader
				if len(fileContent) > 0 {
					fileContents = bytes.NewReader(fileContent)
				} else {
					rc := makeContents(header.Size)
					defer rc.Close()
					fileContents = rc
				}
				if _, err = io.Copy(tw, fileContents); err != nil {
					break
				}
			}
		}
		tw.Close()
		buffered.Flush()
		if err != nil {
			pipeWriter.CloseWithError(err) // nolint:errcheck
		} else {
			pipeWriter.Close()
		}
	}()
	return pipeReader
}

// makeContextFromArchive creates a temporary directory, and a subdirectory
// inside of it, from an archive and returns its location.  It can be removed
// once it's no longer needed.
func makeContextFromArchive(archive io.ReadCloser, subdir string) (string, error) {
	tmp, err := ioutil.TempDir("", "copier-test-")
	if err != nil {
		return "", err
	}
	uidMap := []idtools.IDMap{{HostID: os.Getuid(), ContainerID: 0, Size: 1}}
	gidMap := []idtools.IDMap{{HostID: os.Getgid(), ContainerID: 0, Size: 1}}
	err = Put(tmp, path.Join(tmp, subdir), PutOptions{UIDMap: uidMap, GIDMap: gidMap}, archive)
	archive.Close()
	if err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	return tmp, err
}

// enumerateFiles walks a directory, returning the items it contains as a slice
// of names relative to that directory.
func enumerateFiles(directory string) ([]enumeratedFile, error) {
	var results []enumeratedFile
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if info == nil || err != nil {
			return err
		}
		rel, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		if rel != "" && rel != "." {
			results = append(results, enumeratedFile{
				name:      rel,
				mode:      info.Mode() & os.ModePerm,
				isSymlink: info.Mode()&os.ModeSymlink == os.ModeSymlink,
				date:      info.ModTime().UTC().String(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

type expectedError struct {
	inSubdir bool
	name     string
	err      error
}

type enumeratedFile struct {
	name      string
	mode      os.FileMode
	isSymlink bool
	date      string
}

var (
	testDate = time.Unix(1485449953, 0)

	uid, gid = os.Getuid(), os.Getgid()

	testArchiveSlice = makeArchiveSlice([]tar.Header{
		{Name: "item-0", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 123, Mode: 0600, ModTime: testDate},
		{Name: "item-1", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 456, Mode: 0600, ModTime: testDate},
		{Name: "item-2", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 789, Mode: 0600, ModTime: testDate},
	})

	testArchives = []struct {
		name              string
		rootOnly          bool
		headers           []tar.Header
		contents          map[string][]byte
		excludes          []string
		expectedGetErrors []expectedError
		subdirContents    map[string][]string
		renames           []struct {
			name     string
			renames  map[string]string
			expected []string
		}
	}{
		{
			name:     "regular",
			rootOnly: false,
			headers: []tar.Header{
				{Name: "file-0", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 123456789, Mode: 0600, ModTime: testDate},
				{Name: "file-a", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: 0600, ModTime: testDate},
				{Name: "file-b", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: 0600, ModTime: testDate},
				{Name: "file-c", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "file-a", Mode: 0600, ModTime: testDate},
				{Name: "file-u", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: cISUID | 0755, ModTime: testDate},
				{Name: "file-g", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: cISGID | 0755, ModTime: testDate},
				{Name: "file-t", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: cISVTX | 0755, ModTime: testDate},
				{Name: "link-0", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "../file-0", Size: 123456789, Mode: 0777, ModTime: testDate},
				{Name: "link-a", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "file-a", Size: 23, Mode: 0777, ModTime: testDate},
				{Name: "link-b", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "../file-a", Size: 23, Mode: 0777, ModTime: testDate},
				{Name: "hlink-0", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "file-0", Size: 123456789, Mode: 0600, ModTime: testDate},
				{Name: "hlink-a", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "/file-a", Size: 23, Mode: 0600, ModTime: testDate},
				{Name: "hlink-b", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "../file-b", Size: 23, Mode: 0600, ModTime: testDate},
				{Name: "subdir-a", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700, ModTime: testDate},
				{Name: "subdir-a/file-n", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 108, Mode: 0660, ModTime: testDate},
				{Name: "subdir-a/file-o", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 34, Mode: 0660, ModTime: testDate},
				{Name: "subdir-a/file-a", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "../file-a", Size: 23, Mode: 0777, ModTime: testDate},
				{Name: "subdir-a/file-b", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "../../file-b", Size: 23, Mode: 0777, ModTime: testDate},
				{Name: "subdir-a/file-c", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "/file-c", Size: 23, Mode: 0777, ModTime: testDate},
				{Name: "subdir-b", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700, ModTime: testDate},
				{Name: "subdir-b/file-n", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 216, Mode: 0660, ModTime: testDate},
				{Name: "subdir-b/file-o", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 45, Mode: 0660, ModTime: testDate},
				{Name: "subdir-c", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700, ModTime: testDate},
				{Name: "subdir-c/file-n", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 432, Mode: 0666, ModTime: testDate},
				{Name: "subdir-c/file-o", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 56, Mode: 0666, ModTime: testDate},
				{Name: "subdir-d", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700, ModTime: testDate},
				{Name: "subdir-d/hlink-0", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "../file-0", Size: 123456789, Mode: 0600, ModTime: testDate},
				{Name: "subdir-d/hlink-a", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "/file-a", Size: 23, Mode: 0600, ModTime: testDate},
				{Name: "subdir-d/hlink-b", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "../../file-b", Size: 23, Mode: 0600, ModTime: testDate},
				{Name: "archive-a", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 0, Mode: 0600, ModTime: testDate},
			},
			contents: map[string][]byte{
				"archive-a": testArchiveSlice,
			},
			expectedGetErrors: []expectedError{
				{inSubdir: false, name: "link-0", err: syscall.ENOENT},
				{inSubdir: false, name: "link-b", err: syscall.ENOENT},
				{inSubdir: false, name: "subdir-a/file-b", err: syscall.ENOENT},
				{inSubdir: true, name: "link-0", err: syscall.ENOENT},
				{inSubdir: true, name: "link-b", err: syscall.ENOENT},
				{inSubdir: true, name: "subdir-a/file-b", err: syscall.ENOENT},
				{inSubdir: true, name: "subdir-a/file-c", err: syscall.ENOENT},
			},
			renames: []struct {
				name     string
				renames  map[string]string
				expected []string
			}{
				{
					name:    "no-match-dir",
					renames: map[string]string{"subdir-z": "subdir-y"},
					expected: []string{
						"file-0",
						"file-a",
						"file-b",
						"file-c",
						"file-u",
						"file-g",
						"file-t",
						"link-0",
						"link-a",
						"link-b",
						"hlink-0",
						"hlink-a",
						"hlink-b",
						"subdir-a",
						"subdir-a/file-n",
						"subdir-a/file-o",
						"subdir-a/file-a",
						"subdir-a/file-b",
						"subdir-a/file-c",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-n",
						"subdir-c/file-o",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-d/hlink-a",
						"subdir-d/hlink-b",
						"archive-a",
					},
				},
				{
					name:    "no-match-file",
					renames: map[string]string{"file-n": "file-z"},
					expected: []string{
						"file-0",
						"file-a",
						"file-b",
						"file-c",
						"file-u",
						"file-g",
						"file-t",
						"link-0",
						"link-a",
						"link-b",
						"hlink-0",
						"hlink-a",
						"hlink-b",
						"subdir-a",
						"subdir-a/file-n",
						"subdir-a/file-o",
						"subdir-a/file-a",
						"subdir-a/file-b",
						"subdir-a/file-c",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-n",
						"subdir-c/file-o",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-d/hlink-a",
						"subdir-d/hlink-b",
						"archive-a",
					},
				},
				{
					name:    "directory",
					renames: map[string]string{"subdir-a": "subdir-z"},
					expected: []string{
						"file-0",
						"file-a",
						"file-b",
						"file-c",
						"file-u",
						"file-g",
						"file-t",
						"link-0",
						"link-a",
						"link-b",
						"hlink-0",
						"hlink-a",
						"hlink-b",
						"subdir-z",
						"subdir-z/file-n",
						"subdir-z/file-o",
						"subdir-z/file-a",
						"subdir-z/file-b",
						"subdir-z/file-c",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-n",
						"subdir-c/file-o",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-d/hlink-a",
						"subdir-d/hlink-b",
						"archive-a",
					},
				},
				{
					name:    "file-in-directory",
					renames: map[string]string{"subdir-a/file-n": "subdir-a/file-z"},
					expected: []string{
						"file-0",
						"file-a",
						"file-b",
						"file-c",
						"file-u",
						"file-g",
						"file-t",
						"link-0",
						"link-a",
						"link-b",
						"hlink-0",
						"hlink-a",
						"hlink-b",
						"subdir-a",
						"subdir-a/file-z",
						"subdir-a/file-o",
						"subdir-a/file-a",
						"subdir-a/file-b",
						"subdir-a/file-c",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-n",
						"subdir-c/file-o",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-d/hlink-a",
						"subdir-d/hlink-b",
						"archive-a",
					},
				},
			},
		},
		{
			name:     "devices",
			rootOnly: true,
			headers: []tar.Header{
				{Name: "char-dev", Uid: uid, Gid: gid, Typeflag: tar.TypeChar, Devmajor: 0, Devminor: 0, Mode: 0600, ModTime: testDate},
				{Name: "blk-dev", Uid: uid, Gid: gid, Typeflag: tar.TypeBlock, Devmajor: 0, Devminor: 0, Mode: 0600, ModTime: testDate},
			},
		},
	}
)

func TestPutNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testPut(t)
	canChroot = couldChroot
}

func testPut(t *testing.T) {
	uidMap := []idtools.IDMap{{HostID: os.Getuid(), ContainerID: 0, Size: 1}}
	gidMap := []idtools.IDMap{{HostID: os.Getgid(), ContainerID: 0, Size: 1}}

	for i := range testArchives {
		for _, topdir := range []string{"", ".", "top"} {
			t.Run(fmt.Sprintf("archive=%s,topdir=%s", testArchives[i].name, topdir), func(t *testing.T) {
				if uid != 0 && testArchives[i].rootOnly {
					t.Skipf("test archive %q can only be tested with root privileges, skipping", testArchives[i].name)
				}

				dir, err := makeContextFromArchive(makeArchive(testArchives[i].headers, testArchives[i].contents), topdir)
				require.NoErrorf(t, err, "error creating context from archive %q, topdir=%q", testArchives[i].name, topdir)
				defer os.RemoveAll(dir)

				// enumerate what we expect to have created
				expected := make([]enumeratedFile, 0, len(testArchives[i].headers)+1)
				if topdir != "" && topdir != "." {
					info, err := os.Stat(filepath.Join(dir, topdir))
					require.NoErrorf(t, err, "error statting directory %q", filepath.Join(dir, topdir))
					expected = append(expected, enumeratedFile{
						name:      filepath.FromSlash(topdir),
						mode:      info.Mode() & os.ModePerm,
						isSymlink: info.Mode()&os.ModeSymlink == os.ModeSymlink,
						date:      info.ModTime().UTC().String(),
					})
				}
				for _, hdr := range testArchives[i].headers {
					expected = append(expected, enumeratedFile{
						name:      filepath.Join(filepath.FromSlash(topdir), filepath.FromSlash(hdr.Name)),
						mode:      os.FileMode(hdr.Mode) & os.ModePerm,
						isSymlink: hdr.Typeflag == tar.TypeSymlink,
						date:      hdr.ModTime.UTC().String(),
					})
				}
				sort.Slice(expected, func(i, j int) bool { return strings.Compare(expected[i].name, expected[j].name) < 0 })

				// enumerate what we actually created
				fileList, err := enumerateFiles(dir)
				require.NoErrorf(t, err, "error walking context directory for archive %q, topdir=%q", testArchives[i].name, topdir)
				sort.Slice(fileList, func(i, j int) bool { return strings.Compare(fileList[i].name, fileList[j].name) < 0 })

				// make sure they're the same
				moddedEnumeratedFiles := func(enumerated []enumeratedFile) []enumeratedFile {
					m := make([]enumeratedFile, 0, len(enumerated))
					for i := range enumerated {
						e := enumeratedFile{
							name:      enumerated[i].name,
							mode:      os.FileMode(int64(enumerated[i].mode) & testModeMask),
							isSymlink: enumerated[i].isSymlink,
							date:      enumerated[i].date,
						}
						if testIgnoreSymlinkDates && e.isSymlink {
							e.date = ""
						}
						m = append(m, e)
					}
					return m
				}
				if !reflect.DeepEqual(expected, fileList) && reflect.DeepEqual(moddedEnumeratedFiles(expected), moddedEnumeratedFiles(fileList)) {
					logrus.Warn("chmod() lost some bits and possibly timestamps on symlinks, otherwise we match the source archive")
				} else {
					require.Equal(t, expected, fileList, "list of files in context directory for archive %q under topdir %q should match the archived used to populate it", testArchives[i].name, topdir)
				}
			})
		}

		for _, renames := range testArchives[i].renames {
			t.Run(fmt.Sprintf("archive=%s,rename=%s", testArchives[i].name, renames.name), func(t *testing.T) {
				if uid != 0 && testArchives[i].rootOnly {
					t.Skipf("test archive %q can only be tested with root privileges, skipping", testArchives[i].name)
				}

				tmp, err := ioutil.TempDir("", "copier-test-")
				require.NoErrorf(t, err, "error creating temporary directory")
				defer os.RemoveAll(tmp)

				archive := makeArchive(testArchives[i].headers, testArchives[i].contents)
				err = Put(tmp, tmp, PutOptions{UIDMap: uidMap, GIDMap: gidMap, Rename: renames.renames}, archive)
				require.NoErrorf(t, err, "error extracting archive %q to directory %q", testArchives[i].name, tmp)

				var found []string
				err = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
					if info == nil || err != nil {
						return err
					}
					rel, err := filepath.Rel(tmp, path)
					if err != nil {
						return err
					}
					if rel == "." {
						return nil
					}
					found = append(found, rel)
					return nil
				})
				require.NoErrorf(t, err, "error walking context directory for archive %q under %q", testArchives[i].name, tmp)
				sort.Strings(found)

				expected := renames.expected
				sort.Strings(expected)
				assert.Equal(t, expected, found, "renaming did not work as expected")
			})
		}
	}

	for _, overwrite := range []bool{false, true} {
		for _, typeFlag := range []byte{tar.TypeReg, tar.TypeLink, tar.TypeSymlink, tar.TypeChar, tar.TypeBlock, tar.TypeFifo} {
			t.Run(fmt.Sprintf("overwrite=%v,type=%c", overwrite, typeFlag), func(t *testing.T) {
				archive := makeArchiveSlice([]tar.Header{
					{Name: "target", Typeflag: tar.TypeReg, Size: 123, Mode: 0755, ModTime: testDate},
					{Name: "test", Typeflag: tar.TypeDir, Size: 0, Mode: 0755, ModTime: testDate},
					{Name: "test", Typeflag: typeFlag, Size: 0, Mode: 0755, Linkname: "target", ModTime: testDate},
				})
				tmp, err := ioutil.TempDir("", "copier-test-")
				require.NoErrorf(t, err, "error creating temporary directory")
				defer os.RemoveAll(tmp)
				err = Put(tmp, tmp, PutOptions{UIDMap: uidMap, GIDMap: gidMap, NoOverwriteDirNonDir: !overwrite}, bytes.NewReader(archive))
				if overwrite {
					if unwrapError(err) != syscall.EPERM {
						assert.Nilf(t, err, "expected to overwrite directory with type %c: %v", typeFlag, err)
					}
				} else {
					assert.Errorf(t, err, "expected an error trying to overwrite directory with type %c", typeFlag)
				}
			})
		}
	}

	for _, ignoreDevices := range []bool{false, true} {
		for _, typeFlag := range []byte{tar.TypeChar, tar.TypeBlock} {
			t.Run(fmt.Sprintf("ignoreDevices=%v,type=%c", ignoreDevices, typeFlag), func(t *testing.T) {
				if uid != 0 && !ignoreDevices {
					t.Skip("can only test !IgnoreDevices with root privileges, skipping")
				}
				archive := makeArchiveSlice([]tar.Header{
					{Name: "test", Typeflag: typeFlag, Size: 0, Mode: 0600, ModTime: testDate, Devmajor: 0, Devminor: 0},
					{Name: "link", Typeflag: tar.TypeLink, Size: 0, Mode: 0600, ModTime: testDate, Linkname: "test"},
					{Name: "unrelated", Typeflag: tar.TypeReg, Size: 0, Mode: 0600, ModTime: testDate},
				})
				tmp, err := ioutil.TempDir("", "copier-test-")
				require.NoErrorf(t, err, "error creating temporary directory")
				defer os.RemoveAll(tmp)
				err = Put(tmp, tmp, PutOptions{UIDMap: uidMap, GIDMap: gidMap, IgnoreDevices: ignoreDevices}, bytes.NewReader(archive))
				require.Nilf(t, err, "expected to extract content with typeflag %c without an error: %v", typeFlag, err)
				fileList, err := enumerateFiles(tmp)
				require.Nilf(t, err, "unexpected error scanning the contents of extraction directory for typeflag %c: %v", typeFlag, err)
				expectedItems := 3
				if ignoreDevices {
					expectedItems = 1
				}
				require.Equalf(t, expectedItems, len(fileList), "didn't extract as many things as expected for typeflag %c", typeFlag)
			})
		}
	}
}

func isExpectedError(err error, inSubdir bool, name string, expectedErrors []expectedError) bool {
	// if we couldn't read that content, check if it's one of the expected failures
	for _, expectedError := range expectedErrors {
		if expectedError.inSubdir != inSubdir {
			continue
		}
		if expectedError.name != name {
			continue
		}
		if !strings.Contains(unwrapError(err).Error(), unwrapError(expectedError.err).Error()) {
			// not expecting this specific error
			continue
		}
		// it's an expected failure
		return true
	}
	return false
}

func TestStatNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testStat(t)
	canChroot = couldChroot
}

func testStat(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		for _, topdir := range []string{"", ".", "top"} {
			for _, testArchive := range testArchives {
				if uid != 0 && testArchive.rootOnly {
					t.Logf("test archive %q can only be tested with root privileges, skipping", testArchive.name)
					continue
				}

				dir, err := makeContextFromArchive(makeArchive(testArchive.headers, testArchive.contents), topdir)
				require.NoErrorf(t, err, "error creating context from archive %q", testArchive.name)
				defer os.RemoveAll(dir)

				root := dir

				for _, testItem := range testArchive.headers {
					name := filepath.FromSlash(testItem.Name)
					if absolute {
						name = filepath.Join(root, topdir, name)
					}
					t.Run(fmt.Sprintf("absolute=%t,topdir=%s,archive=%s,item=%s", absolute, topdir, testArchive.name, name), func(t *testing.T) {
						// read stats about this item
						var excludes []string
						for _, exclude := range testArchive.excludes {
							excludes = append(excludes, filepath.FromSlash(exclude))
						}
						options := StatOptions{
							CheckForArchives: false,
							Excludes:         excludes,
						}
						stats, err := Stat(root, topdir, options, []string{name})
						require.NoErrorf(t, err, "error statting %q: %v", name, err)
						for _, st := range stats {
							// should not have gotten an error
							require.Emptyf(t, st.Error, "expected no error from stat %q", st.Glob)
							// no matching characters -> should have matched one item
							require.NotEmptyf(t, st.Globbed, "expected at least one match on glob %q", st.Glob)
							matches := 0
							for _, glob := range st.Globbed {
								matches++
								require.Equal(t, st.Glob, glob, "expected entry for %q", st.Glob)
								require.NotNil(t, st.Results[glob], "%q globbed %q, but there are no results for it", st.Glob, glob)
								toStat := glob
								if !absolute {
									toStat = filepath.Join(root, topdir, name)
								}
								_, err = os.Lstat(toStat)
								require.NoErrorf(t, err, "got error on lstat() of returned value %q(%q(%q)): %v", toStat, glob, name, err)
								result := st.Results[glob]

								switch testItem.Typeflag {
								case tar.TypeReg, tar.TypeRegA:
									if actualContent, ok := testArchive.contents[testItem.Name]; ok {
										testItem.Size = int64(len(actualContent))
									}
									require.Equal(t, testItem.Size, result.Size, "unexpected size difference for %q", name)
									require.True(t, result.IsRegular, "expected %q.IsRegular to be true", glob)
									require.False(t, result.IsDir, "expected %q.IsDir to be false", glob)
									require.False(t, result.IsSymlink, "expected %q.IsSymlink to be false", glob)
								case tar.TypeDir:
									require.False(t, result.IsRegular, "expected %q.IsRegular to be false", glob)
									require.True(t, result.IsDir, "expected %q.IsDir to be true", glob)
									require.False(t, result.IsSymlink, "expected %q.IsSymlink to be false", glob)
								case tar.TypeSymlink:
									require.True(t, result.IsSymlink, "%q is supposed to be a symbolic link, but is not", name)
									require.Equal(t, filepath.FromSlash(testItem.Linkname), result.ImmediateTarget, "%q is supposed to point to %q, but points to %q", glob, testItem.Linkname, result.ImmediateTarget)
								case tar.TypeBlock, tar.TypeChar:
									require.False(t, result.IsRegular, "%q is a regular file, but is not supposed to be", name)
									require.False(t, result.IsDir, "%q is a directory, but is not supposed to be", name)
									require.False(t, result.IsSymlink, "%q is not supposed to be a symbolic link, but appears to be one", name)
								}
							}
							require.Equal(t, 1, matches, "non-glob %q matched %d items, not exactly one", name, matches)
						}
					})
				}
			}
		}
	}
}

func TestGetSingleNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testGetSingle(t)
	canChroot = couldChroot
}

func testGetSingle(t *testing.T) {
	for _, absolute := range []bool{false, true} {
		for _, topdir := range []string{"", ".", "top"} {
			for _, testArchive := range testArchives {
				var excludes []string
				for _, exclude := range testArchive.excludes {
					excludes = append(excludes, filepath.FromSlash(exclude))
				}

				getOptions := GetOptions{
					Excludes:       excludes,
					ExpandArchives: false,
				}

				if uid != 0 && testArchive.rootOnly {
					t.Logf("test archive %q can only be tested with root privileges, skipping", testArchive.name)
					continue
				}

				dir, err := makeContextFromArchive(makeArchive(testArchive.headers, testArchive.contents), topdir)
				require.NoErrorf(t, err, "error creating context from archive %q", testArchive.name)
				defer os.RemoveAll(dir)

				root := dir

				for _, testItem := range testArchive.headers {
					name := filepath.FromSlash(testItem.Name)
					if absolute {
						name = filepath.Join(root, topdir, name)
					}
					t.Run(fmt.Sprintf("absolute=%t,topdir=%s,archive=%s,item=%s", absolute, topdir, testArchive.name, name), func(t *testing.T) {
						// check if we can get this one item
						err := Get(root, topdir, getOptions, []string{name}, ioutil.Discard)
						// if we couldn't read that content, check if it's one of the expected failures
						if err != nil && isExpectedError(err, topdir != "" && topdir != ".", testItem.Name, testArchive.expectedGetErrors) {
							return
						}
						require.NoErrorf(t, err, "error getting %q under %q", name, filepath.Join(root, topdir))
						// we'll check subdirectories later
						if testItem.Typeflag == tar.TypeDir {
							return
						}
						// check what we get when we get this one item
						pipeReader, pipeWriter := io.Pipe()
						var getErr error
						var wg sync.WaitGroup
						wg.Add(1)
						go func() {
							getErr = Get(root, topdir, getOptions, []string{name}, pipeWriter)
							pipeWriter.Close()
							wg.Done()
						}()
						tr := tar.NewReader(pipeReader)
						hdr, err := tr.Next()
						for err == nil {
							assert.Equal(t, filepath.Base(name), filepath.FromSlash(hdr.Name), "expected item named %q, got %q", filepath.Base(name), filepath.FromSlash(hdr.Name))
							hdr, err = tr.Next()
						}
						assert.Equal(t, io.EOF.Error(), err.Error(), "expected EOF at end of archive, got %q", err.Error())
						if !t.Failed() && testItem.Typeflag == tar.TypeReg && testItem.Mode&(cISUID|cISGID|cISVTX) != 0 {
							for _, stripSetuidBit := range []bool{false, true} {
								for _, stripSetgidBit := range []bool{false, true} {
									for _, stripStickyBit := range []bool{false, true} {
										t.Run(fmt.Sprintf("absolute=%t,topdir=%s,archive=%s,item=%s,strip_setuid=%t,strip_setgid=%t,strip_sticky=%t", absolute, topdir, testArchive.name, name, stripSetuidBit, stripSetgidBit, stripStickyBit), func(t *testing.T) {
											var getErr error
											var wg sync.WaitGroup
											getOptions := getOptions
											getOptions.StripSetuidBit = stripSetuidBit
											getOptions.StripSetgidBit = stripSetgidBit
											getOptions.StripStickyBit = stripStickyBit
											pipeReader, pipeWriter := io.Pipe()
											wg.Add(1)
											go func() {
												getErr = Get(root, topdir, getOptions, []string{name}, pipeWriter)
												pipeWriter.Close()
												wg.Done()
											}()
											tr := tar.NewReader(pipeReader)
											hdr, err := tr.Next()
											for err == nil {
												expectedMode := testItem.Mode
												modifier := ""
												if stripSetuidBit {
													expectedMode &^= cISUID
													modifier += "(with setuid bit stripped) "
												}
												if stripSetgidBit {
													expectedMode &^= cISGID
													modifier += "(with setgid bit stripped) "
												}
												if stripStickyBit {
													expectedMode &^= cISVTX
													modifier += "(with sticky bit stripped) "
												}
												if expectedMode != hdr.Mode && expectedMode&testModeMask == hdr.Mode&testModeMask {
													logrus.Warnf("chmod() lost some bits: expected 0%o, got 0%o", expectedMode, hdr.Mode)
												} else {
													assert.Equal(t, expectedMode, hdr.Mode, "expected item named %q %sto have mode 0%o, got 0%o", hdr.Name, modifier, expectedMode, hdr.Mode)
												}
												hdr, err = tr.Next()
											}
											assert.Equal(t, io.EOF.Error(), err.Error(), "expected EOF at end of archive, got %q", err.Error())
											wg.Wait()
											assert.NoErrorf(t, getErr, "unexpected error from Get(%q): %v", name, getErr)
											pipeReader.Close()
										})
									}
								}
							}
						}

						wg.Wait()
						assert.NoErrorf(t, getErr, "unexpected error from Get(%q): %v", name, getErr)
						pipeReader.Close()
					})
				}
			}
		}
	}
}

func TestGetMultipleNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testGetMultiple(t)
	canChroot = couldChroot
}

func testGetMultiple(t *testing.T) {
	type getTestArchiveCase struct {
		name               string
		pattern            string
		exclude            []string
		items              []string
		expandArchives     bool
		stripSetuidBit     bool
		stripSetgidBit     bool
		stripStickyBit     bool
		stripXattrs        bool
		keepDirectoryNames bool
		renames            map[string]string
		noDerefSymlinks    bool
	}
	var getTestArchives = []struct {
		name              string
		headers           []tar.Header
		contents          map[string][]byte
		cases             []getTestArchiveCase
		expectedGetErrors []expectedError
	}{
		{
			name: "regular",
			headers: []tar.Header{
				{Name: "file-0", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 123456789, Mode: 0600},
				{Name: "file-a", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: 0600},
				{Name: "file-b", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 23, Mode: 0600},
				{Name: "link-a", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "file-a", Size: 23, Mode: 0600},
				{Name: "link-c", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "subdir-c", Mode: 0700, ModTime: testDate},
				{Name: "archive-a", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 0, Mode: 0600},
				{Name: "non-archive-a", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 1199, Mode: 0600},
				{Name: "hlink-0", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "file-0", Size: 123456789, Mode: 0600},
				{Name: "something-a", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 34, Mode: 0600},
				{Name: "subdir-a", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700},
				{Name: "subdir-a/file-n", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 108, Mode: 0660},
				{Name: "subdir-a/file-o", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 45, Mode: 0660},
				{Name: "subdir-a/file-a", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "../file-a", Size: 23, Mode: 0600},
				{Name: "subdir-a/file-b", Uid: uid, Gid: gid, Typeflag: tar.TypeSymlink, Linkname: "../../file-b", Size: 23, Mode: 0600},
				{Name: "subdir-a/file-c", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 56, Mode: 0600},
				{Name: "subdir-b", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700},
				{Name: "subdir-b/file-n", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 216, Mode: 0660},
				{Name: "subdir-b/file-o", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 67, Mode: 0660},
				{Name: "subdir-c", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700},
				{Name: "subdir-c/file-p", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 432, Mode: 0666},
				{Name: "subdir-c/file-q", Uid: uid, Gid: gid, Typeflag: tar.TypeReg, Size: 78, Mode: 0666},
				{Name: "subdir-d", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700},
				{Name: "subdir-d/hlink-0", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "../file-0", Size: 123456789, Mode: 0600},
				{Name: "subdir-e", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700},
				{Name: "subdir-e/subdir-f", Uid: uid, Gid: gid, Typeflag: tar.TypeDir, Mode: 0700},
				{Name: "subdir-e/subdir-f/hlink-b", Uid: uid, Gid: gid, Typeflag: tar.TypeLink, Linkname: "../../file-b", Size: 23, Mode: 0600},
			},
			contents: map[string][]byte{
				"archive-a": testArchiveSlice,
			},
			expectedGetErrors: []expectedError{
				{inSubdir: true, name: ".", err: syscall.ENOENT},
				{inSubdir: true, name: "/subdir-b/*", err: syscall.ENOENT},
				{inSubdir: true, name: "../../subdir-b/*", err: syscall.ENOENT},
			},
			cases: []getTestArchiveCase{
				{
					name:    "everything",
					pattern: ".",
					items: []string{
						"file-0",
						"file-a",
						"file-b",
						"link-a",
						"link-c",
						"hlink-0",
						"something-a",
						"archive-a",
						"non-archive-a",
						"subdir-a",
						"subdir-a/file-n",
						"subdir-a/file-o",
						"subdir-a/file-a",
						"subdir-a/file-b",
						"subdir-a/file-c",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-p",
						"subdir-c/file-q",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-e",
						"subdir-e/subdir-f",
						"subdir-e/subdir-f/hlink-b",
					},
				},
				{
					name:    "wildcard",
					pattern: "*",
					items: []string{
						"file-0",
						"file-a",
						"file-b",
						"link-a",
						"hlink-0",
						"something-a",
						"archive-a",
						"non-archive-a",
						"file-n",           // from subdir-a
						"file-o",           // from subdir-a
						"file-a",           // from subdir-a
						"file-b",           // from subdir-a
						"file-c",           // from subdir-a
						"file-n",           // from subdir-b
						"file-o",           // from subdir-b
						"file-p",           // from subdir-c
						"file-p",           // from link-c -> subdir-c
						"file-q",           // from subdir-c
						"file-q",           // from link-c -> subdir-c
						"hlink-0",          // from subdir-d
						"subdir-f",         // from subdir-e
						"subdir-f/hlink-b", // from subdir-e
					},
				},
				{
					name:    "dot-with-wildcard-includes-and-excludes",
					pattern: ".",
					exclude: []string{"**/*-a", "!**/*-c"},
					items: []string{
						"file-0",
						"file-b",
						"link-c",
						"hlink-0",
						// "subdir-a/file-c", // strings.HasPrefix("**/*-c", "subdir-a/") is false
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-p",
						"subdir-c/file-q",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-e",
						"subdir-e/subdir-f",
						"subdir-e/subdir-f/hlink-b",
					},
				},
				{
					name:    "everything-with-wildcard-includes-and-excludes",
					pattern: "*",
					exclude: []string{"**/*-a", "!**/*-c"},
					items: []string{
						"file-0",
						"file-b",
						"file-c",
						"file-n",
						"file-o",
						"file-p", // from subdir-c
						"file-p", // from link-c -> subdir-c
						"file-q", // from subdir-c
						"file-q", // from link-c -> subdir-c
						"hlink-0",
						"hlink-0",
						"subdir-f",
						"subdir-f/hlink-b",
					},
				},
				{
					name:    "dot-with-dot-exclude",
					pattern: ".",
					exclude: []string{".", "!**/*-c"},
					items: []string{
						"file-0",
						"file-a",
						"file-b",
						"link-a",
						"link-c",
						"hlink-0",
						"something-a",
						"archive-a",
						"non-archive-a",
						"subdir-a",
						"subdir-a/file-a",
						"subdir-a/file-b",
						"subdir-a/file-c",
						"subdir-a/file-n",
						"subdir-a/file-o",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-c",
						"subdir-c/file-p",
						"subdir-c/file-q",
						"subdir-d",
						"subdir-d/hlink-0",
						"subdir-e",
						"subdir-e/subdir-f",
						"subdir-e/subdir-f/hlink-b",
					},
				},
				{
					name:    "everything-with-dot-exclude",
					pattern: "*",
					exclude: []string{".", "!**/*-c"},
					items: []string{
						"file-0",
						"file-a",
						"file-a",
						"file-b",
						"file-b",
						"file-c",
						"file-n",
						"file-n",
						"file-o",
						"file-o",
						"file-p",
						"file-p",
						"file-q",
						"file-q",
						"hlink-0",
						"hlink-0",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
						"subdir-f",
						"subdir-f/hlink-b",
					},
				},
				{
					name:    "all-with-all-exclude",
					pattern: "*",
					exclude: []string{"*", "!**/*-c"},
					items: []string{
						"file-c",
						"file-p",
						"file-p",
						"file-q",
						"file-q",
					},
				},
				{
					name:    "everything-with-all-exclude",
					pattern: ".",
					exclude: []string{"*", "!**/*-c"},
					items: []string{
						// "subdir-a/file-c", // strings.HasPrefix("**/*-c", "subdir-a/") is false
						"link-c",
						"subdir-c",
						"subdir-c/file-p",
						"subdir-c/file-q",
					},
				},
				{
					name:    "file-wildcard",
					pattern: "file-*",
					items: []string{
						"file-0",
						"file-a",
						"file-b",
					},
				},
				{
					name:    "file-and-dir-wildcard",
					pattern: "*-a",
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
						"file-n", // from subdir-a
						"file-o", // from subdir-a
						"file-a", // from subdir-a
						"file-b", // from subdir-a
						"file-c", // from subdir-a
					},
				},
				{
					name:    "file-and-dir-wildcard-with-exclude",
					pattern: "*-a",
					exclude: []string{"subdir-a", "top/subdir-a"},
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
					},
				},
				{
					name:    "file-and-dir-wildcard-with-wildcard-exclude",
					pattern: "*-a",
					exclude: []string{"subdir*", "top/subdir*"},
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
					},
				},
				{
					name:    "file-and-dir-wildcard-with-deep-exclude",
					pattern: "*-a",
					exclude: []string{"**/subdir-a"},
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
					},
				},
				{
					name:    "file-and-dir-wildcard-with-wildcard-deep-exclude",
					pattern: "*-a",
					exclude: []string{"**/subdir*"},
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
					},
				},
				{
					name:    "file-and-dir-wildcard-with-deep-include",
					pattern: "*-a",
					exclude: []string{"**/subdir-a", "!**/file-c"},
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
						"file-c",
					},
				},
				{
					name:    "file-and-dir-wildcard-with-wildcard-deep-include",
					pattern: "*-a",
					exclude: []string{"**/subdir*", "!**/file-c"},
					items: []string{
						"file-a",
						"link-a",
						"something-a",
						"archive-a",
						"non-archive-a",
						"file-c",
					},
				},
				{
					name:    "subdirectory",
					pattern: "subdir-e",
					items: []string{
						"subdir-f",
						"subdir-f/hlink-b",
					},
				},
				{
					name:    "subdirectory-wildcard",
					pattern: "*/subdir-*",
					items: []string{
						"hlink-b", // from subdir-e/subdir-f
					},
				},
				{
					name:    "not-expanded-archive",
					pattern: "*archive-a",
					items: []string{
						"archive-a",
						"non-archive-a",
					},
				},
				{
					name:           "expanded-archive",
					pattern:        "*archive-a",
					expandArchives: true,
					items: []string{
						"non-archive-a",
						"item-0",
						"item-1",
						"item-2",
					},
				},
				{
					name:    "subdir-without-name",
					pattern: "subdir-e",
					items: []string{
						"subdir-f",
						"subdir-f/hlink-b",
					},
				},
				{
					name:               "subdir-with-name",
					pattern:            "subdir-e",
					keepDirectoryNames: true,
					items: []string{
						"subdir-e",
						"subdir-e/subdir-f",
						"subdir-e/subdir-f/hlink-b",
					},
				},
				{
					name:               "root-wildcard",
					pattern:            "/subdir-b/*",
					keepDirectoryNames: false,
					items: []string{
						"file-n",
						"file-o",
					},
				},
				{
					name:               "dotdot-wildcard",
					pattern:            "../../subdir-b/*",
					keepDirectoryNames: false,
					items: []string{
						"file-n",
						"file-o",
					},
				},
				{
					name:               "wildcard-with-rename",
					pattern:            "*-a",
					keepDirectoryNames: false,
					renames:            map[string]string{"file-a": "renamed"},
					items: []string{
						"renamed", // from file-a
						"link-a",
						"archive-a",
						"non-archive-a",
						"something-a",
						"file-n",  // from subdir-a
						"file-o",  // from subdir-a
						"renamed", // from subdir-a/file-a -> file-a -> renamed
						"file-b",  // from subdir-a
						"file-c",  // from subdir-a
					},
				},
				{
					name:               "wildcard-with-rename-keep",
					pattern:            "*-a",
					keepDirectoryNames: true,
					renames:            map[string]string{"subdir-a": "subdir-b"},
					items: []string{
						"file-a",
						"link-a",
						"archive-a",
						"non-archive-a",
						"something-a",
						"subdir-b",
						"subdir-b/file-n",
						"subdir-b/file-o",
						"subdir-b/file-a",
						"subdir-b/file-b",
						"subdir-b/file-c",
					},
				},
				{
					name:            "no-deref-symlinks-baseline",
					pattern:         "*-a",
					noDerefSymlinks: true,
					items: []string{
						"file-a",
						"link-a",
						"archive-a",
						"non-archive-a",
						"something-a",
						"file-n", // from subdir-a
						"file-o", // from subdir-a
						"file-a", // from subdir-a
						"file-b", // from subdir-a
						"file-c", // from subdir-a
					},
				},
				{
					name:            "no-deref-symlinks-directory",
					pattern:         "link-c",
					noDerefSymlinks: true,
					items: []string{
						"link-c",
					},
				},
				{
					name:            "deref-symlinks-directory",
					pattern:         "link-c",
					noDerefSymlinks: false,
					items: []string{
						"file-p", // from link-c -> subdir-c
						"file-q", // from link-c -> subdir-c
					},
				},
			},
		},
	}

	for _, topdir := range []string{"", ".", "top"} {
		for _, testArchive := range getTestArchives {
			dir, err := makeContextFromArchive(makeArchive(testArchive.headers, testArchive.contents), topdir)
			require.NoErrorf(t, err, "error creating context from archive %q", testArchive.name)
			defer os.RemoveAll(dir)

			root := dir

			cases := make(map[string]struct{})
			for _, testCase := range testArchive.cases {
				if _, ok := cases[testCase.name]; ok {
					t.Fatalf("duplicate case %q", testCase.name)
				}
				cases[testCase.name] = struct{}{}
			}

			for _, testCase := range testArchive.cases {
				var excludes []string
				for _, exclude := range testCase.exclude {
					excludes = append(excludes, filepath.FromSlash(exclude))
				}

				getOptions := GetOptions{
					Excludes:           excludes,
					ExpandArchives:     testCase.expandArchives,
					StripSetuidBit:     testCase.stripSetuidBit,
					StripSetgidBit:     testCase.stripSetgidBit,
					StripStickyBit:     testCase.stripStickyBit,
					StripXattrs:        testCase.stripXattrs,
					KeepDirectoryNames: testCase.keepDirectoryNames,
					Rename:             testCase.renames,
					NoDerefSymlinks:    testCase.noDerefSymlinks,
				}

				t.Run(fmt.Sprintf("topdir=%s,archive=%s,case=%s,pattern=%s", topdir, testArchive.name, testCase.name, testCase.pattern), func(t *testing.T) {
					// ensure that we can get stuff using this spec
					err := Get(root, topdir, getOptions, []string{testCase.pattern}, ioutil.Discard)
					if err != nil && isExpectedError(err, topdir != "" && topdir != ".", testCase.pattern, testArchive.expectedGetErrors) {
						return
					}
					require.NoErrorf(t, err, "error getting %q under %q", testCase.pattern, filepath.Join(root, topdir))
					// see what we get when we get this pattern
					pipeReader, pipeWriter := io.Pipe()
					var getErr error
					var wg sync.WaitGroup
					wg.Add(1)
					go func() {
						getErr = Get(root, topdir, getOptions, []string{testCase.pattern}, pipeWriter)
						pipeWriter.Close()
						wg.Done()
					}()
					tr := tar.NewReader(pipeReader)
					hdr, err := tr.Next()
					actualContents := []string{}
					for err == nil {
						actualContents = append(actualContents, filepath.FromSlash(hdr.Name))
						hdr, err = tr.Next()
					}
					pipeReader.Close()
					sort.Strings(actualContents)
					// compare it to what we were supposed to get
					expectedContents := make([]string, 0, len(testCase.items))
					for _, item := range testCase.items {
						expectedContents = append(expectedContents, filepath.FromSlash(item))
					}
					sort.Strings(expectedContents)
					assert.Equal(t, io.EOF.Error(), err.Error(), "expected EOF at end of archive, got %q", err.Error())
					wg.Wait()
					assert.NoErrorf(t, getErr, "unexpected error from Get(%q)", testCase.pattern)
					assert.Equal(t, expectedContents, actualContents, "Get(%q,excludes=%v) didn't produce the right set of items", testCase.pattern, excludes)
				})
			}
		}
	}
}

func TestEvalNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testEval(t)
	canChroot = couldChroot
}

func testEval(t *testing.T) {
	tmp, err := ioutil.TempDir("", "copier-test-")
	if err != nil {
		require.NoError(t, err, "error creating temporary directory")
	}
	defer os.RemoveAll(tmp)
	options := EvalOptions{}
	linkname := filepath.Join(tmp, "link")
	vectors := []struct {
		id, linkTarget, inputPath, evaluatedPath string
	}{
		{"0a", "target", "link/foo", "target/foo"},
		{"1a", "/target", "link/foo", "target/foo"},
		{"2a", "../target", "link/foo", "target/foo"},
		{"3a", "/../target", "link/foo", "target/foo"},
		{"4a", "../../target", "link/foo", "target/foo"},
		{"5a", "target/subdirectory", "link/foo", "target/subdirectory/foo"},
		{"6a", "/target/subdirectory", "link/foo", "target/subdirectory/foo"},
		{"7a", "../target/subdirectory", "link/foo", "target/subdirectory/foo"},
		{"8a", "/../target/subdirectory", "link/foo", "target/subdirectory/foo"},
		{"9a", "../../target/subdirectory", "link/foo", "target/subdirectory/foo"},
		// inputPath is lexically cleaned to "foo" early, so callers
		// won't get values consistent with the kernel, but we use the
		// result for ADD and COPY, where docker build seems to have
		// the same limitation
		{"0b", "target", "link/../foo", "foo"},
		{"1b", "/target", "link/../foo", "foo"},
		{"2b", "../target", "link/../foo", "foo"},
		{"3b", "/../target", "link/../foo", "foo"},
		{"4b", "../../target", "link/../foo", "foo"},
		{"5b", "target/subdirectory", "link/../foo", "foo"},
		{"6b", "/target/subdirectory", "link/../foo", "foo"},
		{"7b", "../target/subdirectory", "link/../foo", "foo"},
		{"8b", "/../target/subdirectory", "link/../foo", "foo"},
		{"9b", "../../target/subdirectory", "link/../foo", "foo"},
	}
	for _, vector := range vectors {
		t.Run(fmt.Sprintf("id=%s", vector.id), func(t *testing.T) {
			err = os.Symlink(vector.linkTarget, linkname)
			if err != nil && os.IsExist(err) {
				os.Remove(linkname)
				err = os.Symlink(vector.linkTarget, linkname)
			}
			require.NoErrorf(t, err, "error creating link from %q to %q", linkname, vector.linkTarget)
			evaluated, err := Eval(tmp, filepath.Join(tmp, vector.inputPath), options)
			require.NoErrorf(t, err, "error evaluating %q: %v", vector.inputPath, err)
			require.Equalf(t, filepath.Join(tmp, vector.evaluatedPath), evaluated, "evaluation of %q with %q pointing to %q failed", vector.inputPath, linkname, vector.linkTarget)
		})
	}
}

func TestMkdirNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testMkdir(t)
	canChroot = couldChroot
}

func testMkdir(t *testing.T) {
	type testCase struct {
		name   string
		create string
		expect []string
	}
	testArchives := []struct {
		name      string
		headers   []tar.Header
		testCases []testCase
	}{
		{
			name: "regular",
			headers: []tar.Header{
				{Name: "subdir-a", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-b", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-b/subdir-c", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle1", Typeflag: tar.TypeSymlink, Linkname: "dangle1-target", ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle2", Typeflag: tar.TypeSymlink, Linkname: "../dangle2-target", ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle3", Typeflag: tar.TypeSymlink, Linkname: "../../dangle3-target", ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle4", Typeflag: tar.TypeSymlink, Linkname: "../../../dangle4-target", ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle5", Typeflag: tar.TypeSymlink, Linkname: "../../../../dangle5-target", ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle6", Typeflag: tar.TypeSymlink, Linkname: "/dangle6-target", ModTime: testDate},
				{Name: "subdir-a/subdir-b/dangle7", Typeflag: tar.TypeSymlink, Linkname: "/../dangle7-target", ModTime: testDate},
			},
			testCases: []testCase{
				{
					name:   "basic",
					create: "subdir-d",
					expect: []string{"subdir-d"},
				},
				{
					name:   "subdir",
					create: "subdir-d/subdir-e/subdir-f",
					expect: []string{"subdir-d", "subdir-d/subdir-e", "subdir-d/subdir-e/subdir-f"},
				},
				{
					name:   "dangling-link-itself",
					create: "subdir-a/subdir-b/dangle1",
					expect: []string{"subdir-a/subdir-b/dangle1-target"},
				},
				{
					name:   "dangling-link-as-intermediate-parent",
					create: "subdir-a/subdir-b/dangle2/final",
					expect: []string{"subdir-a/dangle2-target", "subdir-a/dangle2-target/final"},
				},
				{
					name:   "dangling-link-as-intermediate-grandparent",
					create: "subdir-a/subdir-b/dangle3/final",
					expect: []string{"dangle3-target", "dangle3-target/final"},
				},
				{
					name:   "dangling-link-as-intermediate-attempted-relative-breakout",
					create: "subdir-a/subdir-b/dangle4/final",
					expect: []string{"dangle4-target", "dangle4-target/final"},
				},
				{
					name:   "dangling-link-as-intermediate-attempted-relative-breakout-again",
					create: "subdir-a/subdir-b/dangle5/final",
					expect: []string{"dangle5-target", "dangle5-target/final"},
				},
				{
					name:   "dangling-link-itself-absolute",
					create: "subdir-a/subdir-b/dangle6",
					expect: []string{"dangle6-target"},
				},
				{
					name:   "dangling-link-as-intermediate-absolute",
					create: "subdir-a/subdir-b/dangle6/final",
					expect: []string{"dangle6-target", "dangle6-target/final"},
				},
				{
					name:   "dangling-link-as-intermediate-absolute-relative-breakout",
					create: "subdir-a/subdir-b/dangle7/final",
					expect: []string{"dangle7-target", "dangle7-target/final"},
				},
				{
					name:   "parent-parent-final",
					create: "../../final",
					expect: []string{"final"},
				},
				{
					name:   "root-parent-final",
					create: "/../final",
					expect: []string{"final"},
				},
				{
					name:   "root-parent-intermediate-parent-final",
					create: "/../intermediate/../final",
					expect: []string{"final"},
				},
			},
		},
	}
	for i := range testArchives {
		t.Run(testArchives[i].name, func(t *testing.T) {
			for _, testCase := range testArchives[i].testCases {
				t.Run(testCase.name, func(t *testing.T) {
					dir, err := makeContextFromArchive(makeArchive(testArchives[i].headers, nil), "")
					require.NoErrorf(t, err, "error creating context from archive %q, topdir=%q", testArchives[i].name, "")
					defer os.RemoveAll(dir)
					root := dir
					options := MkdirOptions{ChownNew: &idtools.IDPair{UID: os.Getuid(), GID: os.Getgid()}}
					var beforeNames, afterNames []string
					err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if info == nil || err != nil {
							return err
						}
						rel, err := filepath.Rel(dir, path)
						if err != nil {
							return err
						}
						beforeNames = append(beforeNames, rel)
						return nil
					})
					require.NoErrorf(t, err, "error walking directory to catalog pre-Mkdir contents: %v", err)
					err = Mkdir(root, testCase.create, options)
					require.NoErrorf(t, err, "error creating directory %q under %q with Mkdir: %v", testCase.create, root, err)
					err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if info == nil || err != nil {
							return err
						}
						rel, err := filepath.Rel(dir, path)
						if err != nil {
							return err
						}
						afterNames = append(afterNames, rel)
						return nil
					})
					require.NoErrorf(t, err, "error walking directory to catalog post-Mkdir contents: %v", err)
					expected := append([]string{}, beforeNames...)
					for _, expect := range testCase.expect {
						expected = append(expected, filepath.FromSlash(expect))
					}
					sort.Strings(expected)
					sort.Strings(afterNames)
					assert.Equal(t, expected, afterNames, "expected different paths")
				})
			}
		})
	}
}

func TestCleanerSubdirectory(t *testing.T) {
	testCases := [][2]string{
		{".", "."},
		{"..", "."},
		{"/", "."},
		{"directory/subdirectory/..", "directory"},
		{"directory/../..", "."},
		{"../../directory", "directory"},
		{"../directory/subdirectory", "directory/subdirectory"},
		{"/directory/../..", "."},
		{"/directory/../../directory", "directory"},
	}
	for _, testCase := range testCases {
		t.Run(testCase[0], func(t *testing.T) {
			cleaner := cleanerReldirectory(filepath.FromSlash(testCase[0]))
			assert.Equal(t, testCase[1], filepath.ToSlash(cleaner), "expected to get %q, got %q", testCase[1], cleaner)
		})
	}
}

func TestHandleRename(t *testing.T) {
	renames := map[string]string{
		"a":   "b",
		"c":   "d",
		"a/1": "a/2",
	}
	testCases := [][2]string{
		{"a", "b"},
		{"a/1", "a/2"},
		{"a/1/2", "a/2/2"},
		{"a/1/2/3", "a/2/2/3"},
		{"a/2/3/4", "b/2/3/4"},
		{"a/2/3", "b/2/3"},
		{"a/2", "b/2"},
		{"c/2", "d/2"},
	}
	for i, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			renamed := handleRename(renames, testCase[0])
			assert.Equal(t, testCase[1], renamed, "expected to get %q, got %q", testCase[1], renamed)
		})
	}
}

func TestRemoveNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testRemove(t)
	canChroot = couldChroot
}

func testRemove(t *testing.T) {
	type testCase struct {
		name    string
		remove  string
		all     bool
		fail    bool
		removed []string
	}
	testArchives := []struct {
		name      string
		headers   []tar.Header
		testCases []testCase
	}{
		{
			name: "regular",
			headers: []tar.Header{
				{Name: "subdir-a", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/file-a", Typeflag: tar.TypeReg, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/file-b", Typeflag: tar.TypeReg, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-b", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-b/subdir-c", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-b/subdir-c/parent", Typeflag: tar.TypeSymlink, Linkname: "..", ModTime: testDate},
				{Name: "subdir-a/subdir-b/subdir-c/link-b", Typeflag: tar.TypeSymlink, Linkname: "../../file-b", ModTime: testDate},
				{Name: "subdir-a/subdir-b/subdir-c/root", Typeflag: tar.TypeSymlink, Linkname: "/", ModTime: testDate},
				{Name: "subdir-a/subdir-d", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-e", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
				{Name: "subdir-a/subdir-e/subdir-f", Typeflag: tar.TypeDir, Mode: 0755, ModTime: testDate},
			},
			testCases: []testCase{
				{
					name:    "file",
					remove:  "subdir-a/file-a",
					removed: []string{"subdir-a/file-a"},
				},
				{
					name:    "file-all",
					remove:  "subdir-a/file-a",
					all:     true,
					removed: []string{"subdir-a/file-a"},
				},
				{
					name:   "subdir",
					remove: "subdir-a/subdir-b",
					all:    false,
					fail:   true,
				},
				{
					name:   "subdir-all",
					remove: "subdir-a/subdir-b/subdir-c",
					all:    true,
					removed: []string{
						"subdir-a/subdir-b/subdir-c",
						"subdir-a/subdir-b/subdir-c/parent",
						"subdir-a/subdir-b/subdir-c/link-b",
						"subdir-a/subdir-b/subdir-c/root",
					},
				},
				{
					name:    "file-link",
					remove:  "subdir-a/subdir-b/subdir-c/link-b",
					removed: []string{"subdir-a/subdir-b/subdir-c/link-b"},
				},
				{
					name:    "file-link-all",
					remove:  "subdir-a/subdir-b/subdir-c/link-b",
					all:     true,
					removed: []string{"subdir-a/subdir-b/subdir-c/link-b"},
				},
				{
					name:    "file-link-indirect",
					remove:  "subdir-a/subdir-b/subdir-c/parent/subdir-c/link-b",
					removed: []string{"subdir-a/subdir-b/subdir-c/link-b"},
				},
				{
					name:    "file-link-indirect-all",
					remove:  "subdir-a/subdir-b/subdir-c/parent/subdir-c/link-b",
					all:     true,
					removed: []string{"subdir-a/subdir-b/subdir-c/link-b"},
				},
				{
					name:    "dir-link",
					remove:  "subdir-a/subdir-b/subdir-c/root",
					removed: []string{"subdir-a/subdir-b/subdir-c/root"},
				},
				{
					name:    "dir-link-all",
					remove:  "subdir-a/subdir-b/subdir-c/root",
					all:     true,
					removed: []string{"subdir-a/subdir-b/subdir-c/root"},
				},
				{
					name:    "dir-through-link",
					remove:  "subdir-a/subdir-b/subdir-c/root/subdir-a/subdir-d",
					removed: []string{"subdir-a/subdir-d"},
				},
				{
					name:    "dir-through-link-all",
					remove:  "subdir-a/subdir-b/subdir-c/root/subdir-a/subdir-d",
					all:     true,
					removed: []string{"subdir-a/subdir-d"},
				},
				{
					name:   "tree-through-link",
					remove: "subdir-a/subdir-b/subdir-c/root/subdir-a/subdir-e",
					all:    false,
					fail:   true,
				},
				{
					name:    "tree-through-link-all",
					remove:  "subdir-a/subdir-b/subdir-c/root/subdir-a/subdir-e",
					all:     true,
					removed: []string{"subdir-a/subdir-e", "subdir-a/subdir-e/subdir-f"},
				},
			},
		},
	}
	for i := range testArchives {
		t.Run(testArchives[i].name, func(t *testing.T) {
			for _, testCase := range testArchives[i].testCases {
				t.Run(testCase.name, func(t *testing.T) {
					dir, err := makeContextFromArchive(makeArchive(testArchives[i].headers, nil), "")
					require.NoErrorf(t, err, "error creating context from archive %q, topdir=%q", testArchives[i].name, "")
					defer os.RemoveAll(dir)
					root := dir
					options := RemoveOptions{All: testCase.all}
					beforeNames := make(map[string]struct{})
					err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if info == nil || err != nil {
							return err
						}
						rel, err := filepath.Rel(dir, path)
						if err != nil {
							return err
						}
						beforeNames[rel] = struct{}{}
						return nil
					})
					require.NoErrorf(t, err, "error walking directory to catalog pre-Remove contents: %v", err)
					err = Remove(root, testCase.remove, options)
					if testCase.fail {
						require.Errorf(t, err, "did not expect to succeed removing item %q under %q with Remove", testCase.remove, root)
						return
					}
					require.NoErrorf(t, err, "error removing item %q under %q with Remove: %v", testCase.remove, root, err)
					afterNames := make(map[string]struct{})
					err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
						if info == nil || err != nil {
							return err
						}
						rel, err := filepath.Rel(dir, path)
						if err != nil {
							return err
						}
						afterNames[rel] = struct{}{}
						return nil
					})
					require.NoErrorf(t, err, "error walking directory to catalog post-Remove contents: %v", err)
					var removed []string
					for beforeName := range beforeNames {
						if _, stillPresent := afterNames[beforeName]; !stillPresent {
							removed = append(removed, beforeName)
						}
					}
					var expected []string
					for _, expect := range testCase.removed {
						expected = append(expected, filepath.FromSlash(expect))
					}
					sort.Strings(expected)
					sort.Strings(removed)
					assert.Equal(t, expected, removed, "expected different paths to be missing")
				})
			}
		})
	}
}
