package digestvalidation

import (
	"fmt"

	"github.com/opencontainers/go-digest"
	supportedDigests "go.podman.io/storage/pkg/supported-digests"
)

// ValidateBlobAgainstDigest validates that the provided blob matches the expected digest.
// It performs comprehensive validation to prevent panics from malformed digests or unsupported algorithms.
//
// This function handles the following validation steps:
// 1. Empty digest check
// 2. Digest format validation using digest.Parse()
// 3. Algorithm validation
// 4. Algorithm support validation using supported-digests package
// 5. Content validation by computing and comparing digests
//
// Returns an error if any validation step fails, with specific error messages for different failure cases.
func ValidateBlobAgainstDigest(blob []byte, expectedDigest digest.Digest) error {
	// Validate the digest format to prevent panics from invalid digests
	if expectedDigest == "" {
		return fmt.Errorf("expected digest is empty")
	}

	// Parse the digest to validate its format before calling Algorithm()
	parsedDigest, err := digest.Parse(expectedDigest.String())
	if err != nil {
		return fmt.Errorf("invalid digest format: %s", expectedDigest)
	}

	algorithm := parsedDigest.Algorithm()
	if algorithm == "" {
		return fmt.Errorf("invalid digest algorithm: %s", expectedDigest)
	}

	// Validate that the algorithm is supported to prevent panics from FromBytes
	if !supportedDigests.IsSupportedDigestAlgorithm(algorithm) {
		return fmt.Errorf("unsupported digest algorithm: %s (supported: %v)", algorithm, supportedDigests.GetSupportedDigestAlgorithms())
	}

	// Compute the actual digest of the blob
	computedDigest := algorithm.FromBytes(blob)
	if computedDigest != expectedDigest {
		return fmt.Errorf("blob digest mismatch: expected %s, got %s", expectedDigest, computedDigest)
	}

	return nil
}
