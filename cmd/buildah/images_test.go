package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	is "github.com/containers/image/storage"
	"github.com/containers/storage"
)

func TestTemplateOutputBlankTemplate(t *testing.T) {
	params := imageOutputParams{
		ID:        "0123456789abcdef",
		Name:      "test/image:latest",
		Digest:    "sha256:012345789abcdef012345789abcdef012345789abcdef012345789abcdef",
		CreatedAt: "Jan 01 2016 10:45",
		Size:      "97 KB",
	}

	err := outputUsingTemplate("", params)
	//Output: Words
	if err != nil {
		t.Error(err)
	}
}

func TestTemplateOutputValidTemplate(t *testing.T) {
	params := imageOutputParams{
		ID:        "0123456789abcdef",
		Name:      "test/image:latest",
		Digest:    "sha256:012345789abcdef012345789abcdef012345789abcdef012345789abcdef",
		CreatedAt: "Jan 01 2016 10:45",
		Size:      "97 KB",
	}

	templateString := "{{.ID}}"

	output, err := captureOutputWithError(func() error {
		return outputUsingTemplate(templateString, params)
	})
	if err != nil {
		t.Error(err)
	} else if strings.TrimSpace(output) != strings.TrimSpace(params.ID) {
		t.Errorf("Error with template output:\nExpected: %s\nReceived: %s\n", params.ID, output)
	}
}

