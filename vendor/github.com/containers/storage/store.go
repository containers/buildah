package storage

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	// register all of the built-in drivers
	_ "github.com/containers/storage/drivers/register"

	"github.com/BurntSushi/toml"
	drivers "github.com/containers/storage/drivers"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/directory"
	"github.com/containers/storage/pkg/idtools"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/parsers"
	"github.com/containers/storage/pkg/stringid"
	"github.com/containers/storage/pkg/stringutils"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/selinux/go-selinux/label"
	"github.com/pkg/errors"
)

var (
	// DefaultStoreOptions is a reasonable default set of options.
	DefaultStoreOptions StoreOptions
	stores              []*store
	storesLock          sync.Mutex
)

// ROFileBasedStore wraps up the methods of the various types of file-based
// data stores that we implement which are needed for both read-only and
// read-write files.
type ROFileBasedStore interface {
	Locker

	// Load reloads the contents of the store from disk.  It should be called
	// with the lock held.
	Load() error
}

// RWFileBasedStore wraps up the methods of various types of file-based data
// stores that we implement using read-write files.
type RWFileBasedStore interface {
	// Save saves the contents of the store to disk.  It should be called with
	// the lock held, and Touch() should be called afterward before releasing the
	// lock.
	Save() error
}

// FileBasedStore wraps up the common methods of various types of file-based
// data stores that we implement.
type FileBasedStore interface {
	ROFileBasedStore
	RWFileBasedStore
}

// ROMetadataStore wraps a method for reading metadata associated with an ID.
type ROMetadataStore interface {
	// Metadata reads metadata associated with an item with the specified ID.
	Metadata(id string) (string, error)
}

// RWMetadataStore wraps a method for setting metadata associated with an ID.
type RWMetadataStore interface {
	// SetMetadata updates the metadata associated with the item with the specified ID.
	SetMetadata(id, metadata string) error
}

// MetadataStore wraps up methods for getting and setting metadata associated with IDs.
type MetadataStore interface {
	ROMetadataStore
	RWMetadataStore
}

// An ROBigDataStore wraps up the read-only big-data related methods of the
// various types of file-based lookaside stores that we implement.
type ROBigDataStore interface {
	// BigData retrieves a (potentially large) piece of data associated with
	// this ID, if it has previously been set.
	BigData(id, key string) ([]byte, error)

	// BigDataSize retrieves the size of a (potentially large) piece of
	// data associated with this ID, if it has previously been set.
	BigDataSize(id, key string) (int64, error)

	// BigDataDigest retrieves the digest of a (potentially large) piece of
	// data associated with this ID, if it has previously been set.
	BigDataDigest(id, key string) (digest.Digest, error)

	// BigDataNames() returns a list of the names of previously-stored pieces of
	// data.
	BigDataNames(id string) ([]string, error)
}

// A RWBigDataStore wraps up the read-write big-data related methods of the
// various types of file-based lookaside stores that we implement.
type RWBigDataStore interface {
	// SetBigData stores a (potentially large) piece of data associated with this
	// ID.
	SetBigData(id, key string, data []byte) error
}

// A BigDataStore wraps up the most common big-data related methods of the
// various types of file-based lookaside stores that we implement.
type BigDataStore interface {
	ROBigDataStore
	RWBigDataStore
}

// A FlaggableStore can have flags set and cleared on items which it manages.
type FlaggableStore interface {
	// ClearFlag removes a named flag from an item in the store.
	ClearFlag(id string, flag string) error

	// SetFlag sets a named flag and its value on an item in the store.
	SetFlag(id string, flag string, value interface{}) error
}

// StoreOptions is used for passing initialization options to GetStore(), for
// initializing a Store object and the underlying storage that it controls.
type StoreOptions struct {
	// RunRoot is the filesystem path under which we can store run-time
	// information, such as the locations of active mount points, that we
	// want to lose if the host is rebooted.
	RunRoot string `json:"runroot,omitempty"`
	// GraphRoot is the filesystem path under which we will store the
	// contents of layers, images, and containers.
	GraphRoot string `json:"root,omitempty"`
	// GraphDriverName is the underlying storage driver that we'll be
	// using.  It only needs to be specified the first time a Store is
	// initialized for a given RunRoot and GraphRoot.
	GraphDriverName string `json:"driver,omitempty"`
	// GraphDriverOptions are driver-specific options.
	GraphDriverOptions []string `json:"driver-options,omitempty"`
	// UIDMap and GIDMap are used for setting up a container's root filesystem
	// for use inside of a user namespace where UID mapping is being used.
	UIDMap []idtools.IDMap `json:"uidmap,omitempty"`
	GIDMap []idtools.IDMap `json:"gidmap,omitempty"`
}

