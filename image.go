package buildah

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/containers/image/docker/reference"
	"github.com/containers/image/image"
	"github.com/containers/image/manifest"
	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/ioutils"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/docker"
	"github.com/sirupsen/logrus"
)

const (
	// OCIv1ImageManifest is the MIME type of an OCIv1 image manifest,
	// suitable for specifying as a value of the PreferredManifestType
	// member of a CommitOptions structure.  It is also the default.
	OCIv1ImageManifest = v1.MediaTypeImageManifest
	// Dockerv2ImageManifest is the MIME type of a Docker v2s2 image
	// manifest, suitable for specifying as a value of the
	// PreferredManifestType member of a CommitOptions structure.
	Dockerv2ImageManifest = docker.V2S2MediaTypeManifest
)

type containerImageRef struct {
	store                 storage.Store
	compression           archive.Compression
	name                  reference.Named
	names                 []string
	layerID               string
	addHistory            bool
	oconfig               []byte
	dconfig               []byte
	created               time.Time
	createdBy             string
	annotations           map[string]string
	preferredManifestType string
	exporting             bool
}

type containerImageSource struct {
	path         string
	ref          *containerImageRef
	store        storage.Store
	layerID      string
	names        []string
	addHistory   bool
	compression  archive.Compression
	config       []byte
	configDigest digest.Digest
	manifest     []byte
	manifestType string
	exporting    bool
}

func (i *containerImageRef) NewImage(sc *types.SystemContext) (types.Image, error) {
	src, err := i.NewImageSource(sc)
	if err != nil {
		return nil, err
	}
	return image.FromSource(src)
}

