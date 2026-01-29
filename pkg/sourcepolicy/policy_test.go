package sourcepolicy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid empty policy",
			json: `{"rules": []}`,
		},
		{
			name: "valid policy with DENY rule",
			json: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/ubuntu:latest"
						}
					}
				]
			}`,
		},
		{
			name: "valid policy with CONVERT rule",
			json: `{
				"rules": [
					{
						"action": "CONVERT",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest"
						},
						"updates": {
							"identifier": "docker-image://docker.io/library/alpine:latest@sha256:abc123"
						}
					}
				]
			}`,
		},
		{
			name: "valid policy with ALLOW rule",
			json: `{
				"rules": [
					{
						"action": "ALLOW",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:3.18"
						}
					}
				]
			}`,
		},
		{
			name: "valid policy with WILDCARD match type",
			json: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/*:latest",
							"matchType": "WILDCARD"
						}
					}
				]
			}`,
		},
		{
			name:        "invalid JSON",
			json:        `{invalid}`,
			wantErr:     true,
			errContains: "parsing source policy JSON",
		},
		{
			name: "missing action",
			json: `{
				"rules": [
					{
						"selector": {
							"identifier": "docker-image://test"
						}
					}
				]
			}`,
			wantErr:     true,
			errContains: "action is required",
		},
		{
			name: "unknown action",
			json: `{
				"rules": [
					{
						"action": "UNKNOWN",
						"selector": {
							"identifier": "docker-image://test"
						}
					}
				]
			}`,
			wantErr:     true,
			errContains: "unknown action",
		},
		{
			name: "missing selector identifier",
			json: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {}
					}
				]
			}`,
			wantErr:     true,
			errContains: "selector.identifier is required",
		},
		{
			name: "CONVERT without updates",
			json: `{
				"rules": [
					{
						"action": "CONVERT",
						"selector": {
							"identifier": "docker-image://test"
						}
					}
				]
			}`,
			wantErr:     true,
			errContains: "updates.identifier is required for CONVERT",
		},
		{
			name: "CONVERT with empty updates identifier",
			json: `{
				"rules": [
					{
						"action": "CONVERT",
						"selector": {
							"identifier": "docker-image://test"
						},
						"updates": {
							"identifier": ""
						}
					}
				]
			}`,
			wantErr:     true,
			errContains: "updates.identifier is required for CONVERT",
		},
		{
			name: "REGEX match type not supported",
			json: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://.*",
							"matchType": "REGEX"
						}
					}
				]
			}`,
			wantErr:     true,
			errContains: "REGEX match type is not supported",
		},
		{
			name: "unknown match type",
			json: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://test",
							"matchType": "UNKNOWN"
						}
					}
				]
			}`,
			wantErr:     true,
			errContains: "unknown matchType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := Parse([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Parse() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if policy == nil {
				t.Error("Parse() returned nil policy without error")
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a valid policy file
	validPolicy := `{
		"rules": [
			{
				"action": "DENY",
				"selector": {
					"identifier": "docker-image://test"
				}
			}
		]
	}`
	validFile := filepath.Join(tmpDir, "valid.json")
	if err := os.WriteFile(validFile, []byte(validPolicy), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create an invalid policy file
	invalidPolicy := `{invalid json}`
	invalidFile := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidFile, []byte(invalidPolicy), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid file",
			path: validFile,
		},
		{
			name:        "non-existent file",
			path:        filepath.Join(tmpDir, "nonexistent.json"),
			wantErr:     true,
			errContains: "reading source policy file",
		},
		{
			name:        "invalid JSON file",
			path:        invalidFile,
			wantErr:     true,
			errContains: "parsing source policy JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := LoadFromFile(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadFromFile() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("LoadFromFile() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if policy == nil {
				t.Error("LoadFromFile() returned nil policy without error")
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name          string
		policyJSON    string
		sourceID      string
		wantMatched   bool
		wantAction    Action
		wantTargetRef string
		wantErr       bool
	}{
		{
			name:        "nil policy returns no match",
			policyJSON:  "",
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: false,
		},
		{
			name:        "empty policy returns no match",
			policyJSON:  `{"rules": []}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: false,
		},
		{
			name: "exact match DENY",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest",
							"matchType": "EXACT"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: true,
			wantAction:  ActionDeny,
		},
		{
			name: "exact match no match",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/ubuntu:latest",
							"matchType": "EXACT"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: false,
		},
		{
			name: "exact match CONVERT",
			policyJSON: `{
				"rules": [
					{
						"action": "CONVERT",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest",
							"matchType": "EXACT"
						},
						"updates": {
							"identifier": "docker-image://docker.io/library/alpine:latest@sha256:abc123"
						}
					}
				]
			}`,
			sourceID:      "docker-image://docker.io/library/alpine:latest",
			wantMatched:   true,
			wantAction:    ActionConvert,
			wantTargetRef: "docker-image://docker.io/library/alpine:latest@sha256:abc123",
		},
		{
			name: "wildcard match DENY - star matches any",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/*:latest",
							"matchType": "WILDCARD"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: true,
			wantAction:  ActionDeny,
		},
		{
			name: "wildcard match - question mark matches single char",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:3.1?",
							"matchType": "WILDCARD"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:3.18",
			wantMatched: true,
			wantAction:  ActionDeny,
		},
		{
			name: "wildcard no match",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/*:stable",
							"matchType": "WILDCARD"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: false,
		},
		{
			name: "first match wins - DENY before ALLOW",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest"
						}
					},
					{
						"action": "ALLOW",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: true,
			wantAction:  ActionDeny,
		},
		{
			name: "first match wins - ALLOW before DENY",
			policyJSON: `{
				"rules": [
					{
						"action": "ALLOW",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest"
						}
					},
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest"
						}
					}
				]
			}`,
			sourceID:    "docker-image://docker.io/library/alpine:latest",
			wantMatched: true,
			wantAction:  ActionAllow,
		},
		{
			name: "multiple rules - second matches",
			policyJSON: `{
				"rules": [
					{
						"action": "DENY",
						"selector": {
							"identifier": "docker-image://docker.io/library/ubuntu:latest"
						}
					},
					{
						"action": "CONVERT",
						"selector": {
							"identifier": "docker-image://docker.io/library/alpine:latest"
						},
						"updates": {
							"identifier": "docker-image://myregistry/alpine:pinned"
						}
					}
				]
			}`,
			sourceID:      "docker-image://docker.io/library/alpine:latest",
			wantMatched:   true,
			wantAction:    ActionConvert,
			wantTargetRef: "docker-image://myregistry/alpine:pinned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var policy *Policy
			var err error

			if tt.policyJSON != "" {
				policy, err = Parse([]byte(tt.policyJSON))
				if err != nil {
					t.Fatalf("Failed to parse test policy: %v", err)
				}
			}

			decision, matched, err := policy.Evaluate(tt.sourceID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Evaluate() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if matched != tt.wantMatched {
				t.Errorf("Evaluate() matched = %v, want %v", matched, tt.wantMatched)
			}

			if matched {
				if decision.Action != tt.wantAction {
					t.Errorf("Evaluate() action = %v, want %v", decision.Action, tt.wantAction)
				}
				if tt.wantTargetRef != "" && decision.TargetRef != tt.wantTargetRef {
					t.Errorf("Evaluate() targetRef = %v, want %v", decision.TargetRef, tt.wantTargetRef)
				}
			}
		})
	}
}

func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern string
		str     string
		want    bool
	}{
		// Basic exact matches
		{"abc", "abc", true},
		{"abc", "abcd", false},
		{"abc", "ab", false},

		// Star wildcard
		{"*", "", true},
		{"*", "anything", true},
		{"a*", "a", true},
		{"a*", "abc", true},
		{"a*", "b", false},
		{"*c", "c", true},
		{"*c", "abc", true},
		{"*c", "cd", false},
		{"a*c", "ac", true},
		{"a*c", "abc", true},
		{"a*c", "abbc", true},
		{"a*c", "ab", false},

		// Question mark wildcard
		{"?", "a", true},
		{"?", "", false},
		{"?", "ab", false},
		{"a?c", "abc", true},
		{"a?c", "adc", true},
		{"a?c", "ac", false},
		{"a?c", "abbc", false},

		// Combined wildcards
		{"a*?", "ab", true},
		{"a*?", "abc", true},
		{"a*?", "a", false},
		{"a?*", "ab", true},
		{"a?*", "abc", true},
		{"a?*", "a", false},

		// Multiple stars
		{"*a*", "a", true},
		{"*a*", "ba", true},
		{"*a*", "ab", true},
		{"*a*", "bab", true},
		{"*a*", "b", false},

		// Real-world patterns
		{"docker-image://docker.io/library/*:latest", "docker-image://docker.io/library/alpine:latest", true},
		{"docker-image://docker.io/library/*:latest", "docker-image://docker.io/library/ubuntu:latest", true},
		{"docker-image://docker.io/library/*:latest", "docker-image://docker.io/library/alpine:3.18", false},
		{"docker-image://*/*:*", "docker-image://docker.io/library/alpine:3.18", true},
		{"docker-image://*/library/alpine:*", "docker-image://docker.io/library/alpine:latest", true},
		{"docker-image://*/library/alpine:*", "docker-image://gcr.io/library/alpine:v1", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.str, func(t *testing.T) {
			got := matchWildcard(tt.pattern, tt.str)
			if got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.str, got, tt.want)
			}
		})
	}
}