// Store wraps up the various types of file-based stores that we use into a
// singleton object that initializes and manages them all together.
type Store interface {
	// RunRoot, GraphRoot, GraphDriverName, and GraphOptions retrieve
	// settings that were passed to GetStore() when the object was created.
	RunRoot() string
	GraphRoot() string
	GraphDriverName() string
	GraphOptions() []string
	UIDMap() []idtools.IDMap
	GIDMap() []idtools.IDMap

	// GraphDriver obtains and returns a handle to the graph Driver object used
	// by the Store.
	GraphDriver() (drivers.Driver, error)

	// CreateLayer creates a new layer in the underlying storage driver,
	// optionally having the specified ID (one will be assigned if none is
	// specified), with the specified layer (or no layer) as its parent,
	// and with optional names.  (The writeable flag is ignored.)
	CreateLayer(id, parent string, names []string, mountLabel string, writeable bool, options *LayerOptions) (*Layer, error)

	// PutLayer combines the functions of CreateLayer and ApplyDiff,
	// marking the layer for automatic removal if applying the diff fails
	// for any reason.
	//
	// Note that we do some of this work in a child process.  The calling
	// process's main() function needs to import our pkg/reexec package and
	// should begin with something like this in order to allow us to
	// properly start that child process:
	//   if reexec.Init() {
	//       return
	//   }
	PutLayer(id, parent string, names []string, mountLabel string, writeable bool, options *LayerOptions, diff io.Reader) (*Layer, int64, error)

	// CreateImage creates a new image, optionally with the specified ID
	// (one will be assigned if none is specified), with optional names,
	// referring to a specified image, and with optional metadata.  An
	// image is a record which associates the ID of a layer with a
	// additional bookkeeping information which the library stores for the
	// convenience of its caller.
	CreateImage(id string, names []string, layer, metadata string, options *ImageOptions) (*Image, error)

	// CreateContainer creates a new container, optionally with the
	// specified ID (one will be assigned if none is specified), with
	// optional names, using the specified image's top layer as the basis
	// for the container's layer, and assigning the specified ID to that
	// layer (one will be created if none is specified).  A container is a
	// layer which is associated with additional bookkeeping information
	// which the library stores for the convenience of its caller.
	CreateContainer(id string, names []string, image, layer, metadata string, options *ContainerOptions) (*Container, error)

	// Metadata retrieves the metadata which is associated with a layer,
	// image, or container (whichever the passed-in ID refers to).
	Metadata(id string) (string, error)

	// SetMetadata updates the metadata which is associated with a layer,
	// image, or container (whichever the passed-in ID refers to) to match
	// the specified value.  The metadata value can be retrieved at any
	// time using Metadata, or using Layer, Image, or Container and reading
	// the object directly.
	SetMetadata(id, metadata string) error

	// Exists checks if there is a layer, image, or container which has the
	// passed-in ID or name.
	Exists(id string) bool

	// Status asks for a status report, in the form of key-value pairs,
	// from the underlying storage driver.  The contents vary from driver
	// to driver.
	Status() ([][2]string, error)

	// Delete removes the layer, image, or container which has the
	// passed-in ID or name.  Note that no safety checks are performed, so
	// this can leave images with references to layers which do not exist,
	// and layers with references to parents which no longer exist.
	Delete(id string) error

	// DeleteLayer attempts to remove the specified layer.  If the layer is the
	// parent of any other layer, or is referred to by any images, it will return
	// an error.
	DeleteLayer(id string) error

	// DeleteImage removes the specified image if it is not referred to by
	// any containers.  If its top layer is then no longer referred to by
	// any other images and is not the parent of any other layers, its top
	// layer will be removed.  If that layer's parent is no longer referred
	// to by any other images and is not the parent of any other layers,
	// then it, too, will be removed.  This procedure will be repeated
	// until a layer which should not be removed, or the base layer, is
	// reached, at which point the list of removed layers is returned.  If
	// the commit argument is false, the image and layers are not removed,
	// but the list of layers which would be removed is still returned.
	DeleteImage(id string, commit bool) (layers []string, err error)

	// DeleteContainer removes the specified container and its layer.  If
	// there is no matching container, or if the container exists but its
	// layer does not, an error will be returned.
	DeleteContainer(id string) error

	// Wipe removes all known layers, images, and containers.
	Wipe() error

	// Mount attempts to mount a layer, image, or container for access, and
	// returns the pathname if it succeeds.
	// Note if the mountLabel == "", the default label for the container
	// will be used.
	//
	// Note that we do some of this work in a child process.  The calling
	// process's main() function needs to import our pkg/reexec package and
	// should begin with something like this in order to allow us to
	// properly start that child process:
	//   if reexec.Init() {
	//       return
	//   }
	Mount(id, mountLabel string) (string, error)

	// Unmount attempts to unmount a layer, image, or container, given an ID, a
	// name, or a mount path. Returns whether or not the layer is still mounted.
	Unmount(id string, force bool) (bool, error)

	// Mounted returns number of times the layer has been mounted.
	Mounted(id string) (int, error)

	// Changes returns a summary of the changes which would need to be made
	// to one layer to make its contents the same as a second layer.  If
	// the first layer is not specified, the second layer's parent is
	// assumed.  Each Change structure contains a Path relative to the
	// layer's root directory, and a Kind which is either ChangeAdd,
	// ChangeModify, or ChangeDelete.
	Changes(from, to string) ([]archive.Change, error)

	// DiffSize returns a count of the size of the tarstream which would
	// specify the changes returned by Changes.
	DiffSize(from, to string) (int64, error)

	// Diff returns the tarstream which would specify the changes returned
	// by Changes.  If options are passed in, they can override default
	// behaviors.
	Diff(from, to string, options *DiffOptions) (io.ReadCloser, error)

	// ApplyDiff applies a tarstream to a layer.  Information about the
	// tarstream is cached with the layer.  Typically, a layer which is
	// populated using a tarstream will be expected to not be modified in
	// any other way, either before or after the diff is applied.
	//
	// Note that we do some of this work in a child process.  The calling
	// process's main() function needs to import our pkg/reexec package and
	// should begin with something like this in order to allow us to
	// properly start that child process:
	//   if reexec.Init() {
	//       return
	//   }
	ApplyDiff(to string, diff io.Reader) (int64, error)

	// LayersByCompressedDigest returns a slice of the layers with the
	// specified compressed digest value recorded for them.
	LayersByCompressedDigest(d digest.Digest) ([]Layer, error)

	// LayersByUncompressedDigest returns a slice of the layers with the
	// specified uncompressed digest value recorded for them.
	LayersByUncompressedDigest(d digest.Digest) ([]Layer, error)

	// LayerSize returns a cached approximation of the layer's size, or -1
	// if we don't have a value on hand.
	LayerSize(id string) (int64, error)

	// LayerParentOwners returns the UIDs and GIDs of owners of parents of
	// the layer's mountpoint for which the layer's UID and GID maps (if
	// any are defined) don't contain corresponding IDs.
	LayerParentOwners(id string) ([]int, []int, error)

	// Layers returns a list of the currently known layers.
	Layers() ([]Layer, error)

	// Images returns a list of the currently known images.
	Images() ([]Image, error)

	// Containers returns a list of the currently known containers.
	Containers() ([]Container, error)

	// Names returns the list of names for a layer, image, or container.
	Names(id string) ([]string, error)

	// SetNames changes the list of names for a layer, image, or container.
	// Duplicate names are removed from the list automatically.
	SetNames(id string, names []string) error

	// ListImageBigData retrieves a list of the (possibly large) chunks of
	// named data associated with an image.
	ListImageBigData(id string) ([]string, error)

	// ImageBigData retrieves a (possibly large) chunk of named data
	// associated with an image.
	ImageBigData(id, key string) ([]byte, error)

	// ImageBigDataSize retrieves the size of a (possibly large) chunk
	// of named data associated with an image.
	ImageBigDataSize(id, key string) (int64, error)

	// ImageBigDataDigest retrieves the digest of a (possibly large) chunk
	// of named data associated with an image.
	ImageBigDataDigest(id, key string) (digest.Digest, error)

	// SetImageBigData stores a (possibly large) chunk of named data associated
	// with an image.
	SetImageBigData(id, key string, data []byte) error

	// ImageSize computes the size of the image's layers and ancillary data.
	ImageSize(id string) (int64, error)

	// ListContainerBigData retrieves a list of the (possibly large) chunks of
	// named data associated with a container.
	ListContainerBigData(id string) ([]string, error)

	// ContainerBigData retrieves a (possibly large) chunk of named data
	// associated with a container.
	ContainerBigData(id, key string) ([]byte, error)

	// ContainerBigDataSize retrieves the size of a (possibly large)
	// chunk of named data associated with a container.
	ContainerBigDataSize(id, key string) (int64, error)

	// ContainerBigDataDigest retrieves the digest of a (possibly large)
	// chunk of named data associated with a container.
	ContainerBigDataDigest(id, key string) (digest.Digest, error)

	// SetContainerBigData stores a (possibly large) chunk of named data
	// associated with a container.
	SetContainerBigData(id, key string, data []byte) error

	// ContainerSize computes the size of the container's layer and ancillary
	// data.  Warning:  this is a potentially expensive operation.
	ContainerSize(id string) (int64, error)

	// Layer returns a specific layer.
	Layer(id string) (*Layer, error)

	// Image returns a specific image.
	Image(id string) (*Image, error)

	// ImagesByTopLayer returns a list of images which reference the specified
	// layer as their top layer.  They will have different IDs and names
	// and may have different metadata, big data items, and flags.
	ImagesByTopLayer(id string) ([]*Image, error)

	// ImagesByDigest returns a list of images which contain a big data item
	// named ImageDigestBigDataKey whose contents have the specified digest.
	ImagesByDigest(d digest.Digest) ([]*Image, error)

	// Container returns a specific container.
	Container(id string) (*Container, error)

	// ContainerByLayer returns a specific container based on its layer ID or
	// name.
	ContainerByLayer(id string) (*Container, error)

	// ContainerDirectory returns a path of a directory which the caller
	// can use to store data, specific to the container, which the library
	// does not directly manage.  The directory will be deleted when the
	// container is deleted.
	ContainerDirectory(id string) (string, error)

	// SetContainerDirectoryFile is a convenience function which stores
	// a piece of data in the specified file relative to the container's
	// directory.
	SetContainerDirectoryFile(id, file string, data []byte) error

	// FromContainerDirectory is a convenience function which reads
	// the contents of the specified file relative to the container's
	// directory.
	FromContainerDirectory(id, file string) ([]byte, error)

	// ContainerRunDirectory returns a path of a directory which the
	// caller can use to store data, specific to the container, which the
	// library does not directly manage.  The directory will be deleted
	// when the host system is restarted.
	ContainerRunDirectory(id string) (string, error)

	// SetContainerRunDirectoryFile is a convenience function which stores
	// a piece of data in the specified file relative to the container's
	// run directory.
	SetContainerRunDirectoryFile(id, file string, data []byte) error

	// FromContainerRunDirectory is a convenience function which reads
	// the contents of the specified file relative to the container's run
	// directory.
	FromContainerRunDirectory(id, file string) ([]byte, error)

	// ContainerParentOwners returns the UIDs and GIDs of owners of parents
	// of the container's layer's mountpoint for which the layer's UID and
	// GID maps (if any are defined) don't contain corresponding IDs.
	ContainerParentOwners(id string) ([]int, []int, error)

	// Lookup returns the ID of a layer, image, or container with the specified
	// name or ID.
	Lookup(name string) (string, error)

	// Shutdown attempts to free any kernel resources which are being used
	// by the underlying driver.  If "force" is true, any mounted (i.e., in
	// use) layers are unmounted beforehand.  If "force" is not true, then
	// layers being in use is considered to be an error condition.  A list
	// of still-mounted layers is returned along with possible errors.
	Shutdown(force bool) (layers []string, err error)

	// Version returns version information, in the form of key-value pairs, from
	// the storage package.
	Version() ([][2]string, error)
}

// IDMappingOptions are used for specifying how ID mapping should be set up for
// a layer or container.
type IDMappingOptions struct {
	// UIDMap and GIDMap are used for setting up a layer's root filesystem
	// for use inside of a user namespace where ID mapping is being used.
	// If HostUIDMapping/HostGIDMapping is true, no mapping of the
	// respective type will be used.  Otherwise, if UIDMap and/or GIDMap
	// contain at least one mapping, one or both will be used.  By default,
	// if neither of those conditions apply, if the layer has a parent
	// layer, the parent layer's mapping will be used, and if it does not
	// have a parent layer, the mapping which was passed to the Store
	// object when it was initialized will be used.
	HostUIDMapping bool
	HostGIDMapping bool
	UIDMap         []idtools.IDMap
	GIDMap         []idtools.IDMap
}

// LayerOptions is used for passing options to a Store's CreateLayer() and PutLayer() methods.
type LayerOptions struct {
	// IDMappingOptions specifies the type of ID mapping which should be
	// used for this layer.  If nothing is specified, the layer will
	// inherit settings from its parent layer or, if it has no parent
	// layer, the Store object.
	IDMappingOptions
}

// ImageOptions is used for passing options to a Store's CreateImage() method.
type ImageOptions struct {
	// CreationDate, if not zero, will override the default behavior of marking the image as having been
	// created when CreateImage() was called, recording CreationDate instead.
	CreationDate time.Time
	// Digest is a hard-coded digest value that we can use to look up the image.  It is optional.
	Digest digest.Digest
}

// ContainerOptions is used for passing options to a Store's CreateContainer() method.
type ContainerOptions struct {
	// IDMappingOptions specifies the type of ID mapping which should be
	// used for this container's layer.  If nothing is specified, the
	// container's layer will inherit settings from the image's top layer
	// or, if it is not being created based on an image, the Store object.
	IDMappingOptions
	LabelOpts []string
	Flags     map[string]interface{}
	MountOpts []string
}

type store struct {
	lastLoaded      time.Time
	runRoot         string
	graphLock       Locker
	graphRoot       string
	graphDriverName string
	graphOptions    []string
	uidMap          []idtools.IDMap
	gidMap          []idtools.IDMap
	graphDriver     drivers.Driver
	layerStore      LayerStore
	roLayerStores   []ROLayerStore
	imageStore      ImageStore
	roImageStores   []ROImageStore
	containerStore  ContainerStore
}

