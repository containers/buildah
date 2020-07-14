package copier

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containers/buildah/util"
	"github.com/containers/image/v5/pkg/compression"
	"github.com/containers/storage/pkg/fileutils"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	copierCommand    = "buildah-copier"
	maxLoopsFollowed = 64
	// See http://pubs.opengroup.org/onlinepubs/9699919799/utilities/pax.html#tag_20_92_13_06, from archive/tar
	cISUID = 04000 // Set uid, from archive/tar
	cISGID = 02000 // Set gid, from archive/tar
	cISVTX = 01000 // Save text (sticky bit), from archive/tar
)

func init() {
	reexec.Register(copierCommand, copierMain)
}

// isArchivePath returns true if the specified path can be read like a (possibly
// compressed) tarball.
func isArchivePath(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	rc, _, err := compression.AutoDecompress(f)
	if err != nil {
		return false
	}
	defer rc.Close()
	tr := tar.NewReader(rc)
	_, err = tr.Next()
	return err == nil
}

// RequestType is an implementation detail of the copier package.
type RequestType string

const (
	RequestStat RequestType = "STAT"
	RequestGet  RequestType = "GET"
	RequestPut  RequestType = "PUT"
	RequestQuit RequestType = "QUIT"
)

// Options are set at the start and affect every request.  It is an implementation detail of the copier package.
type Options struct {
	Root      string
	Directory string
	Excludes  []string        `json:",omitempty"`
	UIDMap    []idtools.IDMap `json:",omitempty"`
	GIDMap    []idtools.IDMap `json:",omitempty"`
}

// Request encodes a single request.  It is an implementation detail of the copier package.
type Request struct {
	Request            RequestType
	Root               string // used by all requests
	preservedRoot      string
	rootPrefix         string // used to reconstruct paths being handed back to the caller
	Directory          string // used by all requests
	preservedDirectory string
	Globs              []string `json:",omitempty"` // used by stat, get
	preservedGlobs     []string
	StatOptions        StatOptions `json:",omitempty"`
	GetOptions         GetOptions  `json:",omitempty"`
	PutOptions         PutOptions  `json:",omitempty"`
}

// Response encodes a single response.  It is an implementation detail of the copier package.
type Response struct {
	Stat StatResponse
	Get  GetResponse
	Put  PutResponse
}

// StatResponse encodes a response for a single Stat request.  It is an implementation detail of the copier package.
type StatResponse struct {
	Error string
	Globs []*StatsForGlob
}

// StatsForGlob encode results for a single glob pattern passed to Stat().
type StatsForGlob struct {
	Error   string                  `json:",omitempty"` // error if the Glob pattern was malformed
	Glob    string                  // input pattern to which this result corresponds
	Globbed []string                // a slice of zero or more names that match the glob
	Results map[string]*StatForItem // one for each Globbed value if there are any, or for Glob
}

// StatForItem encode results for a single filesystem item, as returned by Stat().
type StatForItem struct {
	Error           string `json:",omitempty"`
	Name            string
	Size            int64       // dereferenced value for symlinks
	Mode            os.FileMode // dereferenced value for symlinks
	ModTime         time.Time   // dereferenced value for symlinks
	IsSymlink       bool
	IsDir           bool   // dereferenced value for symlinks
	IsRegular       bool   // dereferenced value for symlinks
	IsArchive       bool   // dereferenced value for symlinks
	ImmediateTarget string `json:",omitempty"` // raw link content
}

// GetResponse encodes a response for a single Get request.  It is an implementation detail of the copier package.
type GetResponse struct {
	Error string `json:",omitempty"`
}

// PutResponse encodes a response for a single Put request.  It is an implementation detail of the copier package.
type PutResponse struct {
	Error string `json:",omitempty"`
}

// StatOptions controls parts of Stat()'s behavior.
type StatOptions struct {
	CheckForArchives bool     // check for and populate the IsArchive bit in returned values
	Excludes         []string // contents to pretend don't exist
}

// Stat globs the specified pattern in the specified directory and returns its
// results.
// If root and directory are both not specified, the current root directory is
// used, and relative names in the globs list are treated as being relative to
// the current working directory.
// If root is specified and the current OS supports it, the stat() is performed
// in a chrooted context.  If the directory is specified as an absolute path,
// it should either be the root directory or a subdirectory of the root
// directory.  Otherwise, the directory is treated as a path relative to the
// root directory.
// Relative names in the glob list are treated as being relative to the
// directory.
func Stat(root string, directory string, options StatOptions, globs []string) ([]*StatsForGlob, error) {
	req := Request{
		Request:     RequestStat,
		Root:        root,
		Directory:   directory,
		Globs:       append([]string{}, globs...),
		StatOptions: options,
	}
	resp, err := copier(options.Excludes, nil, nil, req)
	if err != nil {
		return nil, err
	}
	if resp.Stat.Error != "" {
		return nil, errors.New(resp.Stat.Error)
	}
	return resp.Stat.Globs, nil
}

// GetOptions controls parts of Get()'s behavior.
type GetOptions struct {
	UIDMap, GIDMap     []idtools.IDMap // map from hostIDs to containerIDs in the output archive
	Excludes           []string        // contents to pretend don't exist
	ExpandArchives     bool            // extract the contents of named items that are archives
	StripSetidBits     bool            // strip the setuid/setgid/sticky bits off of items being copied. no effect on archives being extracted
	StripXattrs        bool            // don't record extended attributes of items being copied. no effect on archives being extracted
	KeepDirectoryNames bool            // export directories as directories containing items rather than as the items they contain
}

// Get produces an archive containing items that match the specified glob
// patterns and writes it to bulkWriter.
// If root and directory are both not specified, the current root directory is
// used, and relative names in the globs list are treated as being relative to
// the current working directory.
// If root is specified and the current OS supports it, the contents are read
// in a chrooted context.  If the directory is specified as an absolute path,
// it should either be the root directory or a subdirectory of the root
// directory.  Otherwise, the directory is treated as a path relative to the
// root directory.
// Relative names in the glob list are treated as being relative to the
// directory.
func Get(root string, directory string, options GetOptions, globs []string, bulkWriter io.Writer) error {
	req := Request{
		Request:   RequestGet,
		Root:      root,
		Directory: directory,
		Globs:     append([]string{}, globs...),
		StatOptions: StatOptions{
			CheckForArchives: options.ExpandArchives,
		},
		GetOptions: options,
	}
	resp, err := copier(options.Excludes, nil, bulkWriter, req)
	if err != nil {
		return err
	}
	if resp.Get.Error != "" {
		return errors.New(resp.Get.Error)
	}
	return nil
}

