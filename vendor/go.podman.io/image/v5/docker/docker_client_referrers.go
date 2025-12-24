package docker

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	digest "github.com/opencontainers/go-digest"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"go.podman.io/image/v5/docker/reference"
	"go.podman.io/image/v5/internal/iolimits"
	"go.podman.io/image/v5/manifest"
)

const (
	// OCI 1.1 referrers API path for fetching artifacts that reference a manifest
	// See https://github.com/opencontainers/distribution-spec/blob/main/spec.md#listing-referrers
	referrersPath = "/v2/%s/referrers/%s"
)

// getReferrers fetches the referrers index for a manifest using the OCI 1.1 referrers API.
// If artifactType is non-empty, it filters the results to only include referrers of that type.
// Returns (nil, nil) if the registry does not support the referrers API or no referrers exist.
func (c *dockerClient) getReferrers(ctx context.Context, ref dockerReference, manifestDigest digest.Digest, artifactType string) (*manifest.OCI1Index, error) {
	if err := manifestDigest.Validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf(referrersPath, reference.Path(ref.ref), manifestDigest.String())
	if artifactType != "" {
		path += "?artifactType=" + url.QueryEscape(artifactType)
	}

	headers := map[string][]string{
		"Accept": {imgspecv1.MediaTypeImageIndex},
	}

	logrus.Debugf("Fetching referrers for %s via %s", manifestDigest.String(), path)
	res, err := c.makeRequest(ctx, http.MethodGet, path, headers, nil, v2Auth, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// 404 means the API is not supported or no referrers exist via the API
	// Fall back to the tag-based referrers scheme (OCI 1.1 fallback)
	if res.StatusCode == http.StatusNotFound {
		logrus.Debugf("Referrers API returned 404 for %s, trying tag-based fallback", manifestDigest.String())
		return c.getReferrersFromTag(ctx, ref, manifestDigest)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching referrers for %s: %w", manifestDigest.String(), registryHTTPResponseToError(res))
	}

	body, err := iolimits.ReadAtMost(res.Body, iolimits.MaxManifestBodySize)
	if err != nil {
		return nil, err
	}

	index, err := manifest.OCI1IndexFromManifest(body)
	if err != nil {
		return nil, fmt.Errorf("parsing referrers index for %s: %w", manifestDigest.String(), err)
	}

	logrus.Debugf("Found %d referrers for %s", len(index.Manifests), manifestDigest.String())
	return index, nil
}

// getReferrersFromTag implements the OCI 1.1 referrers tag-based fallback scheme.
// When the referrers API is not supported, referrers are stored as an image index
// at the tag "sha256-<digest>" (replacing ":" with "-").
// See https://github.com/opencontainers/distribution-spec/blob/main/spec.md#referrers-tag-schema
func (c *dockerClient) getReferrersFromTag(ctx context.Context, ref dockerReference, manifestDigest digest.Digest) (*manifest.OCI1Index, error) {
	// Convert digest to tag format: sha256:abc123 -> sha256-abc123
	tagName := strings.ReplaceAll(manifestDigest.String(), ":", "-")

	logrus.Debugf("Trying referrers tag fallback: %s", tagName)

	// Fetch the manifest at this tag
	path := fmt.Sprintf(manifestPath, reference.Path(ref.ref), tagName)
	headers := map[string][]string{
		"Accept": {imgspecv1.MediaTypeImageIndex, imgspecv1.MediaTypeImageManifest},
	}

	res, err := c.makeRequest(ctx, http.MethodGet, path, headers, nil, v2Auth, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// 404 means no referrers tag exists
	if res.StatusCode == http.StatusNotFound {
		logrus.Debugf("No referrers tag found for %s", manifestDigest.String())
		return nil, nil
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching referrers tag for %s: %w", manifestDigest.String(), registryHTTPResponseToError(res))
	}

	body, err := iolimits.ReadAtMost(res.Body, iolimits.MaxManifestBodySize)
	if err != nil {
		return nil, err
	}

	contentType := res.Header.Get("Content-Type")

	// The tag might point to an image index (containing multiple referrers)
	// or a single image manifest (if there's only one referrer)
	if contentType == imgspecv1.MediaTypeImageIndex {
		index, err := manifest.OCI1IndexFromManifest(body)
		if err != nil {
			return nil, fmt.Errorf("parsing referrers index from tag for %s: %w", manifestDigest.String(), err)
		}
		logrus.Debugf("Found %d referrers via tag fallback for %s", len(index.Manifests), manifestDigest.String())
		return index, nil
	}

	// If it's a single manifest, check if it's a referrer (has subject field)
	if contentType == imgspecv1.MediaTypeImageManifest {
		// Parse as OCI manifest to check for subject
		ociMan, err := manifest.OCI1FromManifest(body)
		if err != nil {
			return nil, fmt.Errorf("parsing manifest from referrers tag for %s: %w", manifestDigest.String(), err)
		}

		// Check if this manifest references our target
		if ociMan.Subject != nil && ociMan.Subject.Digest == manifestDigest {
			// Convert to an index with one entry
			desc := imgspecv1.Descriptor{
				MediaType:    contentType,
				Digest:       digest.FromBytes(body),
				Size:         int64(len(body)),
				ArtifactType: ociMan.ArtifactType,
			}
			syntheticIndex := manifest.OCI1IndexFromComponents([]imgspecv1.Descriptor{desc}, nil)
			logrus.Debugf("Found 1 referrer via tag fallback (single manifest) for %s", manifestDigest.String())
			return syntheticIndex, nil
		}
	}

	logrus.Debugf("Referrers tag for %s does not contain valid referrers", manifestDigest.String())
	return nil, nil
}

// getSigstoreReferrers fetches sigstore signature referrers using the OCI 1.1 referrers API.
// It filters for common sigstore artifact types and returns the matching descriptors.
// Returns nil if the referrers API is not supported or no sigstore signatures exist.
func (c *dockerClient) getSigstoreReferrers(ctx context.Context, ref dockerReference, manifestDigest digest.Digest) ([]imgspecv1.Descriptor, error) {
	// First try without artifact type filter to get all referrers
	index, err := c.getReferrers(ctx, ref, manifestDigest, "")
	if err != nil {
		return nil, err
	}
	if index == nil {
		return nil, nil
	}

	// Filter for sigstore-related artifact types
	var sigstoreReferrers []imgspecv1.Descriptor
	for _, desc := range index.Manifests {
		if isSigstoreReferrer(desc) {
			sigstoreReferrers = append(sigstoreReferrers, desc)
		}
	}

	return sigstoreReferrers, nil
}

// isSigstoreReferrer returns true if the descriptor represents a sigstore signature or bundle.
func isSigstoreReferrer(desc imgspecv1.Descriptor) bool {
	// Check artifact type (OCI 1.1 style)
	if desc.ArtifactType != "" {
		if strings.HasPrefix(desc.ArtifactType, "application/vnd.dev.sigstore") ||
			strings.HasPrefix(desc.ArtifactType, "application/vnd.dev.cosign") {
			return true
		}
	}

	// Check media type for legacy compatibility
	if strings.HasPrefix(desc.MediaType, "application/vnd.dev.sigstore") ||
		strings.HasPrefix(desc.MediaType, "application/vnd.dev.cosign") {
		return true
	}

	// For OCI referrers fallback tag scheme, the descriptor in the index might have
	// artifactType set to "application/vnd.oci.empty.v1+json" even when the actual
	// manifest is a sigstore bundle. Include these and verify the actual content later.
	if desc.ArtifactType == "application/vnd.oci.empty.v1+json" {
		return true
	}

	return false
}
