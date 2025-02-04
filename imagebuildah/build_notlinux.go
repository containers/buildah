//go:build !linux

package imagebuildah

import (
	"github.com/containers/buildah/define"
	"github.com/containers/storage"
)

// platformSetupContextDirectoryOverlay() may set up an overlay _over_ the
// build context directory, and sorts out labeling.  Returns either the new
// location which should be used as the base build context or the old location;
// the process label and mount label for the build; a boolean value that
// indicates whether we did, in fact, mount an overlay; a cleanup function
// which should be called when the location is no longer needed (on success);
// and a non-nil fatal error if any of that failed.
func platformSetupContextDirectoryOverlay(store storage.Store, options *define.BuildOptions) (string, string, string, bool, func(), error) {
	return options.ContextDirectory, "", "", false, func() {}, nil
}