// PutOptions controls parts of Put()'s behavior.
type PutOptions struct {
	UIDMap, GIDMap    []idtools.IDMap // map from containerIDs to hostIDs when writing contents to disk
	ChownDirs         *idtools.IDPair // set ownership of newly-created directories
	ChmodDirs         *os.FileMode    // set permissions on newly-created directories
	ChownFiles        *idtools.IDPair // set ownership of newly-created files
	ChmodFiles        *os.FileMode    // set permissions on newly-created files
	StripXattrs       bool            // don't bother trying to set extended attributes of items being copied
	IgnoreXattrErrors bool            // ignore any errors encountered when attempting to set extended attributes
}

// Put extracts an archive from the bulkReader at the specified directory.
// If root and directory are both not specified, the current root directory is
// used.
// If root is specified and the current OS supports it, the contents are written
// in a chrooted context.  If the directory is specified as an absolute path,
// it should either be the root directory or a subdirectory of the root
// directory.  Otherwise, the directory is treated as a path relative to the
// root directory.
func Put(root string, directory string, options PutOptions, bulkReader io.Reader) error {
	req := Request{
		Request:    RequestPut,
		Root:       root,
		Directory:  directory,
		PutOptions: options,
	}
	resp, err := copier(nil, bulkReader, nil, req)
	if err != nil {
		return err
	}
	if resp.Put.Error != "" {
		return errors.New(resp.Put.Error)
	}
	return nil
}

func copier(excludes []string, bulkReader io.Reader, bulkWriter io.Writer, request Request) (*Response, error) {
	if request.Directory == "" {
		if request.Root == "" {
			wd, err := getcwd()
			if err != nil {
				return nil, errors.Wrapf(err, "error getting current working directory")
			}
			request.Directory = wd
		} else {
			request.Directory = request.Root
		}
	}
	if request.Root == "" {
		request.Root = string(os.PathSeparator)
	}
	if filepath.IsAbs(request.Directory) {
		_, err := filepath.Rel(request.Root, request.Directory)
		if err != nil {
			return nil, errors.Wrapf(err, "error rewriting %q to be relative to %q", request.Directory, request.Root)
		}
	}
	if request.Root != string(os.PathSeparator) && canChroot {
		return copierWithSubprocess(excludes, bulkReader, bulkWriter, request)
	}
	return copierWithoutSubprocess(excludes, bulkReader, bulkWriter, request)
}

func copierWithoutSubprocess(excludes []string, bulkReader io.Reader, bulkWriter io.Writer, request Request) (*Response, error) {
	pm, err := fileutils.NewPatternMatcher(excludes)
	if err != nil {
		return nil, errors.Wrapf(err, "error processing excludes list %v", excludes)
	}

	var idMappings *idtools.IDMappings
	var uidMap, gidMap []idtools.IDMap
	switch request.Request {
	case RequestStat:
		break
	case RequestQuit:
		break
	case RequestGet:
		uidMap, gidMap = request.GetOptions.UIDMap, request.GetOptions.GIDMap
	case RequestPut:
		uidMap, gidMap = request.PutOptions.UIDMap, request.PutOptions.GIDMap
	}

	if len(uidMap) > 0 && len(gidMap) > 0 {
		idMappings = idtools.NewIDMappingsFromMaps(uidMap, gidMap)
	}
	request.preservedRoot = request.Root
	request.rootPrefix = string(os.PathSeparator)
	request.preservedDirectory = request.Directory
	request.preservedGlobs = append([]string{}, request.Globs...)
	if !filepath.IsAbs(request.Directory) {
		request.Directory = filepath.Join(request.Root, request.Directory)
	}
	absoluteGlobs := make([]string, 0, len(request.Globs))
	for i, glob := range request.preservedGlobs {
		if filepath.IsAbs(glob) {
			absoluteGlobs = append(absoluteGlobs, request.Globs[i])
		} else {
			absoluteGlobs = append(absoluteGlobs, filepath.Join(request.Directory, request.Globs[i]))
		}
	}
	request.Globs = absoluteGlobs
	response, cb, err := copierHandler(bulkReader, bulkWriter, request, pm, idMappings)
	if err != nil {
		return nil, err
	}
	if cb != nil {
		if err = cb(); err != nil {
			return nil, err
		}
	}
	return response, nil
}

