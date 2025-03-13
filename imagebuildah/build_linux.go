package imagebuildah

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/containers/buildah/copier"
	"github.com/containers/buildah/define"
	"github.com/containers/buildah/internal/tmpdir"
	"github.com/containers/buildah/pkg/overlay"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/sirupsen/logrus"
	"go.podman.io/storage"
	"go.podman.io/storage/pkg/idtools"
)

// platformSetupContextDirectoryOverlay() either sets up an overlay _over_ the
// build context directory, or creates a temporary copy of it, and sorts out
// labeling.  Returns the location which should be used as the default build
// context; the process label and mount label for the build, if any; a boolean
// value that indicates whether the caller can write directly to the location;
// and a cleanup function which should be called when the location is no longer
// needed (on success). Returned errors should be treated as fatal.
func platformSetupContextDirectoryOverlay(store storage.Store, options *define.BuildOptions) (string, string, string, bool, func(), error) {
	var succeeded bool
	var tmpDir, tmpContextDir, contentDir string
	cleanup := func() {
		if contentDir != "" {
			if err := overlay.CleanupContent(tmpDir); err != nil {
				logrus.Debugf("cleaning up overlay scaffolding for build context directory: %v", err)
			}
		}
		if tmpContextDir != "" {
			if err := os.RemoveAll(tmpContextDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
				logrus.Debugf("removing temporary directory tree %q: %v", tmpContextDir, err)
			}
		}
		if tmpDir != "" {
			if err := os.Remove(tmpDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
				logrus.Debugf("removing should-be-empty temporary directory %q: %v", tmpDir, err)
			}
		}
	}
	defer func() {
		if !succeeded {
			cleanup()
		}
	}()
	// double-check that the context directory location is an absolute path
	contextDirectoryAbsolute, err := filepath.Abs(options.ContextDirectory)
	if err != nil {
		return "", "", "", false, nil, fmt.Errorf("determining absolute path of %q: %w", options.ContextDirectory, err)
	}
	// figure out the labeling situation
	processLabel, mountLabel, err := label.InitLabels(options.CommonBuildOpts.LabelOpts)
	if err != nil {
		return "", "", "", false, nil, err
	}
	// create a temporary directory for whatever we're doing
	tmpDir, err = os.MkdirTemp(tmpdir.GetTempDir(), "buildah-context-")
	if err != nil {
		return "", "", "", false, nil, fmt.Errorf("creating temporary directory: %w", err)
	}
	switch store.GraphDriverName() {
	case "overlay":
		// create the scaffolding for an overlay mount under it
		contentDir, err = overlay.TempDir(tmpDir, 0, 0)
		if err != nil {
			return "", "", "", false, nil, fmt.Errorf("creating overlay scaffolding for build context directory: %w", err)
		}
		// mount an overlay that uses it as a lower
		overlayOptions := overlay.Options{
			GraphOpts:  slices.Clone(store.GraphOptions()),
			ForceMount: true,
			MountLabel: mountLabel,
		}
		targetDir := filepath.Join(contentDir, "target")
		contextDirMountSpec, err := overlay.MountWithOptions(contentDir, contextDirectoryAbsolute, targetDir, &overlayOptions)
		if err != nil {
			return "", "", "", false, nil, fmt.Errorf("creating overlay scaffolding for build context directory: %w", err)
		}
		// going forward, pretend that the merged directory is the actual context directory
		logrus.Debugf("mounted an overlay at %q over %q", contextDirMountSpec.Source, contextDirectoryAbsolute)
		succeeded = true
		return contextDirMountSpec.Source, processLabel, mountLabel, true, cleanup, nil
	default:
		// create a subdirectory under it
		tmpContextDir = filepath.Join(tmpDir, "build-context")
		if err := os.Mkdir(tmpContextDir, 0o755); err != nil {
			return "", "", "", false, nil, fmt.Errorf("creating a temporary directory under %s: %w", tmpDir, err)
		}
		// copy the contents of the default build context to the new location so that it can be written to more or less safely
		getOptions := copier.GetOptions{
			ChownDirs:  &idtools.IDPair{UID: 0, GID: 0},
			ChownFiles: &idtools.IDPair{UID: 0, GID: 0},
		}
		var wg sync.WaitGroup
		var putErr, getErr, labelErr error
		copyReader, copyWriter := io.Pipe()
		wg.Add(1)
		go func() {
			defer wg.Done()
			putErr = copier.Put(tmpContextDir, tmpContextDir, copier.PutOptions{}, copyReader)
			copyReader.Close()
			labelErr = label.Relabel(tmpContextDir, mountLabel, true)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			getErr = copier.Get(contextDirectoryAbsolute, contextDirectoryAbsolute, getOptions, []string{"."}, copyWriter)
			copyWriter.Close()
		}()
		wg.Wait()
		var errs []error
		if getErr != nil {
			errs = append(errs, getErr)
		}
		if putErr != nil {
			errs = append(errs, putErr)
		}
		if labelErr != nil {
			errs = append(errs, labelErr)
		}
		if len(errs) > 0 {
			grouped := errs[0]
			if len(errs) > 1 {
				grouped = errors.Join(errs...)
			}
			return "", "", "", false, nil, fmt.Errorf("creating copy of build context directory: %w", grouped)
		}
		logrus.Debugf("created a copy of %q at %q", contextDirectoryAbsolute, tmpContextDir)
		succeeded = true
		return tmpContextDir, processLabel, mountLabel, true, cleanup, nil
	}
}