// GetStore attempts to find an already-created Store object matching the
// specified location and graph driver, and if it can't, it creates and
// initializes a new Store object, and the underlying storage that it controls.
//
// If StoreOptions `options` haven't been fully populated, then DefaultStoreOptions are used.
//
// These defaults observe environment variables:
//  * `STORAGE_DRIVER` for the name of the storage driver to attempt to use
//  * `STORAGE_OPTS` for the string of options to pass to the driver
//
// Note that we do some of this work in a child process.  The calling process's
// main() function needs to import our pkg/reexec package and should begin with
// something like this in order to allow us to properly start that child
// process:
//   if reexec.Init() {
//       return
//   }
func GetStore(options StoreOptions) (Store, error) {
	if options.RunRoot == "" && options.GraphRoot == "" && options.GraphDriverName == "" && len(options.GraphDriverOptions) == 0 {
		options = DefaultStoreOptions
	}

	if options.GraphRoot != "" {
		options.GraphRoot = filepath.Clean(options.GraphRoot)
	}
	if options.RunRoot != "" {
		options.RunRoot = filepath.Clean(options.RunRoot)
	}

	storesLock.Lock()
	defer storesLock.Unlock()

	for _, s := range stores {
		if s.graphRoot == options.GraphRoot && (options.GraphDriverName == "" || s.graphDriverName == options.GraphDriverName) {
			return s, nil
		}
	}

	if options.GraphRoot == "" {
		return nil, ErrIncompleteOptions
	}
	if options.RunRoot == "" {
		return nil, ErrIncompleteOptions
	}

	if err := os.MkdirAll(options.RunRoot, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	if err := os.MkdirAll(options.GraphRoot, 0700); err != nil && !os.IsExist(err) {
		return nil, err
	}
	for _, subdir := range []string{"mounts", "tmp", options.GraphDriverName} {
		if err := os.MkdirAll(filepath.Join(options.GraphRoot, subdir), 0700); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}

	graphLock, err := GetLockfile(filepath.Join(options.GraphRoot, "storage.lock"))
	if err != nil {
		return nil, err
	}
	s := &store{
		runRoot:         options.RunRoot,
		graphLock:       graphLock,
		graphRoot:       options.GraphRoot,
		graphDriverName: options.GraphDriverName,
		graphOptions:    options.GraphDriverOptions,
		uidMap:          copyIDMap(options.UIDMap),
		gidMap:          copyIDMap(options.GIDMap),
	}
	if err := s.load(); err != nil {
		return nil, err
	}

	stores = append(stores, s)

	return s, nil
}

func copyIDMap(idmap []idtools.IDMap) []idtools.IDMap {
	m := []idtools.IDMap{}
	if idmap != nil {
		m = make([]idtools.IDMap, len(idmap))
		copy(m, idmap)
	}
	if len(m) > 0 {
		return m[:]
	}
	return nil
}

func (s *store) RunRoot() string {
	return s.runRoot
}

func (s *store) GraphDriverName() string {
	return s.graphDriverName
}

func (s *store) GraphRoot() string {
	return s.graphRoot
}

func (s *store) GraphOptions() []string {
	return s.graphOptions
}

func (s *store) UIDMap() []idtools.IDMap {
	return copyIDMap(s.uidMap)
}

func (s *store) GIDMap() []idtools.IDMap {
	return copyIDMap(s.gidMap)
}

func (s *store) load() error {
	driver, err := s.GraphDriver()
	if err != nil {
		return err
	}
	s.graphDriver = driver
	s.graphDriverName = driver.String()
	driverPrefix := s.graphDriverName + "-"

	rls, err := s.LayerStore()
	if err != nil {
		return err
	}
	s.layerStore = rls
	if _, err := s.ROLayerStores(); err != nil {
		return err
	}

	gipath := filepath.Join(s.graphRoot, driverPrefix+"images")
	if err := os.MkdirAll(gipath, 0700); err != nil {
		return err
	}
	ris, err := newImageStore(gipath)
	if err != nil {
		return err
	}
	s.imageStore = ris
	if _, err := s.ROImageStores(); err != nil {
		return err
	}

	gcpath := filepath.Join(s.graphRoot, driverPrefix+"containers")
	if err := os.MkdirAll(gcpath, 0700); err != nil {
		return err
	}
	rcs, err := newContainerStore(gcpath)
	if err != nil {
		return err
	}
	rcpath := filepath.Join(s.runRoot, driverPrefix+"containers")
	if err := os.MkdirAll(rcpath, 0700); err != nil {
		return err
	}
	s.containerStore = rcs
	return nil
}

func (s *store) getGraphDriver() (drivers.Driver, error) {
	if s.graphDriver != nil {
		return s.graphDriver, nil
	}
	config := drivers.Options{
		Root:          s.graphRoot,
		DriverOptions: s.graphOptions,
		UIDMaps:       s.uidMap,
		GIDMaps:       s.gidMap,
	}
	driver, err := drivers.New(s.graphDriverName, config)
	if err != nil {
		return nil, err
	}
	s.graphDriver = driver
	s.graphDriverName = driver.String()
	return driver, nil
}

func (s *store) GraphDriver() (drivers.Driver, error) {
	s.graphLock.Lock()
	defer s.graphLock.Unlock()
	if s.graphLock.TouchedSince(s.lastLoaded) {
		s.graphDriver = nil
		s.layerStore = nil
		s.lastLoaded = time.Now()
	}
	return s.getGraphDriver()
}

// LayerStore obtains and returns a handle to the writeable layer store object
// used by the Store.  Accessing this store directly will bypass locking and
// synchronization, so it is not a part of the exported Store interface.
func (s *store) LayerStore() (LayerStore, error) {
	s.graphLock.Lock()
	defer s.graphLock.Unlock()
	if s.graphLock.TouchedSince(s.lastLoaded) {
		s.graphDriver = nil
		s.layerStore = nil
		s.lastLoaded = time.Now()
	}
	if s.layerStore != nil {
		return s.layerStore, nil
	}
	driver, err := s.getGraphDriver()
	if err != nil {
		return nil, err
	}
	driverPrefix := s.graphDriverName + "-"
	rlpath := filepath.Join(s.runRoot, driverPrefix+"layers")
	if err := os.MkdirAll(rlpath, 0700); err != nil {
		return nil, err
	}
	glpath := filepath.Join(s.graphRoot, driverPrefix+"layers")
	if err := os.MkdirAll(glpath, 0700); err != nil {
		return nil, err
	}
	rls, err := newLayerStore(rlpath, glpath, driver, s.uidMap, s.gidMap)
	if err != nil {
		return nil, err
	}
	s.layerStore = rls
	return s.layerStore, nil
}

// ROLayerStores obtains additional read/only layer store objects used by the
// Store.  Accessing these stores directly will bypass locking and
// synchronization, so it is not part of the exported Store interface.
func (s *store) ROLayerStores() ([]ROLayerStore, error) {
	s.graphLock.Lock()
	defer s.graphLock.Unlock()
	if s.roLayerStores != nil {
		return s.roLayerStores, nil
	}
	driver, err := s.getGraphDriver()
	if err != nil {
		return nil, err
	}
	driverPrefix := s.graphDriverName + "-"
	rlpath := filepath.Join(s.runRoot, driverPrefix+"layers")
	if err := os.MkdirAll(rlpath, 0700); err != nil {
		return nil, err
	}
	for _, store := range driver.AdditionalImageStores() {
		glpath := filepath.Join(store, driverPrefix+"layers")
		rls, err := newROLayerStore(rlpath, glpath, driver)
		if err != nil {
			return nil, err
		}
		s.roLayerStores = append(s.roLayerStores, rls)
	}
	return s.roLayerStores, nil
}

// ImageStore obtains and returns a handle to the writable image store object
// used by the Store.  Accessing this store directly will bypass locking and
// synchronization, so it is not a part of the exported Store interface.
func (s *store) ImageStore() (ImageStore, error) {
	if s.imageStore != nil {
		return s.imageStore, nil
	}
	return nil, ErrLoadError
}

// ROImageStores obtains additional read/only image store objects used by the
// Store.  Accessing these stores directly will bypass locking and
// synchronization, so it is not a part of the exported Store interface.
func (s *store) ROImageStores() ([]ROImageStore, error) {
	if len(s.roImageStores) != 0 {
		return s.roImageStores, nil
	}
	driver, err := s.getGraphDriver()
	if err != nil {
		return nil, err
	}
	driverPrefix := s.graphDriverName + "-"
	for _, store := range driver.AdditionalImageStores() {
		gipath := filepath.Join(store, driverPrefix+"images")
		ris, err := newROImageStore(gipath)
		if err != nil {
			return nil, err
		}
		s.roImageStores = append(s.roImageStores, ris)
	}
	return s.roImageStores, nil
}

// ContainerStore obtains and returns a handle to the container store object
// used by the Store.  Accessing this store directly will bypass locking and
// synchronization, so it is not a part of the exported Store interface.
func (s *store) ContainerStore() (ContainerStore, error) {
	if s.containerStore != nil {
		return s.containerStore, nil
	}
	return nil, ErrLoadError
}

func (s *store) PutLayer(id, parent string, names []string, mountLabel string, writeable bool, options *LayerOptions, diff io.Reader) (*Layer, int64, error) {
	var parentLayer *Layer
	rlstore, err := s.LayerStore()
	if err != nil {
		return nil, -1, err
	}
	rlstores, err := s.ROLayerStores()
	if err != nil {
		return nil, -1, err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, -1, err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	if id == "" {
		id = stringid.GenerateRandomID()
	}
	if options == nil {
		options = &LayerOptions{}
	}
	if options.HostUIDMapping {
		options.UIDMap = nil
	}
	if options.HostGIDMapping {
		options.GIDMap = nil
	}
	uidMap := options.UIDMap
	gidMap := options.GIDMap
	if parent != "" {
		var ilayer *Layer
		for _, lstore := range append([]ROLayerStore{rlstore}, rlstores...) {
			if lstore != rlstore {
				lstore.Lock()
				defer lstore.Unlock()
				if modified, err := lstore.Modified(); modified || err != nil {
					lstore.Load()
				}
			}
			if l, err := lstore.Get(parent); err == nil && l != nil {
				ilayer = l
				parent = ilayer.ID
				break
			}
		}
		if ilayer == nil {
			return nil, -1, ErrLayerUnknown
		}
		parentLayer = ilayer
		containers, err := rcstore.Containers()
		if err != nil {
			return nil, -1, err
		}
		for _, container := range containers {
			if container.LayerID == parent {
				return nil, -1, ErrParentIsContainer
			}
		}
		if !options.HostUIDMapping && len(options.UIDMap) == 0 {
			uidMap = ilayer.UIDMap
		}
		if !options.HostGIDMapping && len(options.GIDMap) == 0 {
			gidMap = ilayer.GIDMap
		}
	} else {
		if !options.HostUIDMapping && len(options.UIDMap) == 0 {
			uidMap = s.uidMap
		}
		if !options.HostGIDMapping && len(options.GIDMap) == 0 {
			gidMap = s.gidMap
		}
	}
	var layerOptions *LayerOptions
	if s.graphDriver.SupportsShifting() {
		layerOptions = &LayerOptions{IDMappingOptions: IDMappingOptions{HostUIDMapping: true, HostGIDMapping: true, UIDMap: nil, GIDMap: nil}}
	} else {
		layerOptions = &LayerOptions{
			IDMappingOptions: IDMappingOptions{
				HostUIDMapping: options.HostUIDMapping,
				HostGIDMapping: options.HostGIDMapping,
				UIDMap:         copyIDMap(uidMap),
				GIDMap:         copyIDMap(gidMap),
			},
		}
	}
	return rlstore.Put(id, parentLayer, names, mountLabel, nil, layerOptions, writeable, nil, diff)
}

func (s *store) CreateLayer(id, parent string, names []string, mountLabel string, writeable bool, options *LayerOptions) (*Layer, error) {
	layer, _, err := s.PutLayer(id, parent, names, mountLabel, writeable, options, nil)
	return layer, err
}

func (s *store) CreateImage(id string, names []string, layer, metadata string, options *ImageOptions) (*Image, error) {
	if id == "" {
		id = stringid.GenerateRandomID()
	}

	if layer != "" {
		lstore, err := s.LayerStore()
		if err != nil {
			return nil, err
		}
		lstores, err := s.ROLayerStores()
		if err != nil {
			return nil, err
		}
		var ilayer *Layer
		for _, store := range append([]ROLayerStore{lstore}, lstores...) {
			store.Lock()
			defer store.Unlock()
			if modified, err := store.Modified(); modified || err != nil {
				store.Load()
			}
			ilayer, err = store.Get(layer)
			if err == nil {
				break
			}
		}
		if ilayer == nil {
			return nil, ErrLayerUnknown
		}
		layer = ilayer.ID
	}

	ristore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}

	creationDate := time.Now().UTC()
	if options != nil && !options.CreationDate.IsZero() {
		creationDate = options.CreationDate
	}

	return ristore.Create(id, names, layer, metadata, creationDate, options.Digest)
}

func (s *store) imageTopLayerForMapping(image *Image, ristore ROImageStore, readWrite bool, rlstore LayerStore, lstores []ROLayerStore, options IDMappingOptions) (*Layer, error) {
	layerMatchesMappingOptions := func(layer *Layer, options IDMappingOptions) bool {
		// If the driver supports shifting and the layer has no mappings, we can use it.
		if s.graphDriver.SupportsShifting() && len(layer.UIDMap) == 0 && len(layer.GIDMap) == 0 {
			return true
		}
		// If we want host mapping, and the layer uses mappings, it's not the best match.
		if options.HostUIDMapping && len(layer.UIDMap) != 0 {
			return false
		}
		if options.HostGIDMapping && len(layer.GIDMap) != 0 {
			return false
		}
		// If we don't care about the mapping, it's fine.
		if len(options.UIDMap) == 0 && len(options.GIDMap) == 0 {
			return true
		}
		// Compare the maps.
		return reflect.DeepEqual(layer.UIDMap, options.UIDMap) && reflect.DeepEqual(layer.GIDMap, options.GIDMap)
	}
	var layer, parentLayer *Layer
	var layerHomeStore ROLayerStore
	// Locate the image's top layer and its parent, if it has one.
	for _, store := range append([]ROLayerStore{rlstore}, lstores...) {
		if store != rlstore {
			store.Lock()
			defer store.Unlock()
			if modified, err := store.Modified(); modified || err != nil {
				store.Load()
			}
		}
		// Walk the top layer list.
		for _, candidate := range append([]string{image.TopLayer}, image.MappedTopLayers...) {
			if cLayer, err := store.Get(candidate); err == nil {
				// We want the layer's parent, too, if it has one.
				var cParentLayer *Layer
				if cLayer.Parent != "" {
					// Its parent should be around here, somewhere.
					if cParentLayer, err = store.Get(cLayer.Parent); err != nil {
						// Nope, couldn't find it.  We're not going to be able
						// to diff this one properly.
						continue
					}
				}
				// If the layer matches the desired mappings, it's a perfect match,
				// so we're actually done here.
				if layerMatchesMappingOptions(cLayer, options) {
					return cLayer, nil
				}
				// Record the first one that we found, even if it's not ideal, so that
				// we have a starting point.
				if layer == nil {
					layer = cLayer
					parentLayer = cParentLayer
					layerHomeStore = store
				}
			}
		}
	}
	if layer == nil {
		return nil, ErrLayerUnknown
	}
	// The top layer's mappings don't match the ones we want, but it's in a read-only
	// image store, so we can't create and add a mapped copy of the layer to the image.
	if !readWrite {
		return layer, nil
	}
	// The top layer's mappings don't match the ones we want, and it's in an image store
	// that lets us edit image metadata...
	if istore, ok := ristore.(*imageStore); ok {
		// ... so extract the layer's contents, create a new copy of it with the
		// desired mappings, and register it as an alternate top layer in the image.
		noCompression := archive.Uncompressed
		diffOptions := DiffOptions{
			Compression: &noCompression,
		}
		rc, err := layerHomeStore.Diff("", layer.ID, &diffOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading layer %q to create an ID-mapped version of it", layer.ID)
		}
		defer rc.Close()

		var layerOptions LayerOptions
		if s.graphDriver.SupportsShifting() {
			layerOptions = LayerOptions{IDMappingOptions: IDMappingOptions{HostUIDMapping: true, HostGIDMapping: true, UIDMap: nil, GIDMap: nil}}
		} else {
			layerOptions = LayerOptions{
				IDMappingOptions: IDMappingOptions{
					HostUIDMapping: options.HostUIDMapping,
					HostGIDMapping: options.HostGIDMapping,
					UIDMap:         copyIDMap(options.UIDMap),
					GIDMap:         copyIDMap(options.GIDMap),
				},
			}
		}
		mappedLayer, _, err := rlstore.Put("", parentLayer, nil, layer.MountLabel, nil, &layerOptions, false, nil, rc)
		if err != nil {
			return nil, errors.Wrapf(err, "error creating ID-mapped copy of layer %q", layer.ID)
		}
		if err = istore.addMappedTopLayer(image.ID, mappedLayer.ID); err != nil {
			if err2 := rlstore.Delete(mappedLayer.ID); err2 != nil {
				err = errors.WithMessage(err, fmt.Sprintf("error deleting layer %q: %v", mappedLayer.ID, err2))
			}
			return nil, errors.Wrapf(err, "error registering ID-mapped layer with image %q", image.ID)
		}
		layer = mappedLayer
	}
	return layer, nil
}

func (s *store) CreateContainer(id string, names []string, image, layer, metadata string, options *ContainerOptions) (*Container, error) {
	if options == nil {
		options = &ContainerOptions{}
	}
	if options.HostUIDMapping {
		options.UIDMap = nil
	}
	if options.HostGIDMapping {
		options.GIDMap = nil
	}
	rlstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	if id == "" {
		id = stringid.GenerateRandomID()
	}

	var imageTopLayer *Layer
	imageID := ""
	uidMap := options.UIDMap
	gidMap := options.GIDMap

	idMappingsOptions := options.IDMappingOptions
	if image != "" {
		var imageHomeStore ROImageStore
		lstores, err := s.ROLayerStores()
		if err != nil {
			return nil, err
		}
		istore, err := s.ImageStore()
		if err != nil {
			return nil, err
		}
		istores, err := s.ROImageStores()
		if err != nil {
			return nil, err
		}
		rlstore.Lock()
		defer rlstore.Unlock()
		if modified, err := rlstore.Modified(); modified || err != nil {
			rlstore.Load()
		}
		var cimage *Image
		for _, store := range append([]ROImageStore{istore}, istores...) {
			store.Lock()
			defer store.Unlock()
			if modified, err := store.Modified(); modified || err != nil {
				store.Load()
			}
			cimage, err = store.Get(image)
			if err == nil {
				imageHomeStore = store
				break
			}
		}
		if cimage == nil {
			return nil, ErrImageUnknown
		}
		imageID = cimage.ID

		ilayer, err := s.imageTopLayerForMapping(cimage, imageHomeStore, imageHomeStore == istore, rlstore, lstores, idMappingsOptions)
		if err != nil {
			return nil, err
		}
		imageTopLayer = ilayer
		if !options.HostUIDMapping && len(options.UIDMap) == 0 {
			uidMap = ilayer.UIDMap
		}
		if !options.HostGIDMapping && len(options.GIDMap) == 0 {
			gidMap = ilayer.GIDMap
		}
	} else {
		rlstore.Lock()
		defer rlstore.Unlock()
		if modified, err := rlstore.Modified(); modified || err != nil {
			rlstore.Load()
		}
		if !options.HostUIDMapping && len(options.UIDMap) == 0 {
			uidMap = s.uidMap
		}
		if !options.HostGIDMapping && len(options.GIDMap) == 0 {
			gidMap = s.gidMap
		}
	}
	var layerOptions *LayerOptions
	if s.graphDriver.SupportsShifting() {
		layerOptions = &LayerOptions{IDMappingOptions: IDMappingOptions{HostUIDMapping: true, HostGIDMapping: true, UIDMap: nil, GIDMap: nil}}
	} else {
		layerOptions = &LayerOptions{
			IDMappingOptions: IDMappingOptions{
				HostUIDMapping: idMappingsOptions.HostUIDMapping,
				HostGIDMapping: idMappingsOptions.HostGIDMapping,
				UIDMap:         copyIDMap(uidMap),
				GIDMap:         copyIDMap(gidMap),
			},
		}
	}
	if options.Flags == nil {
		options.Flags = make(map[string]interface{})
	}
	plabel, _ := options.Flags["ProcessLabel"].(string)
	mlabel, _ := options.Flags["MountLabel"].(string)
	if (plabel == "" && mlabel != "") ||
		(plabel != "" && mlabel == "") {
		return nil, errors.Errorf("ProcessLabel and Mountlabel must either not be specified or both specified")
	}

	if plabel == "" {
		processLabel, mountLabel, err := label.InitLabels(options.LabelOpts)
		if err != nil {
			return nil, err
		}
		options.Flags["ProcessLabel"] = processLabel
		options.Flags["MountLabel"] = mountLabel
	}

	clayer, err := rlstore.Create(layer, imageTopLayer, nil, options.Flags["MountLabel"].(string), nil, layerOptions, true)
	if err != nil {
		return nil, err
	}
	layer = clayer.ID
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	options.IDMappingOptions = IDMappingOptions{
		HostUIDMapping: len(options.UIDMap) == 0,
		HostGIDMapping: len(options.GIDMap) == 0,
		UIDMap:         copyIDMap(options.UIDMap),
		GIDMap:         copyIDMap(options.GIDMap),
	}
	container, err := rcstore.Create(id, names, imageID, layer, metadata, options)
	if err != nil || container == nil {
		rlstore.Delete(layer)
	}
	return container, err
}

func (s *store) SetMetadata(id, metadata string) error {
	rlstore, err := s.LayerStore()
	if err != nil {
		return err
	}
	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rlstore.Exists(id) {
		return rlstore.SetMetadata(id, metadata)
	}
	if ristore.Exists(id) {
		return ristore.SetMetadata(id, metadata)
	}
	if rcstore.Exists(id) {
		return rcstore.SetMetadata(id, metadata)
	}
	return ErrNotAnID
}

func (s *store) Metadata(id string) (string, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return "", err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return "", err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(id) {
			return store.Metadata(id)
		}
	}

	istore, err := s.ImageStore()
	if err != nil {
		return "", err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return "", err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(id) {
			return store.Metadata(id)
		}
	}

	cstore, err := s.ContainerStore()
	if err != nil {
		return "", err
	}
	cstore.Lock()
	defer cstore.Unlock()
	if modified, err := cstore.Modified(); modified || err != nil {
		cstore.Load()
	}
	if cstore.Exists(id) {
		return cstore.Metadata(id)
	}
	return "", ErrNotAnID
}

