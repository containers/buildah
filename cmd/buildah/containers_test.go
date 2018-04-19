package main

import (
	"fmt"
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

func TestContainerTemplateOutputInexistenceField(t *testing.T) {
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
		t.Fatalf("expected error inexistence field")
	}
}

func TestContainerFormatStringOutput(t *testing.T) {
	params := containerOutputParams{
		ContainerID:   "e477836657bb",
		Builder:       " ",
		ImageID:       "f975c5035748",
		ImageName:     "test/image:latest",
		ContainerName: "test-container",
	}

	output := captureOutput(func() {
		containerOutputUsingFormatString(true, params)
	})
	expectedOutput := fmt.Sprintf("%-12.12s  %-8s %-12.12s %-32s %s\n", params.ContainerID, params.Builder, params.ImageID, params.ImageName, params.ContainerName)
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