func TestFormatStringOutput(t *testing.T) {
	params := imageOutputParams{
		ID:        "012345789abcdef",
		Name:      "test/image:latest",
		Digest:    "sha256:012345789abcdef012345789abcdef012345789abcdef012345789abcdef",
		CreatedAt: "Jan 01 2016 10:45",
		Size:      "97 KB",
	}

	output := captureOutput(func() {
		outputUsingFormatString(true, true, params)
	})
	expectedOutput := fmt.Sprintf("%-20.12s %-56s %-64s %-22s %s\n", params.ID, params.Name, params.Digest, params.CreatedAt, params.Size)
	if output != expectedOutput {
		t.Errorf("Error outputting using format string:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}
}

func TestSizeFormatting(t *testing.T) {
	size := formattedSize(0)
	if size != "0 B" {
		t.Errorf("Error formatting size: expected '%s' got '%s'", "0 B", size)
	}

	size = formattedSize(1024)
	if size != "1 KB" {
		t.Errorf("Error formatting size: expected '%s' got '%s'", "1 KB", size)
	}

	size = formattedSize(1024 * 1024 * 1024 * 1024 * 1024)
	if size != "1024 TB" {
		t.Errorf("Error formatting size: expected '%s' got '%s'", "1024 TB", size)
	}
}

func TestOutputHeader(t *testing.T) {
	output := captureOutput(func() {
		outputHeader(true, false)
	})
	expectedOutput := fmt.Sprintf("%-20s %-56s %-22s %s\n", "IMAGE ID", "IMAGE NAME", "CREATED AT", "SIZE")
	if output != expectedOutput {
		t.Errorf("Error outputting header:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}

	output = captureOutput(func() {
		outputHeader(true, true)
	})
	expectedOutput = fmt.Sprintf("%-20s %-56s %-71s %-22s %s\n", "IMAGE ID", "IMAGE NAME", "DIGEST", "CREATED AT", "SIZE")
	if output != expectedOutput {
		t.Errorf("Error outputting header:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}

	output = captureOutput(func() {
		outputHeader(false, false)
	})
	expectedOutput = fmt.Sprintf("%-64s %-56s %-22s %s\n", "IMAGE ID", "IMAGE NAME", "CREATED AT", "SIZE")
	if output != expectedOutput {
		t.Errorf("Error outputting header:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}
}

func TestMatchWithTag(t *testing.T) {
	isMatch := matchesReference("docker.io/kubernetes/pause:latest", "pause:latest")
	if !isMatch {
		t.Error("expected match, got not match")
	}

	isMatch = matchesReference("docker.io/kubernetes/pause:latest", "kubernetes/pause:latest")
	if !isMatch {
		t.Error("expected match, got no match")
	}
}

func TestNoMatchesReferenceWithTag(t *testing.T) {
	isMatch := matchesReference("docker.io/kubernetes/pause:latest", "redis:latest")
	if isMatch {
		t.Error("expected no match, got match")
	}

	isMatch = matchesReference("docker.io/kubernetes/pause:latest", "kubernetes/redis:latest")
	if isMatch {
		t.Error("expected no match, got match")
	}
}

func TestMatchesReferenceWithoutTag(t *testing.T) {
	isMatch := matchesReference("docker.io/kubernetes/pause:latest", "pause")
	if !isMatch {
		t.Error("expected match, got not match")
	}

	isMatch = matchesReference("docker.io/kubernetes/pause:latest", "kubernetes/pause")
	if !isMatch {
		t.Error("expected match, got no match")
	}
}

func TestNoMatchesReferenceWithoutTag(t *testing.T) {
	isMatch := matchesReference("docker.io/kubernetes/pause:latest", "redis")
	if isMatch {
		t.Error("expected no match, got match")
	}

	isMatch = matchesReference("docker.io/kubernetes/pause:latest", "kubernetes/redis")
	if isMatch {
		t.Error("expected no match, got match")
	}
}

func TestOutputImagesQuietTruncated(t *testing.T) {
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

	// Pull an image so that we know we have at least one
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	// Tests quiet and truncated output
	output, err := captureOutputWithError(func() error {
		return outputImages(images[:1], "", store, nil, "", false, true, false, true)
	})
	expectedOutput := fmt.Sprintf("%-64s\n", images[0].ID)
	if err != nil {
		t.Error("quiet/truncated output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("quiet/truncated output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
	}
}

func TestOutputImagesQuietNotTruncated(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests quiet and non-truncated output
	output, err := captureOutputWithError(func() error {
		return outputImages(images[:1], "", store, nil, "", false, false, false, true)
	})
	expectedOutput := fmt.Sprintf("%-64s\n", images[0].ID)
	if err != nil {
		t.Error("quiet/non-truncated output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("quiet/non-truncated output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
	}
}

func TestOutputImagesFormatString(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests output with format template
	output, err := captureOutputWithError(func() error {
		return outputImages(images[:1], "{{.ID}}", store, nil, "", true, true, false, false)
	})
	expectedOutput := images[0].ID
	if err != nil {
		t.Error("format string output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("format string output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
	}
}

func TestOutputImagesFormatTemplate(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests quiet and non-truncated output
	output, err := captureOutputWithError(func() error {
		return outputImages(images[:1], "", store, nil, "", false, false, false, true)
	})
	expectedOutput := fmt.Sprintf("%-64s\n", images[0].ID)
	if err != nil {
		t.Error("format template output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("format template output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
	}
}

func TestOutputImagesArgNoMatch(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull an image so that we know we have at least one
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests output with an arg name that does not match.  Args ending in ":" cannot match
	// because all images in the repository must have a tag, and here the tag is an
	// empty string
	output, err := captureOutputWithError(func() error {
		return outputImages(images[:1], "", store, nil, "foo:", false, true, false, false)
	})
	expectedOutput := fmt.Sprintf("")
	if err != nil {
		t.Error("arg no match output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Error("arg no match output should be empty")
	}
}

func TestOutputMultipleImages(t *testing.T) {
	// Make sure the tests are running as root
	failTestIfNotRoot(t)

	store, err := storage.GetStore(storeOptions)
	if err != nil {
		t.Fatal(err)
	} else if store != nil {
		is.Transport.SetStore(store)
	}

	// Pull two images so that we know we have at least two
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	_, err = pullTestImage(t, "alpine:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// Tests quiet and truncated output
	output, err := captureOutputWithError(func() error {
		return outputImages(images[:2], "", store, nil, "", false, true, false, true)
	})
	expectedOutput := fmt.Sprintf("%-64s\n%-64s\n", images[0].ID, images[1].ID)
	if err != nil {
		t.Error("multi-image output produces error")
	} else if strings.TrimSpace(output) != strings.TrimSpace(expectedOutput) {
		t.Errorf("multi-image output does not match expected value\nExpected: %s\nReceived: %s\n", expectedOutput, output)
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	label := "dangling=true,label=a=b,before=busybox:latest,since=busybox:latest,reference=abcdef"
	params, err := parseFilter(store, images, label)
	if err != nil {
		t.Fatalf("error parsing filter: %v", err)
	}

	ref, err := is.Transport.ParseStoreReference(store, "busybox:latest")
	if err != nil {
		t.Fatalf("error parsing store reference: %v", err)
	}
	img, err := ref.NewImage(nil)
	if err != nil {
		t.Fatalf("error reading image from store: %v", err)
	}
	defer img.Close()
	inspect, err := img.Inspect()
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	label := "dangling=NO,label=a=b,before=busybox:latest,since=busybox:latest,reference=abcdef"
	_, err = parseFilter(store, images, label)
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	label := "dangling=false,label=a=b,before=:,since=busybox:latest,reference=abcdef"
	_, err = parseFilter(store, images, label)
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	label := "dangling=false,label=a=b,before=busybox:latest,since=:,reference=abcdef"
	_, err = parseFilter(store, images, label)
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	label := "foo=bar"
	_, err = parseFilter(store, images, label)
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}

	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.beforeDate = time.Now()
	params.beforeImage = "foo:bar"
	if !matchesBeforeImage(images[0], ":", params) {
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.beforeDate = time.Time{}
	params.beforeImage = "foo:bar"
	// Should return false because the image has been seen
	if matchesBeforeImage(images[0], ":", params) {
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.sinceDate = time.Time{}
	params.sinceImage = "foo:bar"
	if !matchesSinceImage(images[0], ":", params) {
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
	_, err = pullTestImage(t, "busybox:latest")
	if err != nil {
		t.Fatalf("could not pull image to remove: %v", err)
	}
	images, err := store.Images()
	if err != nil {
		t.Fatalf("Error reading images: %v", err)
	}

	// by default, params.seenImage is false
	params := new(filterParams)
	params.sinceDate = time.Now()
	params.sinceImage = "foo:bar"
	// Should return false because the image has been seen
	if matchesSinceImage(images[0], ":", params) {
		t.Error("should not have matched sinceImage")
	}

	if matchesSinceImage(images[0], "foo:bar", params) {
		t.Error("image should have been filtered out")
	}
}

func captureOutputWithError(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
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
	io.Copy(&buf, r)
	return buf.String()
}
