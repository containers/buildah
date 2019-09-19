package manifests

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/buildah/pkg/manifests"
	cp "github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	"github.com/containers/storage"
	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

var (
	_        List = &list{}
	sys           = &types.SystemContext{}
	amd64sys      = &types.SystemContext{ArchitectureChoice: "amd64"}
	arm64sys      = &types.SystemContext{ArchitectureChoice: "arm64"}
	ppc64sys      = &types.SystemContext{ArchitectureChoice: "ppc64le"}
)

const (
	listImageName = "foo"

	otherListImage          = "docker://k8s.gcr.io/pause:3.1"
	otherListDigest         = "sha256:f78411e19d84a252e53bff71a4407a5686c46983a2c2eeed83929b888179acea"
	otherListAmd64Digest    = "sha256:59eec8837a4d942cc19a52b8c09ea75121acc38114a2c68b98983ce9356b8610"
	otherListArm64Digest    = "sha256:f365626a556e58189fc21d099fc64603db0f440bff07f77c740989515c544a39"
	otherListPpc64Digest    = "sha256:bcf9771c0b505e68c65440474179592ffdfa98790eb54ffbf129969c5e429990"
	otherListInstanceDigest = "docker://k8s.gcr.io/pause@sha256:f365626a556e58189fc21d099fc64603db0f440bff07f77c740989515c544a39"
)

func TestSaveLoad(t *testing.T) {
	dir, err := ioutil.TempDir("", "manifests")
	assert.Nilf(t, err, "error creating temporary directory")
	defer os.RemoveAll(dir)

	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.Nilf(t, err, "error opening store")
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.Nilf(t, err, "error closing store")
		}
	}()

	list := Create()
	assert.NotNil(t, list, "Create() returned nil?")

	image, err := list.SaveToImage(store, "", []string{listImageName}, manifest.DockerV2ListMediaType)
	assert.Nilf(t, err, "SaveToImage(1)")
	imageReused, err := list.SaveToImage(store, image, nil, manifest.DockerV2ListMediaType)
	assert.Nilf(t, err, "SaveToImage(2)")

	_, list, err = LoadFromImage(store, image)
	assert.Nilf(t, err, "LoadFromImage(1)")
	assert.NotNilf(t, list, "LoadFromImage(1)")
	_, list, err = LoadFromImage(store, imageReused)
	assert.Nilf(t, err, "LoadFromImage(2)")
	assert.NotNilf(t, list, "LoadFromImage(2)")
	_, list, err = LoadFromImage(store, listImageName)
	assert.Nilf(t, err, "LoadFromImage(3)")
	assert.NotNilf(t, list, "LoadFromImage(3)")
}

func TestAddRemove(t *testing.T) {
	ctx := context.TODO()

	ref, err := alltransports.ParseImageName(otherListImage)
	assert.Nilf(t, err, "ParseImageName(%q)", otherListImage)
	src, err := ref.NewImageSource(ctx, sys)
	assert.Nilf(t, err, "NewImageSource(%q)", otherListImage)
	defer assert.Nilf(t, src.Close(), "ImageSource.Close()")
	m, _, err := src.GetManifest(ctx, nil)
	assert.Nilf(t, err, "ImageSource.GetManifest()")
	assert.Nilf(t, src.Close(), "ImageSource.GetManifest()")
	listDigest, err := manifest.Digest(m)
	assert.Nilf(t, err, "manifest.Digest()")
	assert.Equalf(t, listDigest.String(), otherListDigest, "digest for image %q changed?", otherListImage)

	l, err := manifests.FromBlob(m)
	assert.Nilf(t, err, "manifests.FromBlob()")
	assert.NotNilf(t, l, "manifests.FromBlob()")
	assert.Equalf(t, len(l.Instances()), 5, "image %q had an arch added?", otherListImage)

	list := Create()
	instanceDigest, err := list.Add(ctx, amd64sys, ref, false)
	assert.Nilf(t, err, "list.Add(all=false)")
	assert.Equal(t, instanceDigest.String(), otherListAmd64Digest)
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added")

	list = Create()
	instanceDigest, err = list.Add(ctx, arm64sys, ref, false)
	assert.Nilf(t, err, "list.Add(all=false)")
	assert.Equal(t, instanceDigest.String(), otherListArm64Digest)
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added")

	list = Create()
	instanceDigest, err = list.Add(ctx, ppc64sys, ref, false)
	assert.Nilf(t, err, "list.Add(all=false)")
	assert.Equal(t, instanceDigest.String(), otherListPpc64Digest)
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added")

	_, err = list.Add(ctx, sys, ref, true)
	assert.Nilf(t, err, "list.Add(all=true)")
	assert.Equalf(t, len(list.Instances()), 5, "too many instances added")

	list = Create()
	_, err = list.Add(ctx, sys, ref, true)
	assert.Nilf(t, err, "list.Add(all=true)")
	assert.Equalf(t, len(list.Instances()), 5, "too many instances added", otherListImage)

	for _, instance := range list.Instances() {
		assert.Nilf(t, list.Remove(instance), "error removing instance %q", instance)
	}
	assert.Equalf(t, len(list.Instances()), 0, "should have removed all instances")

	ref, err = alltransports.ParseImageName(otherListInstanceDigest)
	assert.Nilf(t, err, "ParseImageName(%q)", otherListInstanceDigest)

	list = Create()
	_, err = list.Add(ctx, sys, ref, false)
	assert.Nilf(t, err, "list.Add(all=false)")
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added", otherListInstanceDigest)

	list = Create()
	_, err = list.Add(ctx, sys, ref, true)
	assert.Nilf(t, err, "list.Add(all=true)")
	assert.Equalf(t, len(list.Instances()), 1, "too many instances added", otherListInstanceDigest)
}

