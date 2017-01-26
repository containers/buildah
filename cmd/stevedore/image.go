package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/image"
	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/storage"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

type containerImageRef struct {
	store     storage.Store
	container *storage.Container
	name      reference.Named
	config    []byte
}

type containerImageSource struct {
	path         string
	ref          *containerImageRef
	store        storage.Store
	container    *storage.Container
	config       []byte
	configDigest digest.Digest
	manifest     []byte
}

func (i *containerImageRef) NewImage(sc *types.SystemContext) (types.Image, error) {
	src, err := i.NewImageSource(sc, nil)
	if err != nil {
		return nil, err
	}
	return image.FromSource(src)
}

func (i *containerImageRef) NewImageSource(sc *types.SystemContext, manifestTypes []string) (src types.ImageSource, err error) {
	if len(manifestTypes) > 0 {
		ok := false
		for _, mt := range manifestTypes {
			if mt == v1.MediaTypeImageManifest {
				ok = true
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("no supported manifest types")
		}
	}
	layers := []string{}
	layerID := i.container.LayerID
	layer, err := i.store.GetLayer(layerID)
	if err != nil {
		return nil, fmt.Errorf("unable to read layer %q: %v", layerID, err)
	}
	for layer != nil {
		layers = append(append([]string{}, layerID), layers...)
		layerID = layer.Parent
		if layerID == "" {
			err = nil
			break
		}
		layer, err = i.store.GetLayer(layerID)
		if err != nil {
			return nil, fmt.Errorf("unable to read layer %q: %v", layerID, err)
		}
	}

	path, err := ioutil.TempDir(os.TempDir(), "stevedore")
	if err != nil {
		return nil, err
	}
	defer func() {
		if src == nil {
			err2 := os.RemoveAll(path)
			if err2 != nil {
				logrus.Errorf("error removing %q: %v", path, err)
			}
		}
	}()

	manifest := v1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: v1.Descriptor{
			MediaType: v1.MediaTypeImageConfig,
			Digest:    digest.FromBytes(i.config),
			Size:      int64(len(i.config)),
		},
		Layers: []v1.Descriptor{},
	}

	for _, layerID := range layers {
		rc, err := i.store.Diff("", layerID)
		if err != nil {
			return nil, fmt.Errorf("error extracting layer %q: %v", layerID, err)
		}
		defer rc.Close()
		uncompressed, err := archive.DecompressStream(rc)
		if err != nil {
			return nil, fmt.Errorf("error decompressing layer %q: %v", layerID, err)
		}
		defer uncompressed.Close()
		hasher := digest.Canonical.Digester()
		reader := io.TeeReader(uncompressed, hasher.Hash())
		layerFile, err := os.OpenFile(filepath.Join(path, "layer"), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("error opening file for layer %q: %v", layerID, err)
		}
		size, err := io.Copy(layerFile, reader)
		if err != nil {
			return nil, fmt.Errorf("error storing layer %q to file: %v", layerID, err)
		}
		layerFile.Close()
		err = os.Rename(filepath.Join(path, "layer"), filepath.Join(path, hasher.Digest().String()))
		layerDescriptor := v1.Descriptor{
			MediaType: v1.MediaTypeImageLayer,
			Digest:    hasher.Digest(),
			Size:      size,
		}
		manifest.Layers = append(manifest.Layers, layerDescriptor)
	}

	mfest, err := json.Marshal(&manifest)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("manifest = %s\n", mfest)

	src = &containerImageSource{
		path:         path,
		ref:          i,
		store:        i.store,
		container:    i.container,
		manifest:     mfest,
		config:       i.config,
		configDigest: manifest.Config.Digest,
	}
	return src, nil
}

func (i *containerImageRef) NewImageDestination(sc *types.SystemContext) (types.ImageDestination, error) {
	return nil, fmt.Errorf("can't write to a container")
}

func (i *containerImageRef) DockerReference() reference.Named {
	return i.name
}

func (i *containerImageRef) StringWithinTransport() string {
	if len(i.container.Names) > 0 {
		return i.container.Names[0]
	}
	return ""
}

func (i *containerImageRef) DeleteImage(*types.SystemContext) error {
	// we were never here
	return nil
}

func (i *containerImageRef) PolicyConfigurationIdentity() string {
	return ""
}

func (i *containerImageRef) PolicyConfigurationNamespaces() []string {
	return nil
}

func (i *containerImageRef) Transport() types.ImageTransport {
	return is.Transport
}

func (i *containerImageSource) Close() {
	err := os.RemoveAll(i.path)
	if err != nil {
		logrus.Errorf("error removing %q: %v", i.path, err)
	}
}

func (i *containerImageSource) Reference() types.ImageReference {
	return i.ref
}

func (i *containerImageSource) GetSignatures() ([][]byte, error) {
	return nil, nil
}

func (i *containerImageSource) GetTargetManifest(digest digest.Digest) ([]byte, string, error) {
	return []byte{}, "", fmt.Errorf("TODO")
}

func (i *containerImageSource) GetManifest() ([]byte, string, error) {
	return i.manifest, v1.MediaTypeImageManifest, nil
}

func (i *containerImageSource) GetBlob(blob types.BlobInfo) (reader io.ReadCloser, size int64, err error) {
	if blob.Digest == i.configDigest {
		logrus.Debugf("start reading config")
		reader := bytes.NewReader(i.config)
		closer := func() error {
			logrus.Debugf("finished reading config")
			return nil
		}
		return ioutils.NewReadCloserWrapper(reader, closer), reader.Size(), nil
	}
	layerFile, err := os.OpenFile(filepath.Join(i.path, blob.Digest.String()), os.O_RDONLY, 0600)
	if err != nil {
		logrus.Debugf("error reading layer %q: %v", blob.Digest.String(), err)
		return nil, -1, err
	}
	size = -1
	st, err := layerFile.Stat()
	if err != nil {
		logrus.Warnf("error reading size of layer %q: %v", blob.Digest.String(), err)
	} else {
		size = st.Size()
	}
	logrus.Debugf("reading layer %q", blob.Digest.String())
	return layerFile, size, nil
}

func makeContainerImageRef(store storage.Store, container *storage.Container, config string) types.ImageReference {
	var err error
	var name reference.Named
	if len(container.Names) > 0 {
		name, err = reference.ParseNamed(container.Names[0])
		if err != nil {
			name = nil
		}
	}
	return &containerImageRef{
		store:     store,
		container: container,
		name:      name,
		config:    []byte(config),
	}
}