func (s *store) ListImageBigData(id string) ([]string, error) {
	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		bigDataNames, err := store.BigDataNames(id)
		if err == nil {
			return bigDataNames, err
		}
	}
	return nil, ErrImageUnknown
}

func (s *store) ImageBigDataSize(id, key string) (int64, error) {
	istore, err := s.ImageStore()
	if err != nil {
		return -1, err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return -1, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		size, err := store.BigDataSize(id, key)
		if err == nil {
			return size, nil
		}
	}
	return -1, ErrSizeUnknown
}

func (s *store) ImageBigDataDigest(id, key string) (digest.Digest, error) {
	ristore, err := s.ImageStore()
	if err != nil {
		return "", err
	}
	stores, err := s.ROImageStores()
	if err != nil {
		return "", err
	}
	stores = append([]ROImageStore{ristore}, stores...)
	for _, ristore := range stores {
		ristore.Lock()
		defer ristore.Unlock()
		if modified, err := ristore.Modified(); modified || err != nil {
			ristore.Load()
		}
		d, err := ristore.BigDataDigest(id, key)
		if err == nil && d.Validate() == nil {
			return d, nil
		}
	}
	return "", ErrDigestUnknown
}

func (s *store) ImageBigData(id, key string) ([]byte, error) {
	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		data, err := store.BigData(id, key)
		if err == nil {
			return data, nil
		}
	}
	return nil, ErrImageUnknown
}

