package main

import (
	"strings"
	"testing"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
)

func TestProperImageRefTrue(t *testing.T) {
	// Pull an image so we know we have it
	err := pullTestImage("busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove")
	}
	// This should match a url path
	imgRef, err := properImageRef("docker://busybox:latest")
	if err != nil {
		t.Errorf("could not match image: %v", err)
	} else if imgRef == nil {
		t.Error("Returned nil Image Reference")
	}
}

func TestProperImageRefFalse(t *testing.T) {
	// Pull an image so we know we have it
	err := pullTestImage("busybox:latest")
	if err != nil {
		t.Fatal("could not pull image to remove")
	}
	// This should match a url path
	imgRef, _ := properImageRef("docker://:")
	if imgRef != nil {
		t.Error("should not have found an Image Reference")
	}
}

func TestStorageImageRefTrue(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	options := storage.DefaultStoreOptions
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it
	err = pullTestImage("busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	imgRef, err := storageImageRef(store, "busybox")
	if err != nil {
		t.Errorf("could not match image: %v", err)
	} else if imgRef == nil {
		t.Error("Returned nil Image Reference")
	}
}

func TestStorageImageRefFalse(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	options := storage.DefaultStoreOptions
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it
	err = pullTestImage("busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	imgRef, _ := storageImageRef(store, "")
	if imgRef != nil {
		t.Error("should not have found an Image Reference")
	}
}

func TestStorageImageIDTrue(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	options := storage.DefaultStoreOptions
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it
	err = pullTestImage("busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	//Somehow I have to get the id of the image I just pulled
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	id, err := captureOutputWithError(func() error {
		return outputImages(images, "", store, nil, "busybox:latest", false, false, false, true)
	})
	if err != nil {
		t.Fatalf("Error getting id of image: %v", err)
	}
	id = strings.TrimSpace(id)

	imgRef, err := storageImageID(store, id)
	if err != nil {
		t.Errorf("could not match image: %v", err)
	} else if imgRef == nil {
		t.Error("Returned nil Image Reference")
	}
}

func TestStorageImageIDFalse(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	options := storage.DefaultStoreOptions
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	if err != nil {
		t.Fatalf("could not get store: %v", err)
	}
	// Pull an image so we know we have it

	id := ""

	imgRef, _ := storageImageID(store, id)
	if imgRef != nil {
		t.Error("should not have returned Image Reference")
	}
}
