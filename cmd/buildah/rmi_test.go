package main

import (
	"strings"
	"testing"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
)

func TestProperImageRefTrue(t *testing.T) {
	// Pull an image so we know we have it
	_, err := pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove")
	}
	// This should match a url path
	imgRef, err := properImageRef(getContext(), "docker://busybox:latest")
	if err != nil {
		t.Errorf("could not match image: %v", err)
	} else if imgRef == nil {
		t.Error("Returned nil Image Reference")
	}
}

func TestProperImageRefFalse(t *testing.T) {
	// Pull an image so we know we have it
	_, err := pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatal("could not pull image to remove")
	}
	// This should match a url path
	imgRef, _ := properImageRef(getContext(), "docker://:")
	if imgRef != nil {
		t.Error("should not have found an Image Reference")
	}
}

func TestStorageImageRefTrue(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	imgRef, err := storageImageRef(getContext(), &testSystemContext, store, "busybox")
	if err != nil {
		t.Errorf("could not match image: %v", err)
	} else if imgRef == nil {
		t.Error("Returned nil Image Reference")
	}
}

func TestStorageImageRefFalse(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	imgRef, _ := storageImageRef(getContext(), &testSystemContext, store, "")
	if imgRef != nil {
		t.Error("should not have found an Image Reference")
	}
}

func TestStorageImageIDTrue(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	opts := imageOptions{
		quiet: true,
	}
	store, err := storage.GetStore(storeOptions)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	//Somehow I have to get the id of the image I just pulled
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	id, err := captureOutputWithError(func() error {
		return outputImages(getContext(), images, store, nil, "busybox:latest", opts)
	})
	if err != nil {
		t.Fatalf("Error getting id of image: %v", err)
	}
	id = strings.TrimSpace(id)

	imgRef, err := storageImageID(getContext(), store, id)
	if err != nil {
		t.Errorf("could not match image: %v", err)
	} else if imgRef == nil {
		t.Error("Returned nil Image Reference")
	}
}

func TestStorageImageIDFalse(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it

	id := ""

	imgRef, _ := storageImageID(getContext(), store, id)
	if imgRef != nil {
		t.Error("should not have returned Image Reference")
	}
}