func (s *store) SetImageBigData(id, key string, data []byte) error {
	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}

	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}

	return ristore.SetBigData(id, key, data)
}

func (s *store) ImageSize(id string) (int64, error) {
	var image *Image

	lstore, err := s.LayerStore()
	if err != nil {
		return -1, errors.Wrapf(err, "error loading primary layer store data")
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return -1, errors.Wrapf(err, "error loading additional layer stores")
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
	}

	var imageStore ROBigDataStore
	istore, err := s.ImageStore()
	if err != nil {
		return -1, errors.Wrapf(err, "error loading primary image store data")
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return -1, errors.Wrapf(err, "error loading additional image stores")
	}

	// Look for the image's record.
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if image, err = store.Get(id); err == nil {
			imageStore = store
			break
		}
	}
	if image == nil {
		return -1, errors.Wrapf(ErrImageUnknown, "error locating image with ID %q", id)
	}

	// Start with a list of the image's top layers.
	queue := make(map[string]struct{})
	for _, layerID := range append([]string{image.TopLayer}, image.MappedTopLayers...) {
		queue[layerID] = struct{}{}
	}
	visited := make(map[string]struct{})
	// Walk all of the layers.
	var size int64
	for len(visited) < len(queue) {
		for layerID := range queue {
			// Visit each layer only once.
			if _, ok := visited[layerID]; ok {
				continue
			}
			visited[layerID] = struct{}{}
			// Look for the layer and the store that knows about it.
			var layerStore ROLayerStore
			var layer *Layer
			for _, store := range append([]ROLayerStore{lstore}, lstores...) {
				if layer, err = store.Get(layerID); err == nil {
					layerStore = store
					break
				}
			}
			if layer == nil {
				return -1, errors.Wrapf(ErrLayerUnknown, "error locating layer with ID %q", layerID)
			}
			// The UncompressedSize is only valid if there's a digest to go with it.
			n := layer.UncompressedSize
			if layer.UncompressedDigest == "" {
				// Compute the size.
				n, err = layerStore.DiffSize("", layer.ID)
				if err != nil {
					return -1, errors.Wrapf(err, "size/digest of layer with ID %q could not be calculated", layerID)
				}
			}
			// Count this layer.
			size += n
			// Make a note to visit the layer's parent if we haven't already.
			if layer.Parent != "" {
				queue[layer.Parent] = struct{}{}
			}
		}
	}

	// Count big data items.
	names, err := imageStore.BigDataNames(id)
	if err != nil {
		return -1, errors.Wrapf(err, "error reading list of big data items for image %q", id)
	}
	for _, name := range names {
		n, err := imageStore.BigDataSize(id, name)
		if err != nil {
			return -1, errors.Wrapf(err, "error reading size of big data item %q for image %q", name, id)
		}
		size += n
	}

	return size, nil
}

func (s *store) ContainerSize(id string) (int64, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return -1, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return -1, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
	}

	// Get the location of the container directory and container run directory.
	// Do it before we lock the container store because they do, too.
	cdir, err := s.ContainerDirectory(id)
	if err != nil {
		return -1, err
	}
	rdir, err := s.ContainerRunDirectory(id)
	if err != nil {
		return -1, err
	}

	rcstore, err := s.ContainerStore()
	if err != nil {
		return -1, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	// Read the container record.
	container, err := rcstore.Get(id)
	if err != nil {
		return -1, err
	}

	// Read the container's layer's size.
	var layer *Layer
	var size int64
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		if layer, err = store.Get(container.LayerID); err == nil {
			size, err = store.DiffSize("", layer.ID)
			if err != nil {
				return -1, errors.Wrapf(err, "error determining size of layer with ID %q", layer.ID)
			}
			break
		}
	}
	if layer == nil {
		return -1, errors.Wrapf(ErrLayerUnknown, "error locating layer with ID %q", container.LayerID)
	}

	// Count big data items.
	names, err := rcstore.BigDataNames(id)
	if err != nil {
		return -1, errors.Wrapf(err, "error reading list of big data items for container %q", container.ID)
	}
	for _, name := range names {
		n, err := rcstore.BigDataSize(id, name)
		if err != nil {
			return -1, errors.Wrapf(err, "error reading size of big data item %q for container %q", name, id)
		}
		size += n
	}

	// Count the size of our container directory and container run directory.
	n, err := directory.Size(cdir)
	if err != nil {
		return -1, err
	}
	size += n
	n, err = directory.Size(rdir)
	if err != nil {
		return -1, err
	}
	size += n

	return size, nil
}

func (s *store) ListContainerBigData(id string) ([]string, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}

	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	return rcstore.BigDataNames(id)
}

func (s *store) ContainerBigDataSize(id, key string) (int64, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return -1, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	return rcstore.BigDataSize(id, key)
}

func (s *store) ContainerBigDataDigest(id, key string) (digest.Digest, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return "", err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	return rcstore.BigDataDigest(id, key)
}

func (s *store) ContainerBigData(id, key string) ([]byte, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	return rcstore.BigData(id, key)
}

func (s *store) SetContainerBigData(id, key string, data []byte) error {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	return rcstore.SetBigData(id, key, data)
}

func (s *store) Exists(id string) bool {
	lstore, err := s.LayerStore()
	if err != nil {
		return false
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return false
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(id) {
			return true
		}
	}

	istore, err := s.ImageStore()
	if err != nil {
		return false
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return false
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(id) {
			return true
		}
	}

	rcstore, err := s.ContainerStore()
	if err != nil {
		return false
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	if rcstore.Exists(id) {
		return true
	}

	return false
}

func dedupeNames(names []string) []string {
	seen := make(map[string]bool)
	deduped := make([]string, 0, len(names))
	for _, name := range names {
		if _, wasSeen := seen[name]; !wasSeen {
			seen[name] = true
			deduped = append(deduped, name)
		}
	}
	return deduped
}

func (s *store) SetNames(id string, names []string) error {
	deduped := dedupeNames(names)

	rlstore, err := s.LayerStore()
	if err != nil {
		return err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	if rlstore.Exists(id) {
		return rlstore.SetNames(id, deduped)
	}

	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	if ristore.Exists(id) {
		return ristore.SetNames(id, deduped)
	}

	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	if rcstore.Exists(id) {
		return rcstore.SetNames(id, deduped)
	}
	return ErrLayerUnknown
}

func (s *store) Names(id string) ([]string, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if l, err := store.Get(id); l != nil && err == nil {
			return l.Names, nil
		}
	}

	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if i, err := store.Get(id); i != nil && err == nil {
			return i.Names, nil
		}
	}

	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	if c, err := rcstore.Get(id); c != nil && err == nil {
		return c.Names, nil
	}
	return nil, ErrLayerUnknown
}

func (s *store) Lookup(name string) (string, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return "", err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return "", err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if l, err := store.Get(name); l != nil && err == nil {
			return l.ID, nil
		}
	}

	istore, err := s.ImageStore()
	if err != nil {
		return "", err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return "", err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if i, err := store.Get(name); i != nil && err == nil {
			return i.ID, nil
		}
	}

	cstore, err := s.ContainerStore()
	if err != nil {
		return "", err
	}
	cstore.Lock()
	defer cstore.Unlock()
	if modified, err := cstore.Modified(); modified || err != nil {
		cstore.Load()
	}
	if c, err := cstore.Get(name); c != nil && err == nil {
		return c.ID, nil
	}

	return "", ErrLayerUnknown
}

