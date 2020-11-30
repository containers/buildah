package supplemented

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/blobinfocache/none"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	digest "github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

var (
	_   types.ImageReference = &supplementedImageReference{}
	_   types.ImageSource    = &supplementedImageSource{}
	now                      = time.Now()
)

func makeLayer(t *testing.T) []byte {
	var b bytes.Buffer
	len := 512
	randomLen := 8
	tw := tar.NewWriter(&b)
	assert.Nilf(t, tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     "tmpfile",
		Size:     int64(len),
		Mode:     0644,
		Uname:    "root",
		Gname:    "root",
		ModTime:  time.Now(),
	}), "error writing in-memory layer")
	buf := make([]byte, len)
	n, err := rand.Read(buf[0:randomLen])
	assert.Nilf(t, err, "error reading a random byte")
	assert.Equalf(t, randomLen, n, "error reading random content: wrong length")
	for i := randomLen; i < len; i++ {
		buf[i] = (buf[i-1] + 1) & 0xff
	}
	n, err = tw.Write(buf)
	assert.Nilf(t, err, "error writing file content")
	assert.Equalf(t, n, len, "error writing file content: wrong length")
	assert.Nilf(t, tw.Close(), "error flushing file content")
	return b.Bytes()
}

func makeConfig(arch, os string, layer []byte) v1.Image {
	diffID := digest.Canonical.FromBytes(layer)
	return v1.Image{
		Created:      &now,
		Architecture: arch,
		OS:           os,
		Config: v1.ImageConfig{
			User:       "root",
			Entrypoint: []string{"/tmpfile"},
			WorkingDir: "/",
		},
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: []digest.Digest{diffID},
		},
		History: []v1.History{{
			Created:   &now,
			CreatedBy: "shenanigans",
		}},
	}
}

func makeManifest(layer, config []byte) v1.Manifest {
	return v1.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config: v1.Descriptor{
			MediaType: v1.MediaTypeImageConfig,
			Digest:    digest.Canonical.FromBytes(config),
			Size:      int64(len(config)),
		},
		Layers: []v1.Descriptor{{
			MediaType: v1.MediaTypeImageLayer,
			Digest:    digest.Canonical.FromBytes(layer),
			Size:      int64(len(layer)),
		}},
	}
}

func makeImage(t *testing.T, arch, os string) (ref types.ImageReference, dir string, layer, config, manifest []byte) {
	ctx := context.TODO()

	dir, err := ioutil.TempDir("", "supplemented")
	assert.Nilf(t, err, "error creating temporary directory")

	layerBytes := makeLayer(t)
	cb := makeConfig(arch, os, layer)
	configBytes, err := json.Marshal(&cb)
	assert.Nilf(t, err, "error encoding image configuration")
	m := makeManifest(layerBytes, configBytes)
	manifestBytes, err := json.Marshal(&m)
	assert.Nilf(t, err, "error encoding image manifest")

	ref, err = alltransports.ParseImageName(fmt.Sprintf("dir:%s", dir))
	assert.Nilf(t, err, "error parsing reference 'dir:%s'", dir)
	sys := &types.SystemContext{}
	dest, err := ref.NewImageDestination(ctx, sys)
	assert.Nilf(t, err, "error opening 'dir:%s' as an image destination", dir)
	bi := types.BlobInfo{
		MediaType: v1.MediaTypeImageLayer,
		Digest:    digest.Canonical.FromBytes(layerBytes),
		Size:      int64(len(layerBytes)),
	}
	_, err = dest.PutBlob(ctx, bytes.NewReader(layerBytes), bi, none.NoCache, false)
	assert.Nilf(t, err, "error storing layer blob to 'dir:%s'", dir)
	bi = types.BlobInfo{
		MediaType: v1.MediaTypeImageConfig,
		Digest:    digest.Canonical.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}
	_, err = dest.PutBlob(ctx, bytes.NewReader(configBytes), bi, none.NoCache, true)
	assert.Nilf(t, err, "error storing config blob to 'dir:%s'", dir)
	err = dest.PutManifest(ctx, manifestBytes, nil)
	assert.Nilf(t, err, "error storing manifest to 'dir:%s'", dir)
	err = dest.Commit(ctx, nil)
	assert.Nilf(t, err, "error committing image to 'dir:%s'", dir)

	return ref, dir, layerBytes, configBytes, manifestBytes
}

