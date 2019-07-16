package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/containers/buildah/util"
	is "github.com/containers/image/storage"
	"github.com/containers/storage"
)

func TestSizeFormatting(t *testing.T) {
	size := formattedSize(0)
	if size != "0 B" {
		t.Errorf("Error formatting size: expected '%s' got '%s'", "0 B", size)
	}

	size = formattedSize(1000)
	if size != "1 KB" {
		t.Errorf("Error formatting size: expected '%s' got '%s'", "1 KB", size)
	}

	size = formattedSize(1000 * 1000 * 1000 * 1000)
	if size != "1 TB" {
		t.Errorf("Error formatting size: expected '%s' got '%s'", "1 TB", size)
	}
}

func TestMatchWithTag(t *testing.T) {
	isMatch := matchesReference("gcr.io/pause:latest", "pause:latest")
	if !isMatch {
		t.Error("expected match, got not match")
	}

	isMatch = matchesReference("gcr.io/pause:latest", "kubernetes/pause:latest")
	if isMatch {
		t.Error("expected not match, got match")
	}
}

func TestNoMatchesReferenceWithTag(t *testing.T) {
	isMatch := matchesReference("gcr.io/pause:latest", "redis:latest")
	if isMatch {
		t.Error("expected no match, got match")
	}

	isMatch = matchesReference("gcr.io/pause:latest", "kubernetes/redis:latest")
	if isMatch {
		t.Error("expected no match, got match")
	}
}

func TestMatchesReferenceWithoutTag(t *testing.T) {
	isMatch := matchesReference("gcr.io/pause:latest", "pause")
	if !isMatch {
		t.Error("expected match, got not match")
	}

	isMatch = matchesReference("gcr.io/pause:latest", "kubernetes/pause")
	if isMatch {
		t.Error("expected not match, got match")
	}
}

func TestNoMatchesReferenceWithoutTag(t *testing.T) {
	isMatch := matchesReference("gcr.io/pause:latest", "redis")
	if isMatch {
		t.Error("expected no match, got match")
	}

	isMatch = matchesReference("gcr.io/pause:latest", "kubernetes/redis")
	if isMatch {
		t.Error("expected no match, got match")
	}
}