func (s *store) DeleteLayer(id string) error {
	rlstore, err := s.LayerStore()
	if err != nil {
		return err
	}
	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rlstore.Exists(id) {
		if l, err := rlstore.Get(id); err != nil {
			id = l.ID
		}
		layers, err := rlstore.Layers()
		if err != nil {
			return err
		}
		for _, layer := range layers {
			if layer.Parent == id {
				return ErrLayerHasChildren
			}
		}
		images, err := ristore.Images()
		if err != nil {
			return err
		}
		for _, image := range images {
			if image.TopLayer == id || stringutils.InSlice(image.MappedTopLayers, id) {
				return errors.Wrapf(ErrLayerUsedByImage, "Layer %v used by image %v", id, image.ID)
			}
		}
		containers, err := rcstore.Containers()
		if err != nil {
			return err
		}
		for _, container := range containers {
			if container.LayerID == id {
				return errors.Wrapf(ErrLayerUsedByContainer, "Layer %v used by container %v", id, container.ID)
			}
		}
		return rlstore.Delete(id)
	}
	return ErrNotALayer
}

func (s *store) DeleteImage(id string, commit bool) (layers []string, err error) {
	rlstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	ristore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	layersToRemove := []string{}
	if ristore.Exists(id) {
		image, err := ristore.Get(id)
		if err != nil {
			return nil, err
		}
		id = image.ID
		containers, err := rcstore.Containers()
		if err != nil {
			return nil, err
		}
		aContainerByImage := make(map[string]string)
		for _, container := range containers {
			aContainerByImage[container.ImageID] = container.ID
		}
		if container, ok := aContainerByImage[id]; ok {
			return nil, errors.Wrapf(ErrImageUsedByContainer, "Image used by %v", container)
		}
		images, err := ristore.Images()
		if err != nil {
			return nil, err
		}
		layers, err := rlstore.Layers()
		if err != nil {
			return nil, err
		}
		childrenByParent := make(map[string]*[]string)
		for _, layer := range layers {
			parent := layer.Parent
			if list, ok := childrenByParent[parent]; ok {
				newList := append(*list, layer.ID)
				childrenByParent[parent] = &newList
			} else {
				childrenByParent[parent] = &([]string{layer.ID})
			}
		}
		otherImagesByTopLayer := make(map[string]string)
		for _, img := range images {
			if img.ID != id {
				otherImagesByTopLayer[img.TopLayer] = img.ID
				for _, layerID := range img.MappedTopLayers {
					otherImagesByTopLayer[layerID] = img.ID
				}
			}
		}
		if commit {
			if err = ristore.Delete(id); err != nil {
				return nil, err
			}
		}
		layer := image.TopLayer
		lastRemoved := ""
		for layer != "" {
			if rcstore.Exists(layer) {
				break
			}
			if _, ok := otherImagesByTopLayer[layer]; ok {
				break
			}
			parent := ""
			if l, err := rlstore.Get(layer); err == nil {
				parent = l.Parent
			}
			hasOtherRefs := func() bool {
				layersToCheck := []string{layer}
				if layer == image.TopLayer {
					layersToCheck = append(layersToCheck, image.MappedTopLayers...)
				}
				for _, layer := range layersToCheck {
					if childList, ok := childrenByParent[layer]; ok && childList != nil {
						children := *childList
						for _, child := range children {
							if child != lastRemoved {
								return true
							}
						}
					}
				}
				return false
			}
			if hasOtherRefs() {
				break
			}
			lastRemoved = layer
			layersToRemove = append(layersToRemove, lastRemoved)
			if layer == image.TopLayer {
				layersToRemove = append(layersToRemove, image.MappedTopLayers...)
			}
			layer = parent
		}
	} else {
		return nil, ErrNotAnImage
	}
	if commit {
		for _, layer := range layersToRemove {
			if err = rlstore.Delete(layer); err != nil {
				return nil, err
			}
		}
	}
	return layersToRemove, nil
}

func (s *store) DeleteContainer(id string) error {
	rlstore, err := s.LayerStore()
	if err != nil {
		return err
	}
	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rcstore.Exists(id) {
		if container, err := rcstore.Get(id); err == nil {
			if rlstore.Exists(container.LayerID) {
				if err = rlstore.Delete(container.LayerID); err != nil {
					return err
				}
			}
			if err = rcstore.Delete(id); err != nil {
				return err
			}
			middleDir := s.graphDriverName + "-containers"
			gcpath := filepath.Join(s.GraphRoot(), middleDir, container.ID)
			if err = os.RemoveAll(gcpath); err != nil {
				return err
			}
			rcpath := filepath.Join(s.RunRoot(), middleDir, container.ID)
			if err = os.RemoveAll(rcpath); err != nil {
				return err
			}
			return nil
		}
	}
	return ErrNotAContainer
}

func (s *store) Delete(id string) error {
	rlstore, err := s.LayerStore()
	if err != nil {
		return err
	}
	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if rcstore.Exists(id) {
		if container, err := rcstore.Get(id); err == nil {
			if rlstore.Exists(container.LayerID) {
				if err = rlstore.Delete(container.LayerID); err != nil {
					return err
				}
				if err = rcstore.Delete(id); err != nil {
					return err
				}
				middleDir := s.graphDriverName + "-containers"
				gcpath := filepath.Join(s.GraphRoot(), middleDir, container.ID, "userdata")
				if err = os.RemoveAll(gcpath); err != nil {
					return err
				}
				rcpath := filepath.Join(s.RunRoot(), middleDir, container.ID, "userdata")
				if err = os.RemoveAll(rcpath); err != nil {
					return err
				}
				return nil
			}
			return ErrNotALayer
		}
	}
	if ristore.Exists(id) {
		return ristore.Delete(id)
	}
	if rlstore.Exists(id) {
		return rlstore.Delete(id)
	}
	return ErrLayerUnknown
}

func (s *store) Wipe() error {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return err
	}
	ristore, err := s.ImageStore()
	if err != nil {
		return err
	}
	rlstore, err := s.LayerStore()
	if err != nil {
		return err
	}

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	ristore.Lock()
	defer ristore.Unlock()
	if modified, err := ristore.Modified(); modified || err != nil {
		ristore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	if err = rcstore.Wipe(); err != nil {
		return err
	}
	if err = ristore.Wipe(); err != nil {
		return err
	}
	return rlstore.Wipe()
}

func (s *store) Status() ([][2]string, error) {
	rlstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	return rlstore.Status()
}

func (s *store) Version() ([][2]string, error) {
	return [][2]string{}, nil
}

func (s *store) Mount(id, mountLabel string) (string, error) {
	container, err := s.Container(id)
	var (
		uidMap, gidMap []idtools.IDMap
		mountOpts      []string
	)
	if err == nil {
		uidMap, gidMap = container.UIDMap, container.GIDMap
		id = container.LayerID
		mountOpts = container.MountOpts()
	}
	rlstore, err := s.LayerStore()
	if err != nil {
		return "", err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	if rlstore.Exists(id) {
		options := drivers.MountOpts{
			MountLabel: mountLabel,
			UidMaps:    uidMap,
			GidMaps:    gidMap,
			Options:    mountOpts,
		}
		return rlstore.Mount(id, options)
	}
	return "", ErrLayerUnknown
}

func (s *store) Mounted(id string) (int, error) {
	if layerID, err := s.ContainerLayerID(id); err == nil {
		id = layerID
	}
	rlstore, err := s.LayerStore()
	if err != nil {
		return 0, err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	return rlstore.Mounted(id)
}

func (s *store) Unmount(id string, force bool) (bool, error) {
	if layerID, err := s.ContainerLayerID(id); err == nil {
		id = layerID
	}
	rlstore, err := s.LayerStore()
	if err != nil {
		return false, err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	if rlstore.Exists(id) {
		return rlstore.Unmount(id, force)
	}
	return false, ErrLayerUnknown
}

func (s *store) Changes(from, to string) ([]archive.Change, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(to) {
			return store.Changes(from, to)
		}
	}
	return nil, ErrLayerUnknown
}

func (s *store) DiffSize(from, to string) (int64, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return -1, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return -1, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(to) {
			return store.DiffSize(from, to)
		}
	}
	return -1, ErrLayerUnknown
}

func (s *store) Diff(from, to string, options *DiffOptions) (io.ReadCloser, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(to) {
			rc, err := store.Diff(from, to, options)
			if rc != nil && err == nil {
				wrapped := ioutils.NewReadCloserWrapper(rc, func() error {
					err := rc.Close()
					store.Unlock()
					return err
				})
				return wrapped, nil
			}
			store.Unlock()
			return rc, err
		}
		store.Unlock()
	}
	return nil, ErrLayerUnknown
}

func (s *store) ApplyDiff(to string, diff io.Reader) (int64, error) {
	rlstore, err := s.LayerStore()
	if err != nil {
		return -1, err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	if rlstore.Exists(to) {
		return rlstore.ApplyDiff(to, diff)
	}
	return -1, ErrLayerUnknown
}

func (s *store) layersByMappedDigest(m func(ROLayerStore, digest.Digest) ([]Layer, error), d digest.Digest) ([]Layer, error) {
	var layers []Layer
	lstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}

	lstores, err := s.ROLayerStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		storeLayers, err := m(store, d)
		if err != nil {
			if errors.Cause(err) != ErrLayerUnknown {
				return nil, err
			}
			continue
		}
		layers = append(layers, storeLayers...)
	}
	if len(layers) == 0 {
		return nil, ErrLayerUnknown
	}
	return layers, nil
}

func (s *store) LayersByCompressedDigest(d digest.Digest) ([]Layer, error) {
	if err := d.Validate(); err != nil {
		return nil, errors.Wrapf(err, "error looking for compressed layers matching digest %q", d)
	}
	return s.layersByMappedDigest(func(r ROLayerStore, d digest.Digest) ([]Layer, error) { return r.LayersByCompressedDigest(d) }, d)
}

