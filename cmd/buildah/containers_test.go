package main

import (
	"strings"
	"testing"
)

func TestTemplateOutputValidFormat(t *testing.T) {
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

func TestTemplateOutputInvalidFormat(t *testing.T) {
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