func copierWithSubprocess(excludes []string, bulkReader io.Reader, bulkWriter io.Writer, request Request) (*Response, error) {
	options := Options{
		Root:      request.Root,
		Directory: request.Directory,
		Excludes:  excludes,
	}
	switch request.Request {
	case RequestStat:
		break
	case RequestQuit:
		break
	case RequestGet:
		options.UIDMap, options.GIDMap = request.GetOptions.UIDMap, request.GetOptions.GIDMap
	case RequestPut:
		options.UIDMap, options.GIDMap = request.PutOptions.UIDMap, request.PutOptions.GIDMap
	}
	if bulkReader == nil {
		bulkReader = bytes.NewReader([]byte{})
	}
	if bulkWriter == nil {
		bulkWriter = ioutil.Discard
	}
	cmd := reexec.Command(copierCommand)
	stdinRead, stdinWrite, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrapf(err, "pipe")
	}
	encoder := json.NewEncoder(stdinWrite)
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		stdinRead.Close()
		stdinWrite.Close()
		return nil, errors.Wrapf(err, "pipe")
	}
	decoder := json.NewDecoder(stdoutRead)
	bulkReaderRead, bulkReaderWrite, err := os.Pipe()
	if err != nil {
		stdinRead.Close()
		stdinWrite.Close()
		stdoutRead.Close()
		stdoutWrite.Close()
		return nil, errors.Wrapf(err, "pipe")
	}
	bulkWriterRead, bulkWriterWrite, err := os.Pipe()
	if err != nil {
		stdinRead.Close()
		stdinWrite.Close()
		stdoutRead.Close()
		stdoutWrite.Close()
		bulkReaderRead.Close()
		bulkReaderWrite.Close()
		return nil, errors.Wrapf(err, "pipe")
	}
	cmd.Dir = "/"
	cmd.Env = append([]string{fmt.Sprintf("LOGLEVEL=%d", logrus.GetLevel())}, os.Environ()...)

	errorBuffer := bytes.Buffer{}
	cmd.Stdin = stdinRead
	cmd.Stdout = stdoutWrite
	cmd.Stderr = &errorBuffer
	cmd.ExtraFiles = append(cmd.ExtraFiles, bulkReaderRead, bulkWriterWrite)
	if err = cmd.Start(); err != nil {
		stdinRead.Close()
		stdinWrite.Close()
		stdoutRead.Close()
		stdoutWrite.Close()
		bulkReaderRead.Close()
		bulkReaderWrite.Close()
		bulkWriterRead.Close()
		bulkWriterWrite.Close()
		return nil, errors.Wrapf(err, "error starting subprocess")
	}
	stdinRead.Close()
	stdoutWrite.Close()
	bulkReaderRead.Close()
	bulkWriterWrite.Close()
	if err = encoder.Encode(options); err != nil {
		stdinWrite.Close()
		stdoutRead.Close()
		bulkReaderWrite.Close()
		bulkWriterRead.Close()
		if err2 := cmd.Process.Kill(); err2 != nil {
			return nil, errors.Wrapf(err, "error killing subprocess: %v; error encoding options", err2)
		}
		return nil, errors.Wrapf(err, "error encoding options")
	}
	if err = encoder.Encode(request); err != nil {
		stdinWrite.Close()
		stdoutRead.Close()
		bulkReaderWrite.Close()
		bulkWriterRead.Close()
		if err2 := cmd.Process.Kill(); err2 != nil {
			return nil, errors.Wrapf(err, "error killing subprocess: %v; error encoding request", err2)
		}
		return nil, errors.Wrapf(err, "error encoding request")
	}
	var response Response
	if err = decoder.Decode(&response); err != nil {
		stdinWrite.Close()
		stdoutRead.Close()
		bulkReaderWrite.Close()
		bulkWriterRead.Close()
		if err2 := cmd.Process.Kill(); err2 != nil {
			return nil, errors.Wrapf(err, "error killing subprocess: %v; error decoding response", err2)
		}
		return nil, errors.Wrapf(err, "error decoding response")
	}
	if err = encoder.Encode(&Request{Request: RequestQuit}); err != nil {
		stdinWrite.Close()
		stdoutRead.Close()
		bulkReaderWrite.Close()
		bulkWriterRead.Close()
		if err2 := cmd.Process.Kill(); err2 != nil {
			return nil, errors.Wrapf(err, "error killing subprocess: %v; error encoding request", err2)
		}
		return nil, errors.Wrapf(err, "error encoding request")
	}
	stdinWrite.Close()
	stdoutRead.Close()
	var wg sync.WaitGroup
	var readError, writeError error
	wg.Add(1)
	go func() {
		_, writeError = io.Copy(bulkWriter, bulkWriterRead)
		bulkWriterRead.Close()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		_, readError = io.Copy(bulkReaderWrite, bulkReader)
		bulkReaderWrite.Close()
		wg.Done()
	}()
	wg.Wait()
	if err = cmd.Wait(); err != nil {
		if errorBuffer.String() != "" {
			err = fmt.Errorf("%s", errorBuffer.String())
		}
		return nil, err
	}
	if cmd.ProcessState.Exited() && !cmd.ProcessState.Success() {
		err = fmt.Errorf("subprocess exited with error")
		if errorBuffer.String() != "" {
			err = fmt.Errorf("%s", errorBuffer.String())
		}
		return nil, err
	}
	if readError != nil {
		return nil, errors.Wrapf(readError, "error passing bulk input to subprocess")
	}
	if writeError != nil {
		return nil, errors.Wrapf(writeError, "error passing bulk output from subprocess")
	}
	return &response, nil
}

func copierMain() {
	var options Options
	var idMappings *idtools.IDMappings

	// Set logging.
	if level := os.Getenv("LOGLEVEL"); level != "" {
		if ll, err := strconv.Atoi(level); err == nil {
			logrus.SetLevel(logrus.Level(ll))
		}
	}

	// Unpack our configuration.
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&options); err != nil {
		fmt.Fprintf(os.Stderr, "error decoding options: %v", err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)

	// Set up descriptors for receiving and sending tarstreams.
	bulkReader := os.NewFile(3, "bulk-reader")
	bulkWriter := os.NewFile(4, "bulk-writer")

	// Change to the specified root directory.
	if options.Root == "" {
		options.Root = string(os.PathSeparator)
	}
	chrooted, err := chroot(options.Root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error changing to intended-new-root directory %q: %v", options.Root, err)
		os.Exit(1)
	}

	pm, err := fileutils.NewPatternMatcher(options.Excludes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error handling excludes list %v: %v", options.Excludes, err)
		os.Exit(1)
	}
	if len(options.UIDMap) > 0 && len(options.GIDMap) > 0 {
		idMappings = idtools.NewIDMappingsFromMaps(options.UIDMap, options.GIDMap)
	}

	for {
		// Read a request.
		request := new(Request)
		if err := decoder.Decode(request); err != nil {
			fmt.Fprintf(os.Stderr, "error decoding request: %v", err)
			os.Exit(1)
		}
		if request.Request == RequestQuit {
			// Making Quit a specific request means that we could
			// run Stat() at a caller's behest before using the
			// same process for Get() or Put().  Maybe later.
			break
		}
		// Multiple requests should list the same root, because we
		// can't un-chroot to chroot to some other location.
		if request.Root == "" {
			request.Root = string(os.PathSeparator)
		}
		if request.Root != options.Root {
			fmt.Fprintf(os.Stderr, "request %+v used a different root: %q != %q", request, request.Root, options.Root)
			os.Exit(1)
		}
		request.preservedRoot = request.Root
		request.rootPrefix = string(os.PathSeparator)
		request.preservedDirectory = request.Directory
		request.preservedGlobs = append([]string{}, request.Globs...)
		if chrooted {
			// We'll need to adjust some things now that the root
			// directory isn't what it was.  Make the directory and
			// globs absolute paths for simplicity's sake.
			absoluteDirectory := request.Directory
			if !filepath.IsAbs(request.Directory) {
				absoluteDirectory = filepath.Join(request.Root, request.Directory)
			}
			relativeDirectory, err := filepath.Rel(request.preservedRoot, absoluteDirectory)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error rewriting %q to be relative to %q: %v", absoluteDirectory, request.preservedRoot, err)
				os.Exit(1)
			}
			request.Directory = filepath.Clean(string(os.PathSeparator) + relativeDirectory)
			absoluteGlobs := make([]string, 0, len(request.Globs))
			for i, glob := range request.preservedGlobs {
				if filepath.IsAbs(glob) {
					relativeGlob, err := filepath.Rel(request.preservedRoot, glob)
					if err != nil {
						fmt.Fprintf(os.Stderr, "error rewriting %q to be relative to %q: %v", glob, request.preservedRoot, err)
						os.Exit(1)
					}
					absoluteGlobs = append(absoluteGlobs, filepath.Clean(string(os.PathSeparator)+relativeGlob))
				} else {
					absoluteGlobs = append(absoluteGlobs, filepath.Join(request.Directory, request.Globs[i]))
				}
			}
			request.Globs = absoluteGlobs
			request.rootPrefix = request.Root
			request.Root = string(os.PathSeparator)
		} else {
			// Make the directory and globs absolute paths for
			// simplicity's sake.
			if !filepath.IsAbs(request.Directory) {
				request.Directory = filepath.Join(request.Root, request.Directory)
			}
			absoluteGlobs := make([]string, 0, len(request.Globs))
			for i, glob := range request.preservedGlobs {
				if filepath.IsAbs(glob) {
					absoluteGlobs = append(absoluteGlobs, request.Globs[i])
				} else {
					absoluteGlobs = append(absoluteGlobs, filepath.Join(request.Directory, request.Globs[i]))
				}
			}
			request.Globs = absoluteGlobs
		}
		response, cb, err := copierHandler(bulkReader, bulkWriter, *request, pm, idMappings)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error handling request %#v: %v", *request, err)
			os.Exit(1)
		}
		// Encode the response.
		if err := encoder.Encode(response); err != nil {
			fmt.Fprintf(os.Stderr, "error encoding response %#v: %v", *request, err)
			os.Exit(1)
		}
		// If there's bulk data to transfer, run the callback to either
		// read or write it.
		if cb != nil {
			if err = cb(); err != nil {
				fmt.Fprintf(os.Stderr, "error during bulk transfer for %#v: %v", *request, err)
				os.Exit(1)
			}
		}
	}
}