func (s *store) LayersByUncompressedDigest(d digest.Digest) ([]Layer, error) {
	if err := d.Validate(); err != nil {
		return nil, errors.Wrapf(err, "error looking for layers matching digest %q", d)
	}
	return s.layersByMappedDigest(func(r ROLayerStore, d digest.Digest) ([]Layer, error) { return r.LayersByUncompressedDigest(d) }, d)
}

func (s *store) LayerSize(id string) (int64, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return -1, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return -1, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		if store.Exists(id) {
			return store.Size(id)
		}
	}
	return -1, ErrLayerUnknown
}

func (s *store) LayerParentOwners(id string) ([]int, []int, error) {
	rlstore, err := s.LayerStore()
	if err != nil {
		return nil, nil, err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	if rlstore.Exists(id) {
		return rlstore.ParentOwners(id)
	}
	return nil, nil, ErrLayerUnknown
}

func (s *store) ContainerParentOwners(id string) ([]int, []int, error) {
	rlstore, err := s.LayerStore()
	if err != nil {
		return nil, nil, err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, nil, err
	}
	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	container, err := rcstore.Get(id)
	if err != nil {
		return nil, nil, err
	}
	if rlstore.Exists(container.LayerID) {
		return rlstore.ParentOwners(container.LayerID)
	}
	return nil, nil, ErrLayerUnknown
}

func (s *store) Layers() ([]Layer, error) {
	var layers []Layer
	lstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}

	lstores, err := s.ROLayerStores()
	if err != nil {
		return nil, err
	}

	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		storeLayers, err := store.Layers()
		if err != nil {
			return nil, err
		}
		layers = append(layers, storeLayers...)
	}
	return layers, nil
}

func (s *store) Images() ([]Image, error) {
	var images []Image
	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}

	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		storeImages, err := store.Images()
		if err != nil {
			return nil, err
		}
		images = append(images, storeImages...)
	}
	return images, nil
}

func (s *store) Containers() ([]Container, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}

	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	return rcstore.Containers()
}

func (s *store) Layer(id string) (*Layer, error) {
	lstore, err := s.LayerStore()
	if err != nil {
		return nil, err
	}
	lstores, err := s.ROLayerStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROLayerStore{lstore}, lstores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		layer, err := store.Get(id)
		if err == nil {
			return layer, nil
		}
	}
	return nil, ErrLayerUnknown
}

func (s *store) Image(id string) (*Image, error) {
	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}
	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		image, err := store.Get(id)
		if err == nil {
			return image, nil
		}
	}
	return nil, ErrImageUnknown
}

func (s *store) ImagesByTopLayer(id string) ([]*Image, error) {
	images := []*Image{}
	layer, err := s.Layer(id)
	if err != nil {
		return nil, err
	}

	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}

	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		imageList, err := store.Images()
		if err != nil {
			return nil, err
		}
		for _, image := range imageList {
			if image.TopLayer == layer.ID || stringutils.InSlice(image.MappedTopLayers, layer.ID) {
				images = append(images, &image)
			}
		}
	}
	return images, nil
}

func (s *store) ImagesByDigest(d digest.Digest) ([]*Image, error) {
	images := []*Image{}

	istore, err := s.ImageStore()
	if err != nil {
		return nil, err
	}

	istores, err := s.ROImageStores()
	if err != nil {
		return nil, err
	}
	for _, store := range append([]ROImageStore{istore}, istores...) {
		store.Lock()
		defer store.Unlock()
		if modified, err := store.Modified(); modified || err != nil {
			store.Load()
		}
		imageList, err := store.ByDigest(d)
		if err != nil && err != ErrImageUnknown {
			return nil, err
		}
		images = append(images, imageList...)
	}
	return images, nil
}

func (s *store) Container(id string) (*Container, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	return rcstore.Get(id)
}

func (s *store) ContainerLayerID(id string) (string, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return "", err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	container, err := rcstore.Get(id)
	if err != nil {
		return "", err
	}
	return container.LayerID, nil
}

func (s *store) ContainerByLayer(id string) (*Container, error) {
	layer, err := s.Layer(id)
	if err != nil {
		return nil, err
	}
	rcstore, err := s.ContainerStore()
	if err != nil {
		return nil, err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}
	containerList, err := rcstore.Containers()
	if err != nil {
		return nil, err
	}
	for _, container := range containerList {
		if container.LayerID == layer.ID {
			return &container, nil
		}
	}

	return nil, ErrContainerUnknown
}

func (s *store) ContainerDirectory(id string) (string, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return "", err
	}
	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	id, err = rcstore.Lookup(id)
	if err != nil {
		return "", err
	}

	middleDir := s.graphDriverName + "-containers"
	gcpath := filepath.Join(s.GraphRoot(), middleDir, id, "userdata")
	if err := os.MkdirAll(gcpath, 0700); err != nil {
		return "", err
	}
	return gcpath, nil
}

func (s *store) ContainerRunDirectory(id string) (string, error) {
	rcstore, err := s.ContainerStore()
	if err != nil {
		return "", err
	}

	rcstore.Lock()
	defer rcstore.Unlock()
	if modified, err := rcstore.Modified(); modified || err != nil {
		rcstore.Load()
	}

	id, err = rcstore.Lookup(id)
	if err != nil {
		return "", err
	}

	middleDir := s.graphDriverName + "-containers"
	rcpath := filepath.Join(s.RunRoot(), middleDir, id, "userdata")
	if err := os.MkdirAll(rcpath, 0700); err != nil {
		return "", err
	}
	return rcpath, nil
}

func (s *store) SetContainerDirectoryFile(id, file string, data []byte) error {
	dir, err := s.ContainerDirectory(id)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0700)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(dir, file), data, 0600)
}

func (s *store) FromContainerDirectory(id, file string) ([]byte, error) {
	dir, err := s.ContainerDirectory(id)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(filepath.Join(dir, file))
}

func (s *store) SetContainerRunDirectoryFile(id, file string, data []byte) error {
	dir, err := s.ContainerRunDirectory(id)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(filepath.Join(dir, file)), 0700)
	if err != nil {
		return err
	}
	return ioutils.AtomicWriteFile(filepath.Join(dir, file), data, 0600)
}

func (s *store) FromContainerRunDirectory(id, file string) ([]byte, error) {
	dir, err := s.ContainerRunDirectory(id)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(filepath.Join(dir, file))
}

func (s *store) Shutdown(force bool) ([]string, error) {
	mounted := []string{}
	modified := false

	rlstore, err := s.LayerStore()
	if err != nil {
		return mounted, err
	}

	s.graphLock.Lock()
	defer s.graphLock.Unlock()

	rlstore.Lock()
	defer rlstore.Unlock()
	if modified, err := rlstore.Modified(); modified || err != nil {
		rlstore.Load()
	}

	layers, err := rlstore.Layers()
	if err != nil {
		return mounted, err
	}
	for _, layer := range layers {
		if layer.MountCount == 0 {
			continue
		}
		mounted = append(mounted, layer.ID)
		if force {
			for layer.MountCount > 0 {
				_, err2 := rlstore.Unmount(layer.ID, force)
				if err2 != nil {
					if err == nil {
						err = err2
					}
					break
				}
				modified = true
			}
		}
	}
	if len(mounted) > 0 && err == nil {
		err = errors.Wrap(ErrLayerUsedByContainer, "A layer is mounted")
	}
	if err == nil {
		err = s.graphDriver.Cleanup()
		s.graphLock.Touch()
		modified = true
	}
	if modified {
		rlstore.Touch()
	}
	return mounted, err
}

// Convert a BigData key name into an acceptable file name.
func makeBigDataBaseName(key string) string {
	reader := strings.NewReader(key)
	for reader.Len() > 0 {
		ch, size, err := reader.ReadRune()
		if err != nil || size != 1 {
			break
		}
		if ch != '.' && !(ch >= '0' && ch <= '9') && !(ch >= 'a' && ch <= 'z') {
			break
		}
	}
	if reader.Len() > 0 {
		return "=" + base64.StdEncoding.EncodeToString([]byte(key))
	}
	return key
}

func stringSliceWithoutValue(slice []string, value string) []string {
	modified := make([]string, 0, len(slice))
	for _, v := range slice {
		if v == value {
			continue
		}
		modified = append(modified, v)
	}
	return modified
}

func copyStringSlice(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}
	ret := make([]string, len(slice))
	copy(ret, slice)
	return ret
}

