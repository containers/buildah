package buildah

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

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
	store       storage.Store
	container   *storage.Container
	compression archive.Compression
	name        reference.Named
	config      []byte
	createdBy   string
	annotations map[string]string
}

type containerImageSource struct {
	path         string
	ref          *containerImageRef
	store        storage.Store
	container    *storage.Container
	compression  archive.Compression
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
	logrus.Debugf("layer list: %q", layers)

	created := time.Now().UTC()

	path, err := ioutil.TempDir(os.TempDir(), Package)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("using %q to hold temporary data", path)
	defer func() {
		if src == nil {
			err2 := os.RemoveAll(path)
			if err2 != nil {
				logrus.Errorf("error removing %q: %v", path, err)
			}
		}
	}()

	image := v1.Image{}
	err = json.Unmarshal(i.config, &image)
	if err != nil {
		return nil, err
	}

	manifest := v1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: v1.Descriptor{
			MediaType: v1.MediaTypeImageConfig,
		},
		Layers:      []v1.Descriptor{},
		Annotations: i.annotations,
	}

	image.RootFS.Type = "layers"
	image.RootFS.DiffIDs = []string{}

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
		srcHasher := digest.Canonical.Digester()
		reader := io.TeeReader(uncompressed, srcHasher.Hash())
		layerFile, err := os.OpenFile(filepath.Join(path, "layer"), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("error opening file for layer %q: %v", layerID, err)
		}
		destHasher := digest.Canonical.Digester()
		counter := ioutils.NewWriteCounter(layerFile)
		multiWriter := io.MultiWriter(counter, destHasher.Hash())
		mediaType := v1.MediaTypeImageLayer
		if i.compression != archive.Uncompressed {
			switch i.compression {
			case archive.Gzip:
				mediaType = v1.MediaTypeImageLayerGzip
				logrus.Debugf("compressing layer %q with gzip", layerID)
			case archive.Bzip2:
				logrus.Debugf("compressing layer %q with bzip2", layerID)
			default:
				logrus.Debugf("compressing layer %q with unknown compressor(?)", layerID)
			}
		}
		compressor, err := archive.CompressStream(multiWriter, i.compression)
		if err != nil {
			return nil, fmt.Errorf("error compressing layer %q: %v", layerID, err)
		}
		size, err := io.Copy(compressor, reader)
		if err != nil {
			return nil, fmt.Errorf("error storing layer %q to file: %v", layerID, err)
		}
		compressor.Close()
		layerFile.Close()
		if i.compression == archive.Uncompressed {
			if size != counter.Count {
				return nil, fmt.Errorf("error storing layer %q to file: inconsistent layer size (copied %d, wrote %d)", layerID, size, counter.Count)
			}
		} else {
			size = counter.Count
		}
		logrus.Debugf("layer %q size is %d bytes", layerID, size)
		err = os.Rename(filepath.Join(path, "layer"), filepath.Join(path, destHasher.Digest().String()))
		if err != nil {
			return nil, fmt.Errorf("error storing layer %q to file: %v", layerID, err)
		}
		layerDescriptor := v1.Descriptor{
			MediaType: mediaType,
			Digest:    destHasher.Digest(),
			Size:      size,
		}
		manifest.Layers = append(manifest.Layers, layerDescriptor)
		lastLayerDiffID := destHasher.Digest().String()
		image.RootFS.DiffIDs = append(image.RootFS.DiffIDs, lastLayerDiffID)
	}

	news := v1.History{
		Created:    created,
		CreatedBy:  i.createdBy,
		Author:     image.Author,
		EmptyLayer: false,
	}
	image.History = append(image.History, news)

	config, err := json.Marshal(&image)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("config = %s\n", config)
	i.config = config

	manifest.Config.Digest = digest.FromBytes(config)
	manifest.Config.Size = int64(len(config))

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
		compression:  i.compression,
		manifest:     mfest,
		config:       i.config,
		configDigest: digest.FromBytes(config),
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

func (i *containerImageSource) Close() error {
	err := os.RemoveAll(i.path)
	if err != nil {
		logrus.Errorf("error removing %q: %v", i.path, err)
	}
	return err
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
	closer := func() error {
		layerFile.Close()
		logrus.Debugf("finished reading layer %q", blob.Digest.String())
		return nil
	}
	return ioutils.NewReadCloserWrapper(layerFile, closer), size, nil
}

func (b *Builder) makeContainerImageRef(compress archive.Compression) (types.ImageReference, error) {
	var name reference.Named
	container, err := b.store.GetContainer(b.ContainerID)
	if err != nil {
		return nil, err
	}
	if len(container.Names) > 0 {
		name, err = reference.ParseNamed(container.Names[0])
		if err != nil {
			name = nil
		}
	}
	ref := &containerImageRef{
		store:       b.store,
		container:   container,
		compression: compress,
		name:        name,
		config:      b.updatedConfig(),
		createdBy:   b.CreatedBy,
		annotations: b.Annotations,
	}
	return ref, nil
}
