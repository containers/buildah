package main

import (
	"testing"
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