func TestImageSourceIdentifier(t *testing.T) {
	tests := []struct {
		imageRef string
		want     string
	}{
		// Already in docker-image:// format
		{"docker-image://docker.io/library/alpine:latest", "docker-image://docker.io/library/alpine:latest"},

		// Simple image names (no registry)
		{"alpine", "docker-image://docker.io/library/alpine"},
		{"alpine:latest", "docker-image://docker.io/library/alpine:latest"},
		{"alpine:3.18", "docker-image://docker.io/library/alpine:3.18"},

		// User images (no registry, with username)
		{"myuser/myimage", "docker-image://docker.io/myuser/myimage"},
		{"myuser/myimage:latest", "docker-image://docker.io/myuser/myimage:latest"},

		// Full registry paths
		{"docker.io/library/alpine:latest", "docker-image://docker.io/library/alpine:latest"},
		{"gcr.io/project/image:tag", "docker-image://gcr.io/project/image:tag"},
		{"localhost:5000/myimage", "docker-image://localhost:5000/myimage"},
		{"myregistry.com:8080/project/image:v1", "docker-image://myregistry.com:8080/project/image:v1"},

		// Scratch (special case)
		{"scratch", "docker-image://scratch"},

		// With digest (using valid 64-character hex digests)
		{"alpine@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "docker-image://docker.io/library/alpine@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"docker.io/library/alpine@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "docker-image://docker.io/library/alpine@sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	}

	for _, tt := range tests {
		t.Run(tt.imageRef, func(t *testing.T) {
			got := ImageSourceIdentifier(tt.imageRef)
			if got != tt.want {
				t.Errorf("ImageSourceIdentifier(%q) = %q, want %q", tt.imageRef, got, tt.want)
			}
		})
	}
}

func TestExtractImageRef(t *testing.T) {
	tests := []struct {
		sourceID string
		want     string
	}{
		{"docker-image://docker.io/library/alpine:latest", "docker.io/library/alpine:latest"},
		{"docker-image://gcr.io/project/image:tag", "gcr.io/project/image:tag"},
		{"docker-image://alpine", "alpine"},

		// Non-docker-image sources (returned as-is)
		{"https://example.com/file.tar.gz", "https://example.com/file.tar.gz"},
		{"git://github.com/user/repo.git#main", "git://github.com/user/repo.git#main"},
		{"alpine:latest", "alpine:latest"},
	}

	for _, tt := range tests {
		t.Run(tt.sourceID, func(t *testing.T) {
			got := ExtractImageRef(tt.sourceID)
			if got != tt.want {
				t.Errorf("ExtractImageRef(%q) = %q, want %q", tt.sourceID, got, tt.want)
			}
		})
	}
}