func TestReference(t *testing.T) {
	ctx := context.TODO()

	dir, err := ioutil.TempDir("", "manifests")
	assert.Nilf(t, err, "error creating temporary directory")
	defer os.RemoveAll(dir)

	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.Nilf(t, err, "error opening store")
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.Nilf(t, err, "error closing store")
		}
	}()

	ref, err := alltransports.ParseImageName(otherListImage)
	assert.Nilf(t, err, "ParseImageName(%q)", otherListImage)

	list := Create()
	_, err = list.Add(ctx, ppc64sys, ref, false)
	assert.Nilf(t, err, "list.Add(all=false)")

	listRef, err := list.Reference(store, cp.CopyAllImages, nil)
	assert.NotNilf(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopyAllImages, nil)
	assert.NotNilf(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySystemImage, nil)
	assert.NotNilf(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.NotNilf(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.NotNilf(t, err, "list.Reference(never saved)")
	assert.Nilf(t, listRef, "list.Reference(never saved)")

	_, err = list.SaveToImage(store, "", []string{listImageName}, "")
	assert.Nilf(t, err, "SaveToImage")

	listRef, err = list.Reference(store, cp.CopyAllImages, nil)
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySystemImage, nil)
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, nil)
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	_, err = list.Add(ctx, sys, ref, true)
	assert.Nilf(t, err, "list.Add(all=true)")

	listRef, err = list.Reference(store, cp.CopyAllImages, nil)
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySystemImage, nil)
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, nil)
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest})
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")

	listRef, err = list.Reference(store, cp.CopySpecificImages, []digest.Digest{otherListAmd64Digest, otherListArm64Digest})
	assert.Nilf(t, err, "list.Reference(saved)")
	assert.NotNilf(t, listRef, "list.Reference(saved)")
}

func TestPush(t *testing.T) {
	ctx := context.TODO()

	dir, err := ioutil.TempDir("", "manifests")
	assert.Nilf(t, err, "error creating temporary directory")
	defer os.RemoveAll(dir)

	storeOptions := storage.StoreOptions{
		GraphRoot:       filepath.Join(dir, "root"),
		RunRoot:         filepath.Join(dir, "runroot"),
		GraphDriverName: "vfs",
	}
	store, err := storage.GetStore(storeOptions)
	assert.Nilf(t, err, "error opening store")
	defer func() {
		if _, err := store.Shutdown(true); err != nil {
			assert.Nilf(t, err, "error closing store")
		}
	}()

	dest, err := ioutil.TempDir("", "manifests")
	assert.Nilf(t, err, "error creating temporary directory")
	defer os.RemoveAll(dest)

	destRef, err := alltransports.ParseImageName(fmt.Sprintf("dir:%s", dest))
	assert.Nilf(t, err, "ParseImageName()")

	ref, err := alltransports.ParseImageName(otherListImage)
	assert.Nilf(t, err, "ParseImageName(%q)", otherListImage)

	list := Create()
	_, err = list.Add(ctx, sys, ref, true)
	assert.Nilf(t, err, "list.Add(all=true)")

	_, err = list.SaveToImage(store, "", []string{listImageName}, "")
	assert.Nilf(t, err, "SaveToImage")

	options := PushOptions{
		Store:              store,
		SystemContext:      sys,
		ImageListSelection: cp.CopyAllImages,
		Instances:          nil,
	}
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(all)")

	options.ImageListSelection = cp.CopySystemImage
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(local)")

	options.ImageListSelection = cp.CopySpecificImages
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(none specified)")

	options.Instances = []digest.Digest{otherListAmd64Digest}
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(one specified)")

	options.Instances = append(options.Instances, otherListArm64Digest)
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(two specified)")

	options.Instances = append(options.Instances, otherListPpc64Digest)
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(three specified)")

	options.Instances = append(options.Instances, otherListDigest)
	_, _, err = list.Push(ctx, destRef, options)
	assert.Nilf(t, err, "list.Push(four specified)")
}