func copierHandler(bulkReader io.Reader, bulkWriter io.Writer, request Request, pm *fileutils.PatternMatcher, idMappings *idtools.IDMappings) (*Response, func() error, error) {
	switch request.Request {
	case RequestStat:
		resp := copierHandlerStat(request, pm)
		return resp, nil, nil
	case RequestGet:
		return copierHandlerGet(bulkWriter, request, pm, idMappings)
	case RequestPut:
		return copierHandlerPut(bulkReader, request, idMappings)
	case RequestQuit:
		return nil, nil, nil
	}
	return nil, nil, errors.Errorf("unrecognized copier request %q", request.Request)
}

// pathIsExcluded computes path relative to root, then asks the pattern matcher
// if the result is excluded.  Returns the relative path and the matcher's
// results.
func pathIsExcluded(root, path string, pm *fileutils.PatternMatcher) (string, bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false, errors.Wrapf(err, "copier: error computing path of %q relative to root %q", path, root)
	}
	if pm == nil {
		return rel, false, nil
	}
	if rel == "." {
		// special case
		return rel, false, nil
	}
	matches, err := pm.Matches(rel) // nolint:staticcheck
	if err != nil {
		return rel, false, errors.Wrapf(err, "copier: error checking if %q is excluded", rel)
	}
	if matches {
		return rel, true, nil
	}
	return rel, false, nil
}

// resolvePath resolves symbolic links in paths, treating the specified
// directory as the root.
// Resolving the path this way, and using the result, is in no way secure
// against an active party messing with things under us, and it is not expected
// to be.
// This helps us approximate chrooted behavior on systems and in test cases
// where chroot isn't available.
func resolvePath(root, path string, pm *fileutils.PatternMatcher) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", errors.Errorf("error making path %q relative to %q", path, root)
	}
	workingPath := root
	followed := 0
	components := strings.Split(rel, string(os.PathSeparator))
	excluded := false
	for len(components) > 0 {
		// if anything we try to examine is excluded, then resolution has to "break"
		_, thisExcluded, err := pathIsExcluded(root, filepath.Join(workingPath, components[0]), pm)
		if err != nil {
			return "", err
		}
		excluded = excluded || thisExcluded
		if !excluded {
			if target, err := os.Readlink(filepath.Join(workingPath, components[0])); err == nil {
				followed++
				if followed > maxLoopsFollowed {
					return "", &os.PathError{
						Op:   "open",
						Path: path,
						Err:  err,
					}
				}
				if filepath.IsAbs(target) {
					// symlink to an absolute path - prepend the
					// root directory to that absolute path to
					// replace the current location, and resolve
					// the remaining components
					workingPath = root
					components = append(strings.Split(target, string(os.PathSeparator)), components[1:]...)
					continue
				}
				// symlink to a relative path - add the link target to
				// the current location to get the next location, and
				// resolve the remaining components
				rel, err := filepath.Rel(root, filepath.Join(workingPath, target))
				if err != nil {
					return "", errors.Errorf("error making path %q relative to %q", filepath.Join(workingPath, target), root)
				}
				workingPath = root
				components = append(strings.Split(filepath.Clean(string(os.PathSeparator)+rel), string(os.PathSeparator)), components[1:]...)
				continue
			}
		}
		// append the current component's name to get the next location
		workingPath = filepath.Join(workingPath, components[0])
		if workingPath == filepath.Join(root, "..") {
			// attempted to go above the root using a relative path .., scope it
			workingPath = root
		}
		// ready to handle the next component
		components = components[1:]
	}
	return workingPath, nil
}