func (i *containerImageRef) NewImageSource(sc *types.SystemContext) (src types.ImageSource, err error) {
	// Decide which type of manifest and configuration output we're going to provide.
	manifestType := i.preferredManifestType
	// If it's not a format we support, return an error.
	if manifestType != v1.MediaTypeImageManifest && manifestType != docker.V2S2MediaTypeManifest {
		return nil, errors.Errorf("no supported manifest types (attempted to use %q, only know %q and %q)",
			manifestType, v1.MediaTypeImageManifest, docker.V2S2MediaTypeManifest)
	}
	// Start building the list of layers using the read-write layer.
	layers := []string{}
	layerID := i.layerID
	layer, err := i.store.Layer(layerID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read layer %q", layerID)
	}
	// Walk the list of parent layers, prepending each as we go.
	for layer != nil {
		layers = append(append([]string{}, layerID), layers...)
		layerID = layer.Parent
		if layerID == "" {
			err = nil
			break
		}
		layer, err = i.store.Layer(layerID)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read layer %q", layerID)
		}
	}
	logrus.Debugf("layer list: %q", layers)

	// Make a temporary directory to hold blobs.
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

	// Build fresh copies of the configurations so that we don't mess with the values in the Builder
	// object itself.
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

	// Start building manifests.
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
	oimage.RootFS.DiffIDs = []digest.Digest{}
	dimage.RootFS = &docker.V2S2RootFS{}
	dimage.RootFS.Type = docker.TypeLayers
	dimage.RootFS.DiffIDs = []digest.Digest{}

	// Extract each layer and compute its digests, both compressed (if requested) and uncompressed.
	for _, layerID := range layers {
		omediaType := v1.MediaTypeImageLayer
		dmediaType := docker.V2S2MediaTypeUncompressedLayer
		// Figure out which media type we want to call this.  Assume no compression.
		if i.compression != archive.Uncompressed {
			switch i.compression {
			case archive.Gzip:
				omediaType = v1.MediaTypeImageLayerGzip
				dmediaType = docker.V2S2MediaTypeLayer
				logrus.Debugf("compressing layer %q with gzip", layerID)
			case archive.Bzip2:
				// Until the image specs define a media type for bzip2-compressed layers, even if we know
				// how to decompress them, we can't try to compress layers with bzip2.
				return nil, errors.New("media type for bzip2-compressed layers is not defined")
			default:
				logrus.Debugf("compressing layer %q with unknown compressor(?)", layerID)
			}
		}
		// If we're not re-exporting the data, just fake up layer and diff IDs for the manifest.
		if !i.exporting {
			fakeLayerDigest := digest.NewDigestFromHex(digest.Canonical.String(), layerID)
			// Add a note in the manifest about the layer.  The blobs should be identified by their
			// possibly-compressed blob digests, but just use the layer IDs here.
			olayerDescriptor := v1.Descriptor{
				MediaType: omediaType,
				Digest:    fakeLayerDigest,
				Size:      -1,
			}
			omanifest.Layers = append(omanifest.Layers, olayerDescriptor)
			dlayerDescriptor := docker.V2S2Descriptor{
				MediaType: dmediaType,
				Digest:    fakeLayerDigest,
				Size:      -1,
			}
			dmanifest.Layers = append(dmanifest.Layers, dlayerDescriptor)
			// Add a note about the diffID, which should be uncompressed digest of the blob, but
			// just use the layer ID here.
			oimage.RootFS.DiffIDs = append(oimage.RootFS.DiffIDs, fakeLayerDigest)
			dimage.RootFS.DiffIDs = append(dimage.RootFS.DiffIDs, fakeLayerDigest)
			continue
		}
		// Start reading the layer.
		rc, err := i.store.Diff("", layerID, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "error extracting layer %q", layerID)
		}
		defer rc.Close()
		// Set up to decompress the layer, in case it's coming out compressed.  Due to implementation
		// differences, the result may not match the digest the blob had when it was originally imported,
		// so we have to recompute all of this anyway if we want to be sure the digests we use will be
		// correct.
		uncompressed, err := archive.DecompressStream(rc)
		if err != nil {
			return nil, errors.Wrapf(err, "error decompressing layer %q", layerID)
		}
		defer uncompressed.Close()
		srcHasher := digest.Canonical.Digester()
		reader := io.TeeReader(uncompressed, srcHasher.Hash())
		// Set up to write the possibly-recompressed blob.
		layerFile, err := os.OpenFile(filepath.Join(path, "layer"), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, errors.Wrapf(err, "error opening file for layer %q", layerID)
		}
		destHasher := digest.Canonical.Digester()
		counter := ioutils.NewWriteCounter(layerFile)
		multiWriter := io.MultiWriter(counter, destHasher.Hash())
		// Compress the layer, if we're compressing it.
		writer, err := archive.CompressStream(multiWriter, i.compression)
		if err != nil {
			return nil, errors.Wrapf(err, "error compressing layer %q", layerID)
		}
		size, err := io.Copy(writer, reader)
		if err != nil {
			return nil, errors.Wrapf(err, "error storing layer %q to file", layerID)
		}
		writer.Close()
		layerFile.Close()
		if i.compression == archive.Uncompressed {
			if size != counter.Count {
				return nil, errors.Errorf("error storing layer %q to file: inconsistent layer size (copied %d, wrote %d)", layerID, size, counter.Count)
			}
		} else {
			size = counter.Count
		}
		logrus.Debugf("layer %q size is %d bytes", layerID, size)
		// Rename the layer so that we can more easily find it by digest later.
		err = os.Rename(filepath.Join(path, "layer"), filepath.Join(path, destHasher.Digest().String()))
		if err != nil {
			return nil, errors.Wrapf(err, "error storing layer %q to file", layerID)
		}
		// Add a note in the manifest about the layer.  The blobs are identified by their possibly-
		// compressed blob digests.
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
		// Add a note about the diffID, which is always an uncompressed value.
		oimage.RootFS.DiffIDs = append(oimage.RootFS.DiffIDs, srcHasher.Digest())
		dimage.RootFS.DiffIDs = append(dimage.RootFS.DiffIDs, srcHasher.Digest())
	}

	if i.addHistory {
		// Build history notes in the image configurations.
		onews := v1.History{
			Created:    &i.created,
			CreatedBy:  i.createdBy,
			Author:     oimage.Author,
			EmptyLayer: false,
		}
		oimage.History = append(oimage.History, onews)
		dnews := docker.V2S2History{
			Created:    i.created,
			CreatedBy:  i.createdBy,
			Author:     dimage.Author,
			EmptyLayer: false,
		}
		dimage.History = append(dimage.History, dnews)
	}

	// Encode the image configuration blob.
	oconfig, err := json.Marshal(&oimage)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("OCIv1 config = %s", oconfig)

	// Add the configuration blob to the manifest.
	omanifest.Config.Digest = digest.Canonical.FromBytes(oconfig)
	omanifest.Config.Size = int64(len(oconfig))
	omanifest.Config.MediaType = v1.MediaTypeImageConfig

	// Encode the manifest.
	omanifestbytes, err := json.Marshal(&omanifest)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("OCIv1 manifest = %s", omanifestbytes)

	// Encode the image configuration blob.
	dconfig, err := json.Marshal(&dimage)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Docker v2s2 config = %s", dconfig)

	// Add the configuration blob to the manifest.
	dmanifest.Config.Digest = digest.Canonical.FromBytes(dconfig)
	dmanifest.Config.Size = int64(len(dconfig))
	dmanifest.Config.MediaType = docker.V2S2MediaTypeImageConfig

	// Encode the manifest.
	dmanifestbytes, err := json.Marshal(&dmanifest)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Docker v2s2 manifest = %s", dmanifestbytes)

	// Decide which manifest and configuration blobs we'll actually output.
	var config []byte
	var manifest []byte
	switch manifestType {
	case v1.MediaTypeImageManifest:
		manifest = omanifestbytes
		config = oconfig
	case docker.V2S2MediaTypeManifest:
		manifest = dmanifestbytes
		config = dconfig
	default:
		panic("unreachable code: unsupported manifest type")
	}
	src = &containerImageSource{
		path:         path,
		ref:          i,
		store:        i.store,
		layerID:      i.layerID,
		names:        i.names,
		addHistory:   i.addHistory,
		compression:  i.compression,
		config:       config,
		configDigest: digest.Canonical.FromBytes(config),
		manifest:     manifest,
		manifestType: manifestType,
		exporting:    i.exporting,
	}
	return src, nil
}

