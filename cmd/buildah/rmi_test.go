package main

import (
	"strings"
	"testing"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
)

func TestProperImageRefTrue(t *testing.T) {
	// Pull an image so we know we have it
	pullTestImage(t)

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
	pullTestImage(t)

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
	pullTestImage(t)

	imgRef, err := storageImageRef(&testSystemContext, store, "busybox")
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
	pullTestImage(t)

	imgRef, _ := storageImageRef(&testSystemContext, store, "")
	if imgRef != nil {
		t.Error("should not have found an Image Reference")
	}
}

func TestStorageImageIDTrue(t *testing.T) {
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
	pullTestImage(t)

	//Somehow I have to get the id of the image I just pulled
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	var id string
	if len(images) > 0 {
		id = strings.TrimSpace(images[0].ID)
	}
	if id == "" {
		t.Fatalf("Error getting image id")
	}

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