func copierHandlerStat(request Request, pm *fileutils.PatternMatcher) *Response {
	errorResponse := func(fmtspec string, args ...interface{}) *Response {
		return &Response{Stat: StatResponse{Error: fmt.Sprintf(fmtspec, args...)}}
	}
	if len(request.Globs) == 0 {
		return errorResponse("copier: stat: expected at least one glob pattern, got none")
	}
	var stats []*StatsForGlob
	for i, glob := range request.Globs {
		s := StatsForGlob{
			Glob: request.preservedGlobs[i],
		}
		stats = append(stats, &s)
		// glob this pattern
		globMatched, err := filepath.Glob(glob)
		if err != nil {
			s.Error = fmt.Sprintf("copier: stat: %q while matching glob pattern %q", err.Error(), glob)
			continue
		}
		// collect the matches
		s.Globbed = make([]string, 0, len(globMatched))
		s.Results = make(map[string]*StatForItem)
		for _, globbed := range globMatched {
			rel, excluded, err := pathIsExcluded(request.Root, globbed, pm)
			if err != nil {
				return errorResponse("copier: stat: %v", err)
			}
			if excluded {
				continue
			}
			// if the glob was an absolute path, reconstruct the
			// path that we should hand back for the match
			var resultName string
			if filepath.IsAbs(request.preservedGlobs[i]) {
				resultName = filepath.Join(request.rootPrefix, globbed)
			} else {
				relResult := rel
				if request.Directory != request.Root {
					relResult, err = filepath.Rel(request.Directory, globbed)
					if err != nil {
						return errorResponse("copier: stat: error making %q relative to %q: %v", globbed, request.Directory, err)
					}
				}
				resultName = relResult
			}
			result := StatForItem{Name: resultName}
			s.Globbed = append(s.Globbed, resultName)
			s.Results[resultName] = &result
			// lstat the matched value
			linfo, err := os.Lstat(globbed)
			if err != nil {
				result.Error = err.Error()
				continue
			}
			result.Size = linfo.Size()
			result.Mode = linfo.Mode()
			result.ModTime = linfo.ModTime()
			result.IsDir = linfo.IsDir()
			result.IsRegular = result.Mode.IsRegular()
			result.IsSymlink = (linfo.Mode() & os.ModeType) == os.ModeSymlink
			checkForArchive := request.StatOptions.CheckForArchives
			if result.IsSymlink {
				// if the match was a symbolic link, read it
				immediateTarget, err := os.Readlink(globbed)
				if err != nil {
					result.Error = err.Error()
					continue
				}
				// record where it points, both by itself (it
				// could be a relative link) and in the context
				// of the chroot
				result.ImmediateTarget = immediateTarget
				resolvedTarget, err := resolvePath(request.Root, globbed, pm)
				if err != nil {
					return errorResponse("copier: stat: error resolving %q: %v", globbed, err)
				}
				// lstat the thing that we point to
				info, err := os.Lstat(resolvedTarget)
				if err != nil {
					result.Error = err.Error()
					continue
				}
				// replace IsArchive/IsDir/IsRegular with info about the target
				if info.Mode().IsRegular() && request.StatOptions.CheckForArchives {
					result.IsArchive = isArchivePath(resolvedTarget)
					checkForArchive = false
				}
				result.IsDir = info.IsDir()
				result.IsRegular = info.Mode().IsRegular()
			}
			if result.IsRegular && checkForArchive {
				// we were asked to check on this, and it
				// wasn't a symlink, in which case we'd have
				// already checked what the link points to
				result.IsArchive = isArchivePath(globbed)
			}
		}
		// no unskipped matches -> error
		if len(s.Globbed) == 0 {
			s.Globbed = nil
			s.Results = nil
			s.Error = fmt.Sprintf("copier: stat: %q: %v", glob, syscall.ENOENT)
		}
	}
	return &Response{Stat: StatResponse{Globs: stats}}
}

