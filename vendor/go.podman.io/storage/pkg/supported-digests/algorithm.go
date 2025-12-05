package supporteddigests

// Package supporteddigests provides digest algorithm management for container tools.
//
// WARNING: This package is currently Work In Progress (WIP) and is ONLY intended
// for use within Podman, Buildah, and Skopeo. It should NOT be used by external
// applications or libraries, even if shipped in a stable release. The API may
// change without notice and is not considered stable for external consumption.
// Proceed with caution if you must use this package outside of the intended scope.

// FIXME: Use go-digest directly and address all review comments in
// https://github.com/containers/container-libs/pull/374. This is *one* of the blockers
// for removing the WIP warning.

import (
	"fmt"
	"strings"
	"sync"

	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
)

var (
	digestAlgorithm = digest.Canonical // Default to SHA256
	algorithmMutex  sync.RWMutex       // Protects digestAlgorithm from concurrent access
)

// TmpDigestForNewObjects returns the current digest algorithm that will be used
// for computing digests of new objects (e.g., image layers, manifests, blobs).
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// This function returns the globally configured digest algorithm for new object
// creation. It is thread-safe and can be called concurrently from multiple
// goroutines using RWMutex. The default value is SHA256 (digest.Canonical) on
// first call.
//
// This is a read-only operation that does not modify global state. The returned
// value reflects the current global configuration set by TmpSetDigestForNewObjects()
// or the default if never set. Multiple concurrent calls will return the same
// algorithm value. The algorithm is used for computing content hashes during
// image operations such as layer extraction, manifest generation, and blob storage.
func TmpDigestForNewObjects() digest.Algorithm {
	algorithmMutex.RLock()
	defer algorithmMutex.RUnlock()
	return digestAlgorithm
}

// TmpSetDigestForNewObjects sets the digest algorithm that will be used for
// computing digests of new objects (e.g., image layers, manifests, blobs).
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// This function configures the globally shared digest algorithm for new object
// creation. It is thread-safe and can be called concurrently from multiple
// goroutines using RWMutex. Changes affect all subsequent calls to
// TmpDigestForNewObjects().
//
// The function validates the algorithm and returns an error for unsupported values.
// Supported algorithms are SHA256, SHA512, or empty string (which defaults to SHA256).
// This is typically used to configure the digest algorithm for the process where
// an optional --digest flag is provided. For example: "podman|buildah build --digest sha512"
// to configure the digest algorithm for the build process.
//
// The setting persists for the lifetime of the process. This is a write operation
// that modifies global state atomically. Invalid algorithms are rejected without
// changing the current setting. Empty string is treated as a request to reset to
// the default (SHA256). Existing digest values are not affected by algorithm changes.
func TmpSetDigestForNewObjects(algorithm digest.Algorithm) error {
	algorithmMutex.Lock()
	defer algorithmMutex.Unlock()

	// Validate the digest type
	switch algorithm {
	case digest.SHA256, digest.SHA512:
		logrus.Debugf("SetDigestAlgorithm: Setting digest algorithm to %s", algorithm.String())
		digestAlgorithm = algorithm
		return nil
	case "":
		logrus.Debugf("SetDigestAlgorithm: Setting digest algorithm to default %s", digest.Canonical.String())
		digestAlgorithm = digest.Canonical // Default to sha256
		return nil
	default:
		return fmt.Errorf("unsupported digest algorithm: %q", algorithm)
	}
}

// IsSupportedDigestAlgorithm checks if the given algorithm is supported by this package.
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// It returns true if the algorithm is explicitly supported (SHA256, SHA512) or if
// it's an empty string or digest.Canonical (both treated as SHA256 default).
// It returns false for any other algorithm including SHA384, MD5, etc.
//
// This is a pure function with no side effects and is thread-safe for concurrent
// calls from multiple goroutines. It is typically used for validation before
// calling TmpSetDigestForNewObjects().
func IsSupportedDigestAlgorithm(algorithm digest.Algorithm) bool {
	// Handle special cases first
	if algorithm == "" || algorithm == digest.Canonical {
		return true // Empty string and canonical are treated as default (SHA256)
	}

	// Check against the list of supported algorithms
	supportedAlgorithms := GetSupportedDigestAlgorithms()
	for _, supported := range supportedAlgorithms {
		if algorithm == supported {
			return true
		}
	}
	return false
}

// GetSupportedDigestAlgorithms returns a list of all supported digest algorithms.
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// It returns a slice containing all algorithms that can be used with
// TmpSetDigestForNewObjects(). Currently returns [SHA256, SHA512].
//
// This is a pure function with no side effects and is thread-safe for concurrent
// calls from multiple goroutines. The returned slice should not be modified by
// callers. It is typically used for validation and algorithm enumeration.
func GetSupportedDigestAlgorithms() []digest.Algorithm {
	return []digest.Algorithm{
		digest.SHA256,
		digest.SHA512,
	}
}

// GetDigestAlgorithmName returns a human-readable name for the algorithm.
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// It returns a standardized uppercase name for supported algorithms. The function
// is case-insensitive, so "sha256", "SHA256", "Sha256" all return "SHA256".
// It returns "SHA256 (canonical)" for digest.Canonical and "unknown" for
// unsupported algorithms.
//
// This is a pure function with no side effects and is thread-safe for concurrent
// calls from multiple goroutines. It is typically used for logging and user-facing
// display purposes.
func GetDigestAlgorithmName(algorithm digest.Algorithm) string {
	// Normalize to lowercase for case-insensitive matching
	normalized := strings.ToLower(algorithm.String())

	switch normalized {
	case "sha256":
		return "SHA256"
	case "sha512":
		return "SHA512"
	default:
		if algorithm == digest.Canonical {
			return "SHA256 (canonical)"
		}
		return "unknown"
	}
}

// GetDigestAlgorithmExpectedLength returns the expected hex string length for a given algorithm.
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// It returns (length, true) for supported algorithms with known hex lengths.
// SHA256 returns (64, true) and SHA512 returns (128, true). It returns (0, false)
// for unsupported or unknown algorithms. The length represents the number of hex
// characters in the digest string.
//
// This is a pure function with no side effects and is thread-safe for concurrent
// calls from multiple goroutines. It is typically used for validation and algorithm
// detection from hex string lengths.
func GetDigestAlgorithmExpectedLength(algorithm digest.Algorithm) (int, bool) {
	switch algorithm {
	case digest.SHA256:
		return 64, true
	case digest.SHA512:
		return 128, true
	default:
		// For future algorithms, this function can be extended
		// to support additional algorithms as they are added
		return 0, false
	}
}

// DetectDigestAlgorithmFromLength attempts to detect the digest algorithm from a hex string length.
//
// WARNING: This function is part of a WIP package intended only for Podman,
// Buildah, and Skopeo. Do not use in external applications.
//
// It returns (algorithm, true) if a supported algorithm matches the given length,
// or (empty, false) if no supported algorithm matches the length. It checks all
// supported algorithms against their expected hex lengths.
//
// This is a pure function with no side effects and is thread-safe for concurrent
// calls from multiple goroutines. It is typically used for reverse lookup when
// only the hex string length is known. Ambiguous lengths (if any) will return
// the first matching algorithm.
func DetectDigestAlgorithmFromLength(length int) (digest.Algorithm, bool) {
	for _, algorithm := range GetSupportedDigestAlgorithms() {
		if expectedLength, supported := GetDigestAlgorithmExpectedLength(algorithm); supported && expectedLength == length {
			return algorithm, true
		}
	}
	return digest.Algorithm(""), false
}