func TestSupplemented(t *testing.T) {
	ctx := context.TODO()
	arch2 := "foo"
	arch3 := "bar"

	sys := &types.SystemContext{
		SignaturePolicyPath: "../../tests/policy.json",
	}
	defaultPolicy, err := signature.DefaultPolicy(sys)
	assert.Nilf(t, err, "error obtaining default policy")
	policyContext, err := signature.NewPolicyContext(defaultPolicy)
	assert.Nilf(t, err, "error obtaining policy context")

	ref1, dir1, layer1, config1, manifest1 := makeImage(t, runtime.GOARCH, runtime.GOOS)
	defer os.RemoveAll(dir1)
	digest1, err := manifest.Digest(manifest1)
	assert.Nilf(t, err, "error digesting manifest")

	ref2, dir2, layer2, config2, manifest2 := makeImage(t, arch2, runtime.GOOS)
	defer os.RemoveAll(dir2)
	digest2, err := manifest.Digest(manifest2)
	assert.Nilf(t, err, "error digesting manifest")

	ref3, dir3, layer3, config3, manifest3 := makeImage(t, arch3, runtime.GOOS)
	defer os.RemoveAll(dir3)
	digest3, err := manifest.Digest(manifest3)
	assert.Nilf(t, err, "error digesting manifest")

	multidir, err := ioutil.TempDir("", "supplemented")
	assert.Nilf(t, err, "error creating temporary directory")
	defer os.RemoveAll(multidir)

	destDir, err := ioutil.TempDir("", "supplemented")
	assert.Nilf(t, err, "error creating temporary directory")
	defer os.RemoveAll(destDir)

	index := v1.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Manifests: []v1.Descriptor{
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    digest1,
				Size:      int64(len(manifest1)),
				Platform: &v1.Platform{
					Architecture: runtime.GOARCH,
					OS:           runtime.GOOS,
				},
			},
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    digest2,
				Size:      int64(len(manifest2)),
				Platform: &v1.Platform{
					Architecture: arch2,
					OS:           runtime.GOOS,
				},
			},
			{
				MediaType: v1.MediaTypeImageManifest,
				Digest:    digest3,
				Size:      int64(len(manifest3)),
				Platform: &v1.Platform{
					Architecture: arch3,
					OS:           runtime.GOOS,
				},
			},
		},
	}
	indexBytes, err := json.Marshal(&index)
	assert.Nilf(t, err, "error encoding image index")
	indexDigest, err := manifest.Digest(indexBytes)
	assert.Nilf(t, err, "error digesting image index")

	destRef, err := alltransports.ParseImageName(fmt.Sprintf("dir:%s", destDir))
	assert.Nilf(t, err, "error parsing reference 'dir:%s'", destDir)

	multiRef, err := alltransports.ParseImageName(fmt.Sprintf("dir:%s", multidir))
	assert.Nilf(t, err, "error parsing reference 'dir:%s'", multidir)
	destImg, err := multiRef.NewImageDestination(ctx, sys)
	assert.Nilf(t, err, "error opening 'dir:%s' as an image destination", multidir)
	err = destImg.PutManifest(ctx, indexBytes, nil)
	assert.Nilf(t, err, "error storing index to 'dir:%s'", multidir)
	err = destImg.Commit(ctx, nil)
	assert.Nilf(t, err, "error committing image to 'dir:%s'", multidir)

	t.Logf("list: digest=%q,value=%s", indexDigest, string(indexBytes))

	_, err = multiRef.NewImage(ctx, sys)
	assert.NotNilf(t, err, "unexpected success opening image 'dir:%s': shouldn't have been able to read config", multidir)

	src, err := Reference(multiRef, []types.ImageReference{ref1}, cp.CopyAllImages, nil).NewImageSource(ctx, sys)
	assert.NotNilf(t, err, "unexpected success opening image 'dir:%s': shouldn't have been able to read all manifests", multidir)
	assert.Nilf(t, src, "unexpected success opening image 'dir:%s': shouldn't have been able to read all manifests", multidir)
	src, err = Reference(multiRef, []types.ImageReference{ref1}, cp.CopySpecificImages, []digest.Digest{digest1}).NewImageSource(ctx, sys)
	assert.Nilf(t, err, "error opening image 'dir:%s' with specific instances", multidir)
	assert.Nilf(t, src.Close(), "error closing image 'dir:%s' with specific instances", multidir)

	img, err := Reference(multiRef, nil, cp.CopySystemImage, nil).NewImage(ctx, sys)
	assert.NotNilf(t, err, "unexpected success opening image 'dir:%s': shouldn't have been able to read config", multidir)
	assert.Nilf(t, img, "unexpected success opening image 'dir:%s': shouldn't have been able to read config", multidir)
	img, err = Reference(multiRef, []types.ImageReference{ref1}, cp.CopySystemImage, []digest.Digest{digest1}).NewImage(ctx, sys)
	assert.Nilf(t, err, "error opening image %q+%q", transports.ImageName(multiRef), transports.ImageName(ref1))
	assert.Nilf(t, img.Close(), "error closing image %q+%q", transports.ImageName(multiRef), transports.ImageName(ref1))

	type testCase struct {
		label           string
		supplements     []types.ImageReference
		expectToFind    [][]byte
		expectToNotFind [][]byte
		multiple        cp.ImageListSelection
		instances       []digest.Digest
	}

	for _, test := range []testCase{
		{
			label:           "no supplements, nil instances",
			supplements:     nil,
			expectToFind:    nil,
			expectToNotFind: [][]byte{layer1, config1, layer2, config2, layer3, config3},
			multiple:        cp.CopySpecificImages,
			instances:       nil,
		},
		{
			label:           "no supplements, 0 instances",
			supplements:     nil,
			expectToFind:    nil,
			expectToNotFind: [][]byte{layer1, config1, layer2, config2, layer3, config3},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{},
		},
		{
			label:           "just ref1 supplementing",
			supplements:     []types.ImageReference{ref1},
			expectToFind:    [][]byte{layer1, config1},
			expectToNotFind: [][]byte{layer2, config2, layer3, config3},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest1},
		},
		{
			label:           "just ref2 supplementing",
			supplements:     []types.ImageReference{ref2},
			expectToFind:    [][]byte{layer2, config2},
			expectToNotFind: [][]byte{layer1, config1, layer3, config3},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest2},
		},
		{
			label:           "just ref3 supplementing",
			supplements:     []types.ImageReference{ref3},
			expectToFind:    [][]byte{layer3, config3},
			expectToNotFind: [][]byte{layer1, config1, layer2, config2},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest3},
		},
		{
			label:           "refs 1 and 2 supplementing",
			supplements:     []types.ImageReference{ref1, ref2},
			expectToFind:    [][]byte{layer1, config1, layer2, config2},
			expectToNotFind: [][]byte{layer3, config3},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest1, digest2},
		},
		{
			label:           "refs 2 and 3 supplementing",
			supplements:     []types.ImageReference{ref2, ref3},
			expectToFind:    [][]byte{layer2, config2, layer3, config3},
			expectToNotFind: [][]byte{layer1, config1},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest2, digest3},
		},
		{
			label:           "refs 1 and 3 supplementing",
			supplements:     []types.ImageReference{ref1, ref3},
			expectToFind:    [][]byte{layer1, config1, layer3, config3},
			expectToNotFind: [][]byte{layer2, config2},
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest1, digest3},
		},
		{
			label:           "all refs supplementing, all instances",
			supplements:     []types.ImageReference{ref1, ref2, ref3},
			expectToFind:    [][]byte{layer1, config1, layer2, config2, layer3, config3},
			expectToNotFind: nil,
			multiple:        cp.CopySpecificImages,
			instances:       []digest.Digest{digest1, digest2, digest3},
		},
		{
			label:           "all refs supplementing, all images",
			supplements:     []types.ImageReference{ref1, ref2, ref3},
			expectToFind:    [][]byte{layer1, config1, layer2, config2, layer3, config3},
			expectToNotFind: nil,
			multiple:        cp.CopyAllImages,
		},
	} {
		supplemented := Reference(multiRef, test.supplements, test.multiple, test.instances)
		src, err := supplemented.NewImageSource(ctx, sys)
		assert.Nilf(t, err, "error opening image source 'dir:%s'[%s]", multidir, test.label)
		defer src.Close()
		for i, expect := range test.expectToFind {
			bi := types.BlobInfo{
				Digest: digest.Canonical.FromBytes(expect),
				Size:   int64(len(expect)),
			}
			rc, _, err := src.GetBlob(ctx, bi, none.NoCache)
			assert.Nilf(t, err, "error reading blob 'dir:%s'[%s][%d]", multidir, test.label, i)
			_, err = io.Copy(ioutil.Discard, rc)
			assert.Nilf(t, err, "error discarding blob 'dir:%s'[%s][%d]", multidir, test.label, i)
			rc.Close()
		}
		for i, expect := range test.expectToNotFind {
			bi := types.BlobInfo{
				Digest: digest.Canonical.FromBytes(expect),
				Size:   int64(len(expect)),
			}
			_, _, err := src.GetBlob(ctx, bi, none.NoCache)
			assert.NotNilf(t, err, "unexpected success reading blob 'dir:%s'[%s][%d]", multidir, test.label, i)
		}
		options := cp.Options{
			ImageListSelection: test.multiple,
			Instances:          test.instances,
		}
		_, err = cp.Image(ctx, policyContext, destRef, supplemented, &options)
		assert.Nilf(t, err, "error copying image 'dir:%s'[%s]", multidir, test.label)
	}
}
