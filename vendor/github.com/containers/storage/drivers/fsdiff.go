package graphdriver

import (
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
)

var (
	// ApplyUncompressedLayer defines the unpack method used by the graph
	// driver.
	ApplyUncompressedLayer = chrootarchive.ApplyUncompressedLayer
)

// NaiveDiffDriver takes a ProtoDriver and adds the
// capability of the Diffing methods which it may or may not
// support on its own. See the comment on the exported
// NewNaiveDiffDriver function below.
// Notably, the AUFS driver doesn't need to be wrapped like this.
type NaiveDiffDriver struct {
	ProtoDriver
	uidMaps []idtools.IDMap
	gidMaps []idtools.IDMap
}

// NewNaiveDiffDriver returns a fully functional driver that wraps the
// given ProtoDriver and adds the capability of the following methods which
// it may or may not support on its own:
//     Diff(id, parent string) (archive.Archive, error)
//     Changes(id, parent string) ([]archive.Change, error)
//     ApplyDiff(id, parent string, diff archive.Reader) (size int64, err error)
//     DiffSize(id, parent string) (size int64, err error)
func NewNaiveDiffDriver(driver ProtoDriver, uidMaps, gidMaps []idtools.IDMap) Driver {
	gdw := &NaiveDiffDriver{
		ProtoDriver: driver,
		uidMaps:     uidMaps,
		gidMaps:     gidMaps,
	}
	return gdw
}

// Diff produces an archive of the changes between the specified
// layer and its parent layer which may be "".
func (gdw *NaiveDiffDriver) Diff(id, parent string) (arch archive.Archive, err error) {
	layerFs, err := gdw.Get(id, "")
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			gdw.Put(id)
		}
	}()

	if parent == "" {
		archive, err := archive.Tar(layerFs, archive.Uncompressed)
		if err != nil {
			return nil, err
		}
		return ioutils.NewReadCloserWrapper(archive, func() error {
			err := archive.Close()
			gdw.Put(id)
			return err
		}), nil
	}

	parentFs, err := gdw.Get(parent, "")
	if err != nil {
		return nil, err
	}
	defer gdw.Put(parent)

	changes, err := archive.ChangesDirs(layerFs, parentFs)
	if err != nil {
		return nil, err
	}

	archive, err := archive.ExportChanges(layerFs, changes, gdw.uidMaps, gdw.gidMaps)
	if err != nil {
		return nil, err
	}

	return ioutils.NewReadCloserWrapper(archive, func() error {
		err := archive.Close()
		gdw.Put(id)
		return err
	}), nil
}

// Changes produces a list of changes between the specified layer
// and its parent layer. If parent is "", then all changes will be ADD changes.
func (gdw *NaiveDiffDriver) Changes(id, parent string) ([]archive.Change, error) {
	layerFs, err := gdw.Get(id, "")
	if err != nil {
		return nil, err
	}
	defer gdw.Put(id)

	parentFs := ""

	if parent != "" {
		parentFs, err = gdw.Get(parent, "")
		if err != nil {
			return nil, err
		}
		defer gdw.Put(parent)
	}

	return archive.ChangesDirs(layerFs, parentFs)
}

// ApplyDiff extracts the changeset from the given diff into the
// layer with the specified id and parent, returning the size of the
// new layer in bytes.
func (gdw *NaiveDiffDriver) ApplyDiff(id, parent string, diff archive.Reader) (size int64, err error) {
	// Mount the root filesystem so we can apply the diff/layer.
	layerFs, err := gdw.Get(id, "")
	if err != nil {
		return
	}
	defer gdw.Put(id)

	options := &archive.TarOptions{UIDMaps: gdw.uidMaps,
		GIDMaps: gdw.gidMaps}
	start := time.Now().UTC()
	logrus.Debug("Start untar layer")
	if size, err = ApplyUncompressedLayer(layerFs, diff, options); err != nil {
		return
	}
	logrus.Debugf("Untar time: %vs", time.Now().UTC().Sub(start).Seconds())

	return
}

// DiffSize calculates the changes between the specified layer
// and its parent and returns the size in bytes of the changes
// relative to its base filesystem directory.
func (gdw *NaiveDiffDriver) DiffSize(id, parent string) (size int64, err error) {
	changes, err := gdw.Changes(id, parent)
	if err != nil {
		return
	}

	layerFs, err := gdw.Get(id, "")
	if err != nil {
		return
	}
	defer gdw.Put(id)

	return archive.ChangesSize(layerFs, changes), nil
}
