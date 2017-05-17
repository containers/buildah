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
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/ioutils"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/projectatomic/buildah/docker"
)

type containerImageRef struct {
	store       storage.Store
	container   *storage.Container
	compression archive.Compression
	name        reference.Named
	oconfig     []byte
	dconfig     []byte
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
	manifestType string
}

func (i *containerImageRef) NewImage(sc *types.SystemContext) (types.Image, error) {
	src, err := i.NewImageSource(sc, nil)
	if err != nil {
		return nil, err
	}
	return image.FromSource(src)
}

func (i *containerImageRef) NewImageSource(sc *types.SystemContext, manifestTypes []string) (src types.ImageSource, err error) {
	manifestType := ""
	// Look for a supported format in the acceptable list.
	for _, mt := range manifestTypes {
		if mt == v1.MediaTypeImageManifest {
			manifestType = mt
			break
		}
		if mt == docker.V2S2MediaTypeManifest {
			manifestType = mt
			break
		}
	}
	// If it's not a format we support, return an error.
	if manifestType != v1.MediaTypeImageManifest && manifestType != docker.V2S2MediaTypeManifest {
		return nil, fmt.Errorf("no supported manifest types (attempted to use %q, only know %q and %q)",
			manifestType, v1.MediaTypeImageManifest, docker.V2S2MediaTypeManifest)
	}
	layers := []string{}
	layerID := i.container.LayerID
	layer, err := i.store.Layer(layerID)
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
		layer, err = i.store.Layer(layerID)
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

	oimage := v1.Image{}
	err = json.Unmarshal(i.oconfig, &oimage)
	if err != nil {
		return nil, err
	}
	dimage := docker.V2Image{}
	err = json.Unmarshal(i.dconfig, &dimage)
	if err != nil {
		return nil, err
	}

	omanifest := v1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: v1.Descriptor{
			MediaType: v1.MediaTypeImageConfig,
		},
		Layers:      []v1.Descriptor{},
		Annotations: i.annotations,
	}
	dmanifest := docker.V2S2Manifest{
		V2Versioned: docker.V2Versioned{
			SchemaVersion: 2,
			MediaType:     docker.V2S2MediaTypeManifest,
		},
		Config: docker.V2S2Descriptor{
			MediaType: docker.V2S2MediaTypeImageConfig,
		},
		Layers: []docker.V2S2Descriptor{},
	}

	oimage.RootFS.Type = docker.TypeLayers
	oimage.RootFS.DiffIDs = []string{}
	dimage.RootFS = &docker.V2S2RootFS{}
	dimage.RootFS.Type = docker.TypeLayers
	dimage.RootFS.DiffIDs = []digest.Digest{}

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
		omediaType := v1.MediaTypeImageLayer
		dmediaType := docker.V2S2MediaTypeUncompressedLayer
		if i.compression != archive.Uncompressed {
			switch i.compression {
			case archive.Gzip:
				omediaType = v1.MediaTypeImageLayerGzip
				dmediaType = docker.V2S2MediaTypeLayer
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
		olayerDescriptor := v1.Descriptor{
			MediaType: omediaType,
			Digest:    destHasher.Digest(),
			Size:      size,
		}
		omanifest.Layers = append(omanifest.Layers, olayerDescriptor)
		dlayerDescriptor := docker.V2S2Descriptor{
			MediaType: dmediaType,
			Digest:    destHasher.Digest(),
			Size:      size,
		}
		dmanifest.Layers = append(dmanifest.Layers, dlayerDescriptor)
		oimage.RootFS.DiffIDs = append(oimage.RootFS.DiffIDs, srcHasher.Digest().String())
		dimage.RootFS.DiffIDs = append(dimage.RootFS.DiffIDs, srcHasher.Digest())
	}

	onews := v1.History{
		Created:    created,
		CreatedBy:  i.createdBy,
		Author:     oimage.Author,
		EmptyLayer: false,
	}
	oimage.History = append(oimage.History, onews)
	dnews := docker.V2S2History{
		Created:    created,
		CreatedBy:  i.createdBy,
		Author:     dimage.Author,
		EmptyLayer: false,
	}
	dimage.History = append(dimage.History, dnews)

	oconfig, err := json.Marshal(&oimage)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("OCIv1 config = %s", oconfig)
	i.oconfig = oconfig

	omanifest.Config.Digest = digest.FromBytes(oconfig)
	omanifest.Config.Size = int64(len(oconfig))
	omanifest.Config.MediaType = v1.MediaTypeImageConfig

	omanifestbytes, err := json.Marshal(&omanifest)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("OCIv1 manifest = %s", omanifestbytes)

	dconfig, err := json.Marshal(&dimage)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Docker v2s2 config = %s", dconfig)
	i.dconfig = dconfig

	dmanifest.Config.Digest = digest.FromBytes(dconfig)
	dmanifest.Config.Size = int64(len(dconfig))
	dmanifest.Config.MediaType = docker.V2S2MediaTypeImageConfig

	dmanifestbytes, err := json.Marshal(&dmanifest)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Docker v2s2 manifest = %s", dmanifestbytes)

	var config []byte
	var manifest []byte
	switch manifestType {
	case v1.MediaTypeImageManifest:
		manifest = omanifestbytes
		config = i.oconfig
	case docker.V2S2MediaTypeManifest:
		manifest = dmanifestbytes
		config = i.dconfig
	default:
		panic("unreachable code: unsupported manifest type")
	}
	src = &containerImageSource{
		path:         path,
		ref:          i,
		store:        i.store,
		container:    i.container,
		compression:  i.compression,
		manifest:     manifest,
		manifestType: manifestType,
		config:       config,
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
	return i.manifest, i.manifestType, nil
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
	container, err := b.store.Container(b.ContainerID)
	if err != nil {
		return nil, err
	}
	if len(container.Names) > 0 {
		name, err = reference.ParseNamed(container.Names[0])
		if err != nil {
			name = nil
		}
	}
	oconfig, err := json.Marshal(&b.OCIv1)
	if err != nil {
		return nil, err
	}
	dconfig, err := json.Marshal(&b.Docker)
	if err != nil {
		return nil, err
	}
	ref := &containerImageRef{
		store:       b.store,
		container:   container,
		compression: compress,
		name:        name,
		oconfig:     oconfig,
		dconfig:     dconfig,
		createdBy:   b.CreatedBy(),
		annotations: b.Annotations(),
	}
	return ref, nil
}