func copierHandlerGet(bulkWriter io.Writer, request Request, pm *fileutils.PatternMatcher, idMappings *idtools.IDMappings) (*Response, func() error, error) {
	statRequest := request
	statRequest.Request = RequestStat
	statResponse := copierHandlerStat(request, pm)
	errorResponse := func(fmtspec string, args ...interface{}) (*Response, func() error, error) {
		return &Response{Stat: statResponse.Stat, Get: GetResponse{Error: fmt.Sprintf(fmtspec, args...)}}, nil, nil
	}
	if statResponse.Stat.Error != "" {
		return errorResponse("%s", statResponse.Stat.Error)
	}
	if len(request.Globs) == 0 {
		return errorResponse("copier: get: expected at least one glob pattern, got 0")
	}
	// build a queue of items by globbing
	var queue []string
	globMatchedCount := 0
	for _, glob := range request.Globs {
		globMatched, err := filepath.Glob(glob)
		if err != nil {
			return errorResponse("copier: get: glob %q: %v", glob, err)
		}
		globMatchedCount += len(globMatched)
		filtered := make([]string, 0, len(globMatched))
		for _, globbed := range globMatched {
			rel, excluded, err := pathIsExcluded(request.Root, globbed, pm)
			if err != nil {
				return errorResponse("copier: get: checking if %q is excluded: %v", globbed, err)
			}
			if rel == "." || !excluded {
				filtered = append(filtered, globbed)
			}
		}
		if len(filtered) == 0 {
			return errorResponse("copier: get: glob %q matched nothing (%d filtered out of %v): %v", glob, len(globMatched), globMatched, syscall.ENOENT)
		}
		queue = append(queue, filtered...)
	}
	// no matches -> error
	if len(queue) == 0 {
		return errorResponse("copier: get: globs %v matched nothing (%d filtered out): %v", request.Globs, globMatchedCount, syscall.ENOENT)
	}
	cb := func() error {
		tw := tar.NewWriter(bulkWriter)
		defer tw.Close()
		hardlinkChecker := new(util.HardlinkChecker)
		itemsCopied := 0
		for i, item := range queue {
			// if we're not discarding the names of individual directories, keep track of this one
			relNamePrefix := ""
			if request.GetOptions.KeepDirectoryNames {
				relNamePrefix = filepath.Base(item)
			}
			// if the named thing-to-read is a symlink, dereference it
			info, err := os.Lstat(item)
			if err != nil {
				return errors.Wrapf(err, "copier: get: lstat %q", item)
			}
			// chase links. if we hit a dead end, we should just fail
			followedLinks := 0
			const maxFollowedLinks = 16
			for info.Mode()&os.ModeType == os.ModeSymlink && followedLinks < maxFollowedLinks {
				path, err := os.Readlink(item)
				if err != nil {
					continue
				}
				if filepath.IsAbs(path) {
					path = filepath.Join(request.Root, path)
				} else {
					path = filepath.Join(filepath.Dir(item), path)
				}
				item = path
				if _, err = filepath.Rel(request.Root, item); err != nil {
					return errors.Wrapf(err, "copier: get: computing path of %q(%q) relative to %q", queue[i], item, request.Root)
				}
				if info, err = os.Lstat(item); err != nil {
					return errors.Wrapf(err, "copier: get: lstat %q(%q)", queue[i], item)
				}
				followedLinks++
			}
			if followedLinks >= maxFollowedLinks {
				return errors.Wrapf(syscall.ELOOP, "copier: get: resolving symlink %q(%q)", queue[i], item)
			}
			// evaluate excludes relative to the root directory
			if info.Mode().IsDir() {
				walkfn := func(path string, info os.FileInfo, err error) error {
					// compute the path of this item
					// relative to the top-level directory,
					// for the tar header
					rel, relErr := filepath.Rel(item, path)
					if relErr != nil {
						return errors.Wrapf(relErr, "copier: get: error computing path of %q relative to top directory %q", path, item)
					}
					if err != nil {
						return errors.Wrapf(err, "copier: get: error reading %q", path)
					}
					// prefix the original item's name if we're keeping it
					if relNamePrefix != "" {
						rel = filepath.Join(relNamePrefix, rel)
					}
					if rel == "" || rel == "." {
						// skip the "." entry
						return nil
					}
					_, skip, err := pathIsExcluded(request.Root, path, pm)
					if err != nil {
						return err
					}
					if skip {
						// don't use filepath.SkipDir
						// here, since a more specific
						// but-include-this for
						// something under it might
						// also be in the excludes list
						return nil
					}
					// if it's a symlink, read its target
					symlinkTarget := ""
					if info.Mode()&os.ModeType == os.ModeSymlink {
						target, err := os.Readlink(path)
						if err != nil {
							return errors.Wrapf(err, "copier: get: readlink(%q(%q))", rel, path)
						}
						symlinkTarget = target
					}
					// add the item to the outgoing tar stream
					return copierHandlerGetOne(info, symlinkTarget, rel, path, request.GetOptions, tw, hardlinkChecker, idMappings)
				}
				// walk the directory tree, checking/adding items individually
				if err := filepath.Walk(item, walkfn); err != nil {
					return errors.Wrapf(err, "copier: get: %q(%q)", queue[i], item)
				}
				itemsCopied++
			} else {
				_, skip, err := pathIsExcluded(request.Root, item, pm)
				if err != nil {
					return err
				}
				if skip {
					return nil
				}
				// add the item to the outgoing tar stream.  in
				// cases where this was a symlink that we
				// dereferenced, be sure to use the name of the
				// link.
				if err := copierHandlerGetOne(info, "", filepath.Base(queue[i]), item, request.GetOptions, tw, hardlinkChecker, idMappings); err != nil {
					return errors.Wrapf(err, "copier: get: %q", queue[i])
				}
				itemsCopied++
			}
		}
		if itemsCopied == 0 {
			return errors.New("copier: get: copied no items")
		}
		return nil
	}
	return &Response{Stat: statResponse.Stat, Get: GetResponse{Error: ""}}, cb, nil
}

func copierHandlerGetOne(srcfi os.FileInfo, symlinkTarget, name, contentPath string, options GetOptions, tw *tar.Writer, hardlinkChecker *util.HardlinkChecker, idMappings *idtools.IDMappings) error {
	// build the header using the name provided
	hdr, err := tar.FileInfoHeader(srcfi, symlinkTarget)
	if err != nil {
		return errors.Wrapf(err, "error generating tar header for %s (%s)", contentPath, symlinkTarget)
	}
	if name != "" {
		hdr.Name = name
	}
	if options.StripSetidBits {
		hdr.Mode &^= (cISUID | cISGID | cISVTX)
	}
	// read extended attributes
	var xattrs map[string]string
	if !options.StripXattrs {
		xattrs, err = Lgetxattrs(contentPath)
		if err != nil {
			return errors.Wrapf(err, "error getting extended attributes for %q", contentPath)
		}
	}
	hdr.Xattrs = xattrs // nolint:staticcheck
	if hdr.Typeflag == tar.TypeReg {
		// if it's an archive and we're extracting archives, read the
		// file and spool out its contents in-line.  (if we just
		// inlined the whole file, we'd also be inlining the EOF marker
		// it contains)
		if options.ExpandArchives && isArchivePath(contentPath) {
			f, err := os.Open(contentPath)
			if err != nil {
				return errors.Wrapf(err, "error opening %s", contentPath)
			}
			defer f.Close()
			rc, _, err := compression.AutoDecompress(f)
			if err != nil {
				return errors.Wrapf(err, "error decompressing %s", contentPath)
			}
			defer rc.Close()
			tr := tar.NewReader(rc)
			hdr, err := tr.Next()
			for err == nil {
				if err = tw.WriteHeader(hdr); err != nil {
					return errors.Wrapf(err, "error writing tar header from %q to pipe", contentPath)
				}
				if hdr.Size != 0 {
					n, err := io.Copy(tw, tr)
					if err != nil {
						return errors.Wrapf(err, "error extracting content from archive %s: %s", contentPath, hdr.Name)
					}
					if n != hdr.Size {
						return errors.Errorf("error extracting contents of archive %s: incorrect length for %q", contentPath, hdr.Name)
					}
					tw.Flush()
				}
				hdr, err = tr.Next()
			}
			if err != io.EOF {
				return errors.Wrapf(err, "error extracting contents of archive %s", contentPath)
			}
			return nil
		}
		// if this regular file is hard linked to something else we've
		// already added, set up to output a TypeLink entry instead of
		// a TypeReg entry
		target := hardlinkChecker.Check(srcfi)
		if target != "" {
			hdr.Typeflag = tar.TypeLink
			if filepath.Dir(filepath.Clean(string(os.PathSeparator)+name)) == filepath.Dir(target) {
				target = filepath.Base(target)
			}
			hdr.Linkname = target
			hdr.Size = 0
		} else {
			// note the device/inode pair for this file
			hardlinkChecker.Add(srcfi, string(os.PathSeparator)+name)
		}
	}
	// map the ownership for the archive
	if idMappings != nil && !idMappings.Empty() {
		hostPair := idtools.IDPair{UID: hdr.Uid, GID: hdr.Gid}
		hdr.Uid, hdr.Gid, err = idMappings.ToContainer(hostPair)
		if err != nil {
			return errors.Wrapf(err, "error mapping host filesystem owners %#v to container filesystem owners", hostPair)
		}
	}
	// output the header
	if err = tw.WriteHeader(hdr); err != nil {
		return errors.Wrapf(err, "error writing header for %s (%s)", contentPath, hdr.Name)
	}
	if hdr.Typeflag == tar.TypeReg {
		// output the content
		f, err := os.Open(contentPath)
		if err != nil {
			return errors.Wrapf(err, "error opening %s", contentPath)
		}
		defer f.Close()
		n, err := io.Copy(tw, f)
		if err != nil {
			return errors.Wrapf(err, "error copying %s", contentPath)
		}
		if n != hdr.Size {
			return errors.Errorf("error copying %s: incorrect size (expected %d bytes, read %d bytes)", contentPath, n, hdr.Size)
		}
		tw.Flush()
	}
	return nil
}