func copyStringInt64Map(m map[string]int64) map[string]int64 {
	ret := make(map[string]int64, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

func copyStringDigestMap(m map[string]digest.Digest) map[string]digest.Digest {
	ret := make(map[string]digest.Digest, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

// copyStringInterfaceMap still forces us to assume that the interface{} is
// a non-pointer scalar value
func copyStringInterfaceMap(m map[string]interface{}) map[string]interface{} {
	ret := make(map[string]interface{}, len(m))
	for k, v := range m {
		ret[k] = v
	}
	return ret
}

const defaultConfigFile = "/etc/containers/storage.conf"

// ThinpoolOptionsConfig represents the "storage.options.thinpool"
// TOML config table.
type ThinpoolOptionsConfig struct {
	// AutoExtendPercent determines the amount by which pool needs to be
	// grown. This is specified in terms of % of pool size. So a value of
	// 20 means that when threshold is hit, pool will be grown by 20% of
	// existing pool size.
	AutoExtendPercent string `toml:"autoextend_percent"`

	// AutoExtendThreshold determines the pool extension threshold in terms
	// of percentage of pool size. For example, if threshold is 60, that
	// means when pool is 60% full, threshold has been hit.
	AutoExtendThreshold string `toml:"autoextend_threshold"`

	// BaseSize specifies the size to use when creating the base device,
	// which limits the size of images and containers.
	BaseSize string `toml:"basesize"`

	// BlockSize specifies a custom blocksize to use for the thin pool.
	BlockSize string `toml:"blocksize"`

	// DirectLvmDevice specifies a custom block storage device to use for
	// the thin pool.
	DirectLvmDevice string `toml:"directlvm_device"`

	// DirectLvmDeviceForcewipes device even if device already has a
	// filesystem
	DirectLvmDeviceForce string `toml:"directlvm_device_force"`

	// Fs specifies the filesystem type to use for the base device.
	Fs string `toml:"fs"`

	// log_level sets the log level of devicemapper.
	LogLevel string `toml:"log_level"`

	// MinFreeSpace specifies the min free space percent in a thin pool
	// require for new device creation to
	MinFreeSpace string `toml:"min_free_space"`

	// MkfsArg specifies extra mkfs arguments to be used when creating the
	// basedevice.
	MkfsArg string `toml:"mkfsarg"`

	// MountOpt specifies extra mount options used when mounting the thin
	// devices.
	MountOpt string `toml:"mountopt"`

	// UseDeferredDeletion marks device for deferred deletion
	UseDeferredDeletion string `toml:"use_deferred_deletion"`

	// UseDeferredRemoval marks device for deferred removal
	UseDeferredRemoval string `toml:"use_deferred_removal"`

	// XfsNoSpaceMaxRetriesFreeSpace specifies the maximum number of
	// retries XFS should attempt to complete IO when ENOSPC (no space)
	// error is returned by underlying storage device.
	XfsNoSpaceMaxRetries string `toml:"xfs_nospace_max_retries"`
}

// OptionsConfig represents the "storage.options" TOML config table.
type OptionsConfig struct {
	// AdditionalImagesStores is the location of additional read/only
	// Image stores.  Usually used to access Networked File System
	// for shared image content
	AdditionalImageStores []string `toml:"additionalimagestores"`

	// Size
	Size string `toml:"size"`

	// OverrideKernelCheck
	OverrideKernelCheck string `toml:"override_kernel_check"`

	// RemapUIDs is a list of default UID mappings to use for layers.
	RemapUIDs string `toml:"remap-uids"`
	// RemapGIDs is a list of default GID mappings to use for layers.
	RemapGIDs string `toml:"remap-gids"`

	// RemapUser is the name of one or more entries in /etc/subuid which
	// should be used to set up default UID mappings.
	RemapUser string `toml:"remap-user"`
	// RemapGroup is the name of one or more entries in /etc/subgid which
	// should be used to set up default GID mappings.
	RemapGroup string `toml:"remap-group"`
	// Thinpool container options to be handed to thinpool drivers
	Thinpool struct{ ThinpoolOptionsConfig } `toml:"thinpool"`
	// OSTree repository
	OstreeRepo string `toml:"ostree_repo"`

	// Do not create a bind mount on the storage home
	SkipMountHome string `toml:"skip_mount_home"`

	// Alternative program to use for the mount of the file system
	MountProgram string `toml:"mount_program"`

	// MountOpt specifies extra mount options used when mounting
	MountOpt string `toml:"mountopt"`
}

// TOML-friendly explicit tables used for conversions.
type tomlConfig struct {
	Storage struct {
		Driver    string                  `toml:"driver"`
		RunRoot   string                  `toml:"runroot"`
		GraphRoot string                  `toml:"graphroot"`
		Options   struct{ OptionsConfig } `toml:"options"`
	} `toml:"storage"`
}

// ReloadConfigurationFile parses the specified configuration file and overrides
// the configuration in storeOptions.
func ReloadConfigurationFile(configFile string, storeOptions *StoreOptions) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Failed to read %s %v\n", configFile, err.Error())
			return
		}
	}

	config := new(tomlConfig)

	if _, err := toml.Decode(string(data), config); err != nil {
		fmt.Printf("Failed to parse %s %v\n", configFile, err.Error())
		return
	}
	if config.Storage.Driver != "" {
		storeOptions.GraphDriverName = config.Storage.Driver
	}
	if config.Storage.RunRoot != "" {
		storeOptions.RunRoot = config.Storage.RunRoot
	}
	if config.Storage.GraphRoot != "" {
		storeOptions.GraphRoot = config.Storage.GraphRoot
	}
	if config.Storage.Options.Thinpool.AutoExtendPercent != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.thinp_autoextend_percent=%s", config.Storage.Options.Thinpool.AutoExtendPercent))
	}

	if config.Storage.Options.Thinpool.AutoExtendThreshold != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.thinp_autoextend_threshold=%s", config.Storage.Options.Thinpool.AutoExtendThreshold))
	}

	if config.Storage.Options.Thinpool.BaseSize != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.basesize=%s", config.Storage.Options.Thinpool.BaseSize))
	}
	if config.Storage.Options.Thinpool.BlockSize != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.blocksize=%s", config.Storage.Options.Thinpool.BlockSize))
	}
	if config.Storage.Options.Thinpool.DirectLvmDevice != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.directlvm_device=%s", config.Storage.Options.Thinpool.DirectLvmDevice))
	}
	if config.Storage.Options.Thinpool.DirectLvmDeviceForce != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.directlvm_device_force=%s", config.Storage.Options.Thinpool.DirectLvmDeviceForce))
	}
	if config.Storage.Options.Thinpool.Fs != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.fs=%s", config.Storage.Options.Thinpool.Fs))
	}
	if config.Storage.Options.Thinpool.LogLevel != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.libdm_log_level=%s", config.Storage.Options.Thinpool.LogLevel))
	}
	if config.Storage.Options.Thinpool.MinFreeSpace != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.min_free_space=%s", config.Storage.Options.Thinpool.MinFreeSpace))
	}
	if config.Storage.Options.Thinpool.MkfsArg != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.mkfsarg=%s", config.Storage.Options.Thinpool.MkfsArg))
	}
	if config.Storage.Options.Thinpool.MountOpt != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.mountopt=%s", config.Storage.Driver, config.Storage.Options.Thinpool.MountOpt))
	}
	if config.Storage.Options.Thinpool.UseDeferredDeletion != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.use_deferred_deletion=%s", config.Storage.Options.Thinpool.UseDeferredDeletion))
	}
	if config.Storage.Options.Thinpool.UseDeferredRemoval != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.use_deferred_removal=%s", config.Storage.Options.Thinpool.UseDeferredRemoval))
	}
	if config.Storage.Options.Thinpool.XfsNoSpaceMaxRetries != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.xfs_nospace_max_retries=%s", config.Storage.Options.Thinpool.XfsNoSpaceMaxRetries))
	}
	for _, s := range config.Storage.Options.AdditionalImageStores {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.imagestore=%s", config.Storage.Driver, s))
	}
	if config.Storage.Options.Size != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.size=%s", config.Storage.Driver, config.Storage.Options.Size))
	}
	if config.Storage.Options.OstreeRepo != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.ostree_repo=%s", config.Storage.Driver, config.Storage.Options.OstreeRepo))
	}
	if config.Storage.Options.SkipMountHome != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.skip_mount_home=%s", config.Storage.Driver, config.Storage.Options.SkipMountHome))
	}
	if config.Storage.Options.MountProgram != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.mount_program=%s", config.Storage.Driver, config.Storage.Options.MountProgram))
	}
	if config.Storage.Options.MountOpt != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.mountopt=%s", config.Storage.Driver, config.Storage.Options.MountOpt))
	}
	if config.Storage.Options.OverrideKernelCheck != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.override_kernel_check=%s", config.Storage.Driver, config.Storage.Options.OverrideKernelCheck))
	}
	if config.Storage.Options.RemapUser != "" && config.Storage.Options.RemapGroup == "" {
		config.Storage.Options.RemapGroup = config.Storage.Options.RemapUser
	}
	if config.Storage.Options.RemapGroup != "" && config.Storage.Options.RemapUser == "" {
		config.Storage.Options.RemapUser = config.Storage.Options.RemapGroup
	}
	if config.Storage.Options.RemapUser != "" && config.Storage.Options.RemapGroup != "" {
		mappings, err := idtools.NewIDMappings(config.Storage.Options.RemapUser, config.Storage.Options.RemapGroup)
		if err != nil {
			fmt.Printf("Error initializing ID mappings for %s:%s %v\n", config.Storage.Options.RemapUser, config.Storage.Options.RemapGroup, err)
			return
		}
		storeOptions.UIDMap = mappings.UIDs()
		storeOptions.GIDMap = mappings.GIDs()
	}

	uidmap, err := idtools.ParseIDMap([]string{config.Storage.Options.RemapUIDs}, "remap-uids")
	if err != nil {
		fmt.Print(err)
	} else {
		storeOptions.UIDMap = append(storeOptions.UIDMap, uidmap...)
	}
	gidmap, err := idtools.ParseIDMap([]string{config.Storage.Options.RemapGIDs}, "remap-gids")
	if err != nil {
		fmt.Print(err)
	} else {
		storeOptions.GIDMap = append(storeOptions.GIDMap, gidmap...)
	}
	if os.Getenv("STORAGE_DRIVER") != "" {
		storeOptions.GraphDriverName = os.Getenv("STORAGE_DRIVER")
	}
	if os.Getenv("STORAGE_OPTS") != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, strings.Split(os.Getenv("STORAGE_OPTS"), ",")...)
	}
	if len(storeOptions.GraphDriverOptions) == 1 && storeOptions.GraphDriverOptions[0] == "" {
		storeOptions.GraphDriverOptions = nil
	}
}

func init() {
	DefaultStoreOptions.RunRoot = "/var/run/containers/storage"
	DefaultStoreOptions.GraphRoot = "/var/lib/containers/storage"
	DefaultStoreOptions.GraphDriverName = ""

	ReloadConfigurationFile(defaultConfigFile, &DefaultStoreOptions)
}

func GetDefaultMountOptions() ([]string, error) {
	mountOpts := []string{
		".mountopt",
		fmt.Sprintf("%s.mountopt", DefaultStoreOptions.GraphDriverName),
	}
	for _, option := range DefaultStoreOptions.GraphDriverOptions {
		key, val, err := parsers.ParseKeyValueOpt(option)
		if err != nil {
			return nil, err
		}
		key = strings.ToLower(key)
		for _, m := range mountOpts {
			if m == key {
				return strings.Split(val, ","), nil
			}
		}
	}
	return nil, nil
}
