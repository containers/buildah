package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestContainerTemplateOutputValidFormat(t *testing.T) {
	params := containerOutputParams{
		ContainerID:   "e477836657bb",
		Builder:       " ",
		ImageID:       "f975c5035748",
		ImageName:     "test/image:latest",
		ContainerName: "test-container",
	}

	formatString := "Container ID: {{.ContainerID}}"
	expectedString := "Container ID: " + params.ContainerID

	output, err := captureOutputWithError(func() error {
		return containerOutputUsingTemplate(formatString, params)
	})
	if err != nil {
		t.Error(err)
	} else if strings.TrimSpace(output) != expectedString {
		t.Errorf("Errorf with template output:\nExpected: %s\nReceived: %s\n", expectedString, output)
	}
}

func TestContainerTemplateOutputInvalidFormat(t *testing.T) {
	params := containerOutputParams{
		ContainerID:   "e477836657bb",
		Builder:       " ",
		ImageID:       "f975c5035748",
		ImageName:     "test/image:latest",
		ContainerName: "test-container",
	}

	formatString := "ContainerID"

	err := containerOutputUsingTemplate(formatString, params)
	if err == nil || err.Error() != "error invalid format provided: ContainerID" {
		t.Fatalf("expected error invalid format")
	}
}

func TestContainerTemplateOutputNonexistentField(t *testing.T) {
	params := containerOutputParams{
		ContainerID:   "e477836657bb",
		Builder:       " ",
		ImageID:       "f975c5035748",
		ImageName:     "test/image:latest",
		ContainerName: "test-container",
	}

	formatString := "{{.ID}}"

	err := containerOutputUsingTemplate(formatString, params)
	if err == nil || !strings.Contains(err.Error(), "can't evaluate field ID") {
		t.Fatalf("expected error nonexistent field")
	}
}

func TestContainerFormatStringOutput(t *testing.T) {
	params := containerOutputParams{
		ContainerID:   "e477836657bb",
		Builder:       " ",
		ImageID:       "f975c5035748",
		ImageName:     "test/with/this/very/long/image:latest",
		ContainerName: "test-container",
	}
	const trimmedImageName = "test/with/this/very/long/imag..."

	output := captureOutput(func() {
		containerOutputUsingFormatString(true, params)
	})
	expectedOutput := fmt.Sprintf("%-12.12s  %-8s %-12.12s %-32s %s\n", params.ContainerID, params.Builder, params.ImageID, trimmedImageName, params.ContainerName)
	if output != expectedOutput {
		t.Errorf("Error outputting using format string:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}

	output = captureOutput(func() {
		containerOutputUsingFormatString(false, params)
	})
	expectedOutput = fmt.Sprintf("%-64s %-8s %-64s %-32s %s\n", params.ContainerID, params.Builder, params.ImageID, params.ImageName, params.ContainerName)
	if output != expectedOutput {
		t.Errorf("Error outputting using format string:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}
}

func TestContainerHeaderOutput(t *testing.T) {
	output := captureOutput(func() {
		containerOutputHeader(true)
	})
	expectedOutput := fmt.Sprintf("%-12s  %-8s %-12s %-32s %s\n", "CONTAINER ID", "BUILDER", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
	if output != expectedOutput {
		t.Errorf("Error outputting using format string:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
	}

	output = captureOutput(func() {
		containerOutputHeader(false)
	})
	expectedOutput = fmt.Sprintf("%-64s %-8s %-64s %-32s %s\n", "CONTAINER ID", "BUILDER", "IMAGE ID", "IMAGE NAME", "CONTAINER NAME")
	if output != expectedOutput {
		t.Errorf("Error outputting using format string:\n\texpected: %s\n\treceived: %s\n", expectedOutput, output)
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