func copierHandlerPut(bulkReader io.Reader, request Request, idMappings *idtools.IDMappings) (*Response, func() error, error) {
	errorResponse := func(fmtspec string, args ...interface{}) (*Response, func() error, error) {
		return &Response{Put: PutResponse{Error: fmt.Sprintf(fmtspec, args...)}}, nil, nil
	}
	dirUID, dirGID := 0, 0
	if request.PutOptions.ChownDirs != nil {
		dirUID, dirGID = request.PutOptions.ChownDirs.UID, request.PutOptions.ChownDirs.GID
	}
	dirMode := os.FileMode(0755)
	if request.PutOptions.ChmodDirs != nil {
		dirMode = *request.PutOptions.ChmodDirs
	}
	var fileUID, fileGID *int
	if request.PutOptions.ChownFiles != nil {
		fileUID, fileGID = &request.PutOptions.ChownFiles.UID, &request.PutOptions.ChownFiles.GID
	}
	if idMappings != nil && !idMappings.Empty() {
		containerDirPair := idtools.IDPair{UID: dirUID, GID: dirGID}
		hostDirPair, err := idMappings.ToHost(containerDirPair)
		if err != nil {
			return errorResponse("copier: put: error mapping container filesystem owner %d:%d to host filesystem owners: %v", dirUID, dirGID, err)
		}
		dirUID, dirGID = hostDirPair.UID, hostDirPair.GID
		if request.PutOptions.ChownFiles != nil {
			containerFilePair := idtools.IDPair{UID: *fileUID, GID: *fileGID}
			hostFilePair, err := idMappings.ToHost(containerFilePair)
			if err != nil {
				return errorResponse("copier: put: error mapping container filesystem owner %d:%d to host filesystem owners: %v", fileUID, fileGID, err)
			}
			fileUID, fileGID = &hostFilePair.UID, &hostFilePair.GID
		}
	}
	ensureDirectoryUnderRoot := func(directory string) error {
		rel, err := filepath.Rel(request.Root, directory)
		if err != nil {
			return errors.Wrapf(err, "%q is not a subdirectory of %q", directory, request.Root)
		}
		subdir := ""
		for _, component := range strings.Split(rel, string(os.PathSeparator)) {
			subdir = filepath.Join(subdir, component)
			path := filepath.Join(request.Root, subdir)
			if err := os.Mkdir(path, dirMode); err == nil {
				err = os.Chown(path, dirUID, dirGID)
				if err != nil {
					return errors.Wrapf(err, "copier: put: error setting owner of %q to %d:%d", path, dirUID, dirGID)
				}
			} else {
				if !os.IsExist(err) {
					return errors.Wrapf(err, "copier: put: error checking directory %q", path)
				}
			}
		}
		return nil
	}
	createFile := func(path string, tr *tar.Reader, mode os.FileMode) (int64, error) {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_EXCL, mode)
		if err != nil && os.IsExist(err) {
			if err = os.Remove(path); err != nil {
				return 0, errors.Wrapf(err, "copier: put: error removing file to be overwritten %q", path)
			}
			f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_EXCL, mode)
		}
		if err != nil {
			return 0, errors.Wrapf(err, "copier: put: error opening file %q for writing", path)
		}
		defer f.Close()
		n, err := io.Copy(f, tr)
		if err != nil {
			return n, errors.Wrapf(err, "copier: put: error writing file %q", path)
		}
		if err = f.Chmod(mode); err != nil {
			return n, errors.Wrapf(err, "copier: put: error setting permissions on file %q", path)
		}
		return n, nil
	}
	targetDirectory, err := resolvePath(request.Root, request.Directory, nil)
	if err != nil {
		return errorResponse("copier: put: error resolving %q: %v", request.Directory, err)
	}
	info, err := os.Lstat(targetDirectory)
	if err == nil {
		if !info.IsDir() {
			return errorResponse("copier: put: %s (%s): exists but is not a directory", request.Directory, targetDirectory)
		}
	} else {
		if !os.IsNotExist(err) {
			return errorResponse("copier: put: %s: %v", request.Directory, err)
		}
		if err := ensureDirectoryUnderRoot(request.Directory); err != nil {
			return errorResponse("copier: put: %v", err)
		}
	}
	cb := func() error {
		type directoryAndTimes struct {
			directory    string
			atime, mtime time.Time
		}
		var directoriesAndTimes []directoryAndTimes
		defer func() {
			for i := range directoriesAndTimes {
				directoryAndTimes := directoriesAndTimes[len(directoriesAndTimes)-i-1]
				if err := lutimes(false, directoryAndTimes.directory, directoryAndTimes.atime, directoryAndTimes.mtime); err != nil {
					logrus.Debugf("error setting access and modify timestamps on %q to %s and %s: %v", directoryAndTimes.directory, directoryAndTimes.atime, directoryAndTimes.mtime, err)
				}
			}
		}()
		tr := tar.NewReader(bulkReader)
		hdr, err := tr.Next()
		for err == nil {
			// figure out who should own this new item
			if idMappings != nil && !idMappings.Empty() {
				containerPair := idtools.IDPair{UID: hdr.Uid, GID: hdr.Gid}
				hostPair, err := idMappings.ToHost(containerPair)
				if err != nil {
					return errors.Wrapf(err, "error mapping container filesystem owner 0,0 to host filesystem owners")
				}
				hdr.Uid, hdr.Gid = hostPair.UID, hostPair.GID
			}
			if hdr.Typeflag == tar.TypeDir {
				if request.PutOptions.ChownDirs != nil {
					hdr.Uid, hdr.Gid = dirUID, dirGID
				}
			} else {
				if request.PutOptions.ChownFiles != nil {
					hdr.Uid, hdr.Gid = *fileUID, *fileGID
				}
			}
			// make sure the parent directory exists
			path := filepath.Join(targetDirectory, string(os.PathSeparator)+hdr.Name)
			if err := ensureDirectoryUnderRoot(filepath.Dir(path)); err != nil {
				return err
			}
			// figure out what the permissions should be
			mode := os.FileMode(hdr.Mode) & os.ModePerm
			if hdr.Typeflag == tar.TypeDir {
				if request.PutOptions.ChmodDirs != nil {
					hdr.Mode = int64(*request.PutOptions.ChmodDirs)
				}
			} else {
				if request.PutOptions.ChmodFiles != nil {
					hdr.Mode = int64(*request.PutOptions.ChmodFiles)
				}
			}
			// create the new item
			devMajor := uint32(hdr.Devmajor)
			devMinor := uint32(hdr.Devminor)
			written := hdr.Size
			switch hdr.Typeflag {
			// no type flag for sockets
			case tar.TypeReg, tar.TypeRegA:
				written, err = createFile(path, tr, mode)
			case tar.TypeLink:
				var linkTarget string
				if filepath.IsAbs(hdr.Linkname) {
					linkTarget, err = resolvePath(targetDirectory, filepath.Join(request.Root, hdr.Linkname), nil)
					if err != nil {
						return errors.Errorf("error resolving hardlink target path %q under root %q", hdr.Linkname, request.Root)
					}
				} else {
					linkTarget, err = resolvePath(targetDirectory, filepath.Join(targetDirectory, filepath.Dir(hdr.Name), hdr.Linkname), nil)
					if err != nil {
						return errors.Errorf("error resolving hardlink target path %q under root %q in directory %q", hdr.Linkname, request.Root, filepath.Dir(hdr.Name))
					}
				}
				err = os.Link(linkTarget, path)
				if err != nil && os.IsExist(err) {
					if err = os.Remove(path); err == nil {
						err = os.Link(linkTarget, path)
					}
				}
			case tar.TypeSymlink:
				err = os.Symlink(hdr.Linkname, path)
				if err != nil && os.IsExist(err) {
					if err = os.Remove(path); err == nil {
						err = os.Symlink(hdr.Linkname, path)
					}
				}
			case tar.TypeChar:
				err = mknod(path, chrMode(mode), int(mkdev(devMajor, devMinor)))
				if err != nil && os.IsExist(err) {
					if err = os.Remove(path); err == nil {
						err = mknod(path, chrMode(mode), int(mkdev(devMajor, devMinor)))
					}
				}
			case tar.TypeBlock:
				err = mknod(path, blkMode(mode), int(mkdev(devMajor, devMinor)))
				if err != nil && os.IsExist(err) {
					if err = os.Remove(path); err == nil {
						err = mknod(path, blkMode(mode), int(mkdev(devMajor, devMinor)))
					}
				}
			case tar.TypeDir:
				err = os.Mkdir(path, mode)
				if err != nil && os.IsExist(err) {
					err = nil
				}
				// make a note of the directory's times.  we
				// might create items under it, which will
				// cause the mtime to change after we correct
				// it, so we'll need to correct it again later
				directoriesAndTimes = append(directoriesAndTimes, directoryAndTimes{
					directory: path,
					atime:     hdr.AccessTime,
					mtime:     hdr.ModTime,
				})
			case tar.TypeFifo:
				fifoMode := uint32(hdr.Mode)
				err = mkfifo(path, fifoMode)
				if err != nil && os.IsExist(err) {
					if err = os.Remove(path); err == nil {
						err = mkfifo(path, fifoMode)
					}
				}
			}
			// check for errors
			if err != nil {
				return errors.Wrapf(err, "error creating %q", path)
			}
			if written != hdr.Size {
				return errors.Errorf("error creating %q: incorrect length (%d != %d)", path, written, hdr.Size)
			}
			// restore xattrs
			if !request.PutOptions.StripXattrs {
				if err = Lsetxattrs(path, hdr.Xattrs); err != nil { // nolint:staticcheck
					if !request.PutOptions.IgnoreXattrErrors {
						return errors.Wrapf(err, "error setting extended attributes on %q", path)
					}
				}
			}
			// restore permissions, except for symlinks, since we don't have lchmod
			if hdr.Typeflag != tar.TypeSymlink {
				if err = os.Chmod(path, mode); err != nil {
					return errors.Wrapf(err, "error setting permissions on %q to 0%o", path, mode)
				}
			}
			// set ownership
			if err = os.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
				return errors.Wrapf(err, "error setting ownership of %q to %d:%d", path, hdr.Uid, hdr.Gid)
			}
			// set other bits that might have been reset by chown()
			if hdr.Typeflag != tar.TypeSymlink {
				if hdr.Mode&cISUID == cISUID {
					mode |= syscall.S_ISUID
				}
				if hdr.Mode&cISGID == cISGID {
					mode |= syscall.S_ISGID
				}
				if hdr.Mode&cISVTX == cISVTX {
					mode |= syscall.S_ISVTX
				}
				if err = syscall.Chmod(path, uint32(mode)); err != nil {
					return errors.Wrapf(err, "error setting additional permissions on %q to 0%o", path, mode)
				}
			}
			// set time
			if hdr.AccessTime.IsZero() || hdr.AccessTime.Before(hdr.ModTime) {
				hdr.AccessTime = hdr.ModTime
			}
			if err = lutimes(hdr.Typeflag == tar.TypeSymlink, path, hdr.AccessTime, hdr.ModTime); err != nil {
				return errors.Wrapf(err, "error setting access and modify timestamps on %q to %s and %s", path, hdr.AccessTime, hdr.ModTime)
			}
			hdr, err = tr.Next()
		}
		if err != io.EOF {
			return errors.Wrapf(err, "error reading tar stream: expected EOF")
		}
		return nil
	}
	return &Response{Put: PutResponse{Error: ""}}, cb, nil
}