func (i *containerImageRef) NewImageDestination(sc *types.SystemContext) (types.ImageDestination, error) {
	return nil, errors.Errorf("can't write to a container")
}

func (i *containerImageRef) DockerReference() reference.Named {
	return i.name
}

func (i *containerImageRef) StringWithinTransport() string {
	if len(i.names) > 0 {
		return i.names[0]
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

func (i *containerImageSource) GetSignatures(ctx context.Context) ([][]byte, error) {
	return nil, nil
}

func (i *containerImageSource) GetTargetManifest(digest digest.Digest) ([]byte, string, error) {
	return []byte{}, "", errors.Errorf("TODO")
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

func (b *Builder) makeImageRef(manifestType string, exporting, addHistory bool, compress archive.Compression, names []string, layerID string, historyTimestamp *time.Time) (types.ImageReference, error) {
	var name reference.Named
	if len(names) > 0 {
		if parsed, err := reference.ParseNamed(names[0]); err == nil {
			name = parsed
		}
	}
	if manifestType == "" {
		manifestType = OCIv1ImageManifest
	}
	oconfig, err := json.Marshal(&b.OCIv1)
	if err != nil {
		return nil, errors.Wrapf(err, "error encoding OCI-format image configuration")
	}
	dconfig, err := json.Marshal(&b.Docker)
	if err != nil {
		return nil, errors.Wrapf(err, "error encoding docker-format image configuration")
	}
	created := time.Now().UTC()
	if historyTimestamp != nil {
		created = historyTimestamp.UTC()
	}
	ref := &containerImageRef{
		store:                 b.store,
		compression:           compress,
		name:                  name,
		names:                 names,
		layerID:               layerID,
		addHistory:            addHistory,
		oconfig:               oconfig,
		dconfig:               dconfig,
		created:               created,
		createdBy:             b.CreatedBy(),
		annotations:           b.Annotations(),
		preferredManifestType: manifestType,
		exporting:             exporting,
	}
	return ref, nil
}

func (b *Builder) makeContainerImageRef(manifestType string, exporting bool, compress archive.Compression, historyTimestamp *time.Time) (types.ImageReference, error) {
	if manifestType == "" {
		manifestType = OCIv1ImageManifest
	}
	container, err := b.store.Container(b.ContainerID)
	if err != nil {
		return nil, errors.Wrapf(err, "error locating container %q", b.ContainerID)
	}
	return b.makeImageRef(manifestType, exporting, true, compress, container.Names, container.LayerID, historyTimestamp)
}

func (b *Builder) makeImageImageRef(compress archive.Compression, names []string, layerID string, historyTimestamp *time.Time) (types.ImageReference, error) {
	return b.makeImageRef(manifest.GuessMIMEType(b.Manifest), true, false, compress, names, layerID, historyTimestamp)
}