func TestOutputImagesQuietNotTruncated(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	opts := imageOptions{
		quiet: true,
	}
	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	pullTestImage(t)

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests quiet and non-truncated output
	output, err := captureOutputWithError(func() error {
		return outputImages(getContext(), &testSystemContext, store, images[:1], nil, "", opts)
	})
	expectedOutput := fmt.Sprintf("sha256:%s\n", images[0].ID)
	if err != nil {
		t.Error("quiet/non-truncated output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("quiet/non-truncated output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
	}
}

func TestOutputImagesFormatString(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	opts := imageOptions{
		format:   "{{.ID}}",
		truncate: true,
	}
	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	pullTestImage(t)

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests output with format template
	output, err := captureOutputWithError(func() error {
		return outputImages(getContext(), &testSystemContext, store, images[:1], nil, "", opts)
	})
	expectedOutput := images[0].ID
	if err != nil {
		t.Error("format string output produces error")
	} else if !strings.Contains(expectedOutput, strings.TrimSpace(output)) {
		t.Errorf("format string output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
	}
}

func TestOutputImagesArgNoMatch(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	opts := imageOptions{
		truncate: true,
	}
	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	pullTestImage(t)

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests output with an arg name that does not match.  Args ending in ":" cannot match
	// because all images in the repository must have a tag, and here the tag is an
	// empty string
	_, err = captureOutputWithError(func() error {
		return outputImages(getContext(), &testSystemContext, store, images[:1], nil, "foo:", opts)
	})
	if err == nil || err.Error() != "No such image foo:" {
		t.Fatalf("expected error arg no match")
	}
}

func TestParseFilterAllParams(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	// Pull an image so we know we have it
	pullTestImage(t)

	label := "dangling=true,label=a=b,before=busybox:latest,since=busybox:latest,reference=abcdef"
	params, err := parseFilter(getContext(), store, images, label)
	if err != nil {
		t.Fatalf("error parsing filter: %v", err)
	}

	ref, _, err := util.FindImage(store, "", &testSystemContext, "busybox:latest")
	if err != nil {
		t.Fatalf("error finding local copy of image: %v", err)
	}
	img, err := ref.NewImage(getContext(), nil)
	if err != nil {
		t.Fatalf("error reading image from store: %v", err)
	}
	defer img.Close()
	inspect, err := img.Inspect(getContext())
	if err != nil {
		t.Fatalf("error inspecting image in store: %v", err)
	}

	expectedParams := &filterParams{
		dangling:         "true",
		label:            "a=b",
		beforeImage:      "busybox:latest",
		beforeDate:       *inspect.Created,
		sinceImage:       "busybox:latest",
		sinceDate:        *inspect.Created,
		referencePattern: "abcdef",
	}
	if *params != *expectedParams {
		t.Errorf("filter did not return expected result\n\tExpected: %v\n\tReceived: %v", expectedParams, params)
	}
}

func TestParseFilterInvalidDangling(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	// Pull an image so we know we have it
	pullTestImage(t)

	label := "dangling=NO,label=a=b,before=busybox:latest,since=busybox:latest,reference=abcdef"
	_, err = parseFilter(getContext(), store, images, label)
	if err == nil || err.Error() != "invalid filter: 'dangling=[NO]'" {
		t.Fatalf("expected error parsing filter")
	}
}

func TestParseFilterInvalidBefore(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	// Pull an image so we know we have it
	pullTestImage(t)

	label := "dangling=false,label=a=b,before=:,since=busybox:latest,reference=abcdef"
	_, err = parseFilter(getContext(), store, images, label)
	if err == nil || !strings.Contains(err.Error(), "no such id") {
		t.Fatalf("expected error parsing filter")
	}
}

func TestParseFilterInvalidSince(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	// Pull an image so we know we have it
	pullTestImage(t)

	label := "dangling=false,label=a=b,before=busybox:latest,since=:,reference=abcdef"
	_, err = parseFilter(getContext(), store, images, label)
	if err == nil || !strings.Contains(err.Error(), "no such id") {
		t.Fatalf("expected error parsing filter")
	}
}

func TestParseFilterInvalidFilter(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}
	// Pull an image so we know we have it
	pullTestImage(t)

	label := "foo=bar"
	_, err = parseFilter(getContext(), store, images, label)
	if err == nil || err.Error() != "invalid filter: 'foo'" {
		t.Fatalf("expected error parsing filter")
	}
}

func TestMatchesDangingTrue(t *testing.T) {
	if !matchesDangling("<none>", "true") {
		t.Error("matchesDangling() should return true with dangling=true and name=<none>")
	}

	if !matchesDangling("hello", "false") {
		t.Error("matchesDangling() should return true with dangling=false and name='hello'")
	}
}

func TestMatchesDangingFalse(t *testing.T) {
	if matchesDangling("hello", "true") {
		t.Error("matchesDangling() should return false with dangling=true and name=hello")
	}

	if matchesDangling("<none>", "false") {
		t.Error("matchesDangling() should return false with dangling=false and name=<none>")
	}
}

func TestMatchesLabelTrue(t *testing.T) {
	//TODO: How do I implement this?
}

func TestMatchesLabelFalse(t *testing.T) {
	// TODO: How do I implement this?
}

func TestMatchesBeforeImageTrue(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	pullTestImage(t)

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.beforeDate = time.Now()
	params.beforeImage = "foo:bar"
	if !matchesBeforeImage(images[0], params) {
		t.Error("should have matched beforeImage")
	}
}

func TestMatchesBeforeImageFalse(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	// Pull an image so that we know we have at least one
	pullTestImage(t)
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.beforeDate = time.Time{}
	params.beforeImage = "foo:bar"
	// Should return false because the image has been seen
	if matchesBeforeImage(images[0], params) {
		t.Error("should not have matched beforeImage")
	}
}

func TestMatchesSinceeImageTrue(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	// Pull an image so that we know we have at least one
	pullTestImage(t)
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.sinceDate = time.Time{}
	params.sinceImage = "foo:bar"
	if !matchesSinceImage(images[0], params) {
		t.Error("should have matched SinceImage")
	}
}

func TestMatchesSinceImageFalse(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}
	// Pull an image so that we know we have at least one
	pullTestImage(t)
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.sinceDate = time.Now()
	params.sinceImage = "foo:bar"
	// Should return false because the image has been seen
	if matchesSinceImage(images[0], params) {
		t.Error("should not have matched sinceImage")
	}
}

func captureOutputWithError(f func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	if err := f(); err != nil {
		return "", err
	}

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint
	return buf.String(), err
}

// Captures output so that it can be compared to expected values
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint
	return buf.String()
}
