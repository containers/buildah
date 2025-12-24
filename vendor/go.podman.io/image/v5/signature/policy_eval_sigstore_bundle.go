package signature

import (
	"context"
	"crypto"
	"errors"
	"fmt"

	digest "github.com/opencontainers/go-digest"
	"go.podman.io/image/v5/internal/private"
	"go.podman.io/image/v5/internal/signature"
	"go.podman.io/image/v5/manifest"
	"go.podman.io/image/v5/signature/internal"
)

// isSignatureAcceptedBundle verifies a Cosign v3 sigstore bundle format signature.
func (pr *prSigstoreSigned) isSignatureAcceptedBundle(ctx context.Context, image private.UnparsedImage, sig signature.Sigstore, trustRoot *sigstoreSignedTrustRoot) (signatureAcceptanceResult, error) {
	bundleBytes := sig.UntrustedPayload()

	// Parse the bundle using sigstore-go
	bundle, err := internal.LoadBundle(bundleBytes)
	if err != nil {
		return sarRejected, err
	}

	// Handle DSSE bundles differently from MessageSignature bundles
	if bundle.IsDSSE() {
		return pr.verifyDSSEBundle(ctx, image, bundle, trustRoot)
	}

	// For MessageSignature bundles, use the legacy conversion approach
	return pr.verifyMessageSignatureBundle(ctx, image, bundle, trustRoot)
}

// verifyMessageSignatureBundle verifies a bundle with MessageSignature format.
func (pr *prSigstoreSigned) verifyMessageSignatureBundle(ctx context.Context, image private.UnparsedImage, bundle *internal.Bundle, trustRoot *sigstoreSignedTrustRoot) (signatureAcceptanceResult, error) {
	// Extract verification material from the bundle
	_, base64Sig, payload, err := internal.ConvertBundleToLegacyFormat(bundle)
	if err != nil {
		return sarRejected, err
	}

	keySources := 0
	if trustRoot.publicKeys != nil {
		keySources++
	}
	if trustRoot.fulcio != nil {
		keySources++
	}
	if trustRoot.pki != nil {
		keySources++
	}

	var publicKeys []crypto.PublicKey
	switch {
	case keySources > 1:
		return sarRejected, errors.New("Internal inconsistency: More than one of public key, Fulcio, or PKI specified")
	case keySources == 0:
		return sarRejected, errors.New("Internal inconsistency: A public key, Fulcio, or PKI must be specified.")

	case trustRoot.publicKeys != nil:
		// Use sigstore-go's bundle verification with public keys
		result, err := internal.VerifyBundle(bundle.RawBytes(), internal.BundleVerifyOptions{
			PublicKeys:           trustRoot.publicKeys,
			RekorPublicKeys:      trustRoot.rekorPublicKeys,
			SkipTlogVerification: trustRoot.rekorPublicKeys == nil,
		})
		if err != nil {
			return sarRejected, err
		}
		publicKeys = []crypto.PublicKey{result.PublicKey}

	case trustRoot.fulcio != nil:
		if trustRoot.rekorPublicKeys == nil {
			return sarRejected, errors.New("Internal inconsistency: Fulcio CA specified without a Rekor public key")
		}

		// Get certificate PEM from the bundle
		certPEM, err := bundle.GetCertificatePEM()
		if err != nil {
			return sarRejected, err
		}
		if certPEM == nil {
			return sarRejected, internal.NewInvalidSignatureError("bundle does not contain a certificate for Fulcio verification")
		}

		// Get intermediate chain PEM if present
		chainPEM, err := bundle.GetIntermediateChainPEM()
		if err != nil {
			return sarRejected, err
		}

		// For Fulcio verification, we need a trusted timestamp from Rekor
		// First, verify the bundle has a tlog entry
		if !bundle.HasTlogEntry() {
			return sarRejected, internal.NewInvalidSignatureError("bundle does not contain a transparency log entry required for Fulcio verification")
		}

		// Get the integrated time from the tlog entry
		integratedTime, err := bundle.GetIntegratedTime()
		if err != nil {
			return sarRejected, err
		}
		if integratedTime.IsZero() {
			return sarRejected, internal.NewInvalidSignatureError("bundle transparency log entry has no integrated time")
		}

		// Verify the Fulcio certificate chain at the tlog integrated time
		pk, err := trustRoot.fulcio.verifyFulcioCertificateAtTime(integratedTime, certPEM, chainPEM)
		if err != nil {
			return sarRejected, err
		}
		publicKeys = []crypto.PublicKey{pk}

	case trustRoot.pki != nil:
		// Get certificate PEM from the bundle
		certPEM, err := bundle.GetCertificatePEM()
		if err != nil {
			return sarRejected, err
		}
		if certPEM == nil {
			return sarRejected, internal.NewInvalidSignatureError("bundle does not contain a certificate for PKI verification")
		}

		// Get intermediate chain PEM if present
		chainPEM, err := bundle.GetIntermediateChainPEM()
		if err != nil {
			return sarRejected, err
		}

		pk, err := verifyPKI(trustRoot.pki, certPEM, chainPEM)
		if err != nil {
			return sarRejected, err
		}
		publicKeys = []crypto.PublicKey{pk}
	}

	if len(publicKeys) == 0 {
		return sarRejected, fmt.Errorf("Internal inconsistency: publicKey not set before verifying sigstore bundle payload")
	}

	// Verify the payload signature using the extracted components
	verifiedPayload, err := internal.VerifySigstorePayload(publicKeys, payload, base64Sig, internal.SigstorePayloadAcceptanceRules{
		ValidateSignedDockerReference: func(ref string) error {
			if !pr.SignedIdentity.matchesDockerReference(image, ref) {
				return PolicyRequirementError(fmt.Sprintf("Signature for identity %q is not accepted", ref))
			}
			return nil
		},
		ValidateSignedDockerManifestDigest: func(digest digest.Digest) error {
			m, _, err := image.Manifest(ctx)
			if err != nil {
				return err
			}
			digestMatches, err := manifest.MatchesDigest(m, digest)
			if err != nil {
				return err
			}
			if !digestMatches {
				return PolicyRequirementError(fmt.Sprintf("Signature for digest %s does not match", digest))
			}
			return nil
		},
	})
	if err != nil {
		return sarRejected, err
	}
	if verifiedPayload == nil {
		return sarRejected, errors.New("internal error: VerifySigstorePayload succeeded but returned no data")
	}

	return sarAccepted, nil
}

// verifyDSSEBundle verifies a bundle with DSSE envelope format.
// DSSE bundles contain attestations (not signatures over simple signing payloads).
func (pr *prSigstoreSigned) verifyDSSEBundle(ctx context.Context, image private.UnparsedImage, bundle *internal.Bundle, trustRoot *sigstoreSignedTrustRoot) (signatureAcceptanceResult, error) {
	keySources := 0
	if trustRoot.publicKeys != nil {
		keySources++
	}
	if trustRoot.fulcio != nil {
		keySources++
	}
	if trustRoot.pki != nil {
		keySources++
	}

	switch {
	case keySources > 1:
		return sarRejected, errors.New("Internal inconsistency: More than one of public key, Fulcio, or PKI specified")
	case keySources == 0:
		return sarRejected, errors.New("Internal inconsistency: A public key, Fulcio, or PKI must be specified.")

	case trustRoot.publicKeys != nil:
		// Use sigstore-go's bundle verification with public keys
		result, err := internal.VerifyBundle(bundle.RawBytes(), internal.BundleVerifyOptions{
			PublicKeys:           trustRoot.publicKeys,
			RekorPublicKeys:      trustRoot.rekorPublicKeys,
			SkipTlogVerification: trustRoot.rekorPublicKeys == nil,
		})
		if err != nil {
			return sarRejected, err
		}
		// Use the verified payload for further validation
		return pr.validateDSSEPayload(ctx, image, result.EnvelopePayload)

	case trustRoot.fulcio != nil:
		if trustRoot.rekorPublicKeys == nil {
			return sarRejected, errors.New("Internal inconsistency: Fulcio CA specified without a Rekor public key")
		}

		// Get certificate PEM from the bundle for Fulcio verification
		certPEM, err := bundle.GetCertificatePEM()
		if err != nil {
			return sarRejected, err
		}
		if certPEM == nil {
			return sarRejected, internal.NewInvalidSignatureError("bundle does not contain a certificate for Fulcio verification")
		}

		// Get intermediate chain PEM if present
		chainPEM, err := bundle.GetIntermediateChainPEM()
		if err != nil {
			return sarRejected, err
		}

		// For Fulcio verification, we need a trusted timestamp from Rekor
		if !bundle.HasTlogEntry() {
			return sarRejected, internal.NewInvalidSignatureError("bundle does not contain a transparency log entry required for Fulcio verification")
		}

		integratedTime, err := bundle.GetIntegratedTime()
		if err != nil {
			return sarRejected, err
		}
		if integratedTime.IsZero() {
			return sarRejected, internal.NewInvalidSignatureError("bundle transparency log entry has no integrated time")
		}

		// Verify the Fulcio certificate chain at the tlog integrated time
		pk, err := trustRoot.fulcio.verifyFulcioCertificateAtTime(integratedTime, certPEM, chainPEM)
		if err != nil {
			return sarRejected, err
		}

		// Now verify the DSSE envelope signature using the verified public key
		result, err := internal.VerifyBundle(bundle.RawBytes(), internal.BundleVerifyOptions{
			PublicKeys:           []crypto.PublicKey{pk},
			SkipTlogVerification: true, // Tlog already verified above via integrated time
		})
		if err != nil {
			return sarRejected, err
		}

		return pr.validateDSSEPayload(ctx, image, result.EnvelopePayload)

	case trustRoot.pki != nil:
		// Get certificate PEM from the bundle
		certPEM, err := bundle.GetCertificatePEM()
		if err != nil {
			return sarRejected, err
		}
		if certPEM == nil {
			return sarRejected, internal.NewInvalidSignatureError("bundle does not contain a certificate for PKI verification")
		}

		// Get intermediate chain PEM if present
		chainPEM, err := bundle.GetIntermediateChainPEM()
		if err != nil {
			return sarRejected, err
		}

		// Verify the PKI certificate chain
		pk, err := verifyPKI(trustRoot.pki, certPEM, chainPEM)
		if err != nil {
			return sarRejected, err
		}

		// Now verify the DSSE envelope signature using the verified public key
		result, err := internal.VerifyBundle(bundle.RawBytes(), internal.BundleVerifyOptions{
			PublicKeys:           []crypto.PublicKey{pk},
			SkipTlogVerification: true,
		})
		if err != nil {
			return sarRejected, err
		}

		return pr.validateDSSEPayload(ctx, image, result.EnvelopePayload)
	}

	return sarRejected, errors.New("Internal inconsistency: no key source matched")
}

// validateDSSEPayload validates the payload from a verified DSSE envelope.
// The payload is typically an in-toto attestation or a simple signing payload.
// The DSSE signature has already been verified, so we just need to validate the content.
func (pr *prSigstoreSigned) validateDSSEPayload(ctx context.Context, image private.UnparsedImage, payload []byte) (signatureAcceptanceResult, error) {
	// Try to parse as a simple signing payload first
	parsedPayload, err := internal.ParseSigstorePayload(payload)
	if err == nil {
		// Validate the parsed payload
		if err := pr.validateParsedPayload(ctx, image, parsedPayload); err != nil {
			return sarRejected, err
		}
		return sarAccepted, nil
	}

	// If it's not a simple signing payload, try to parse as an in-toto attestation
	inTotoPayload, err := internal.ParseInTotoStatement(payload)
	if err == nil {
		// Validate the in-toto statement
		if err := pr.validateInTotoStatement(ctx, image, inTotoPayload); err != nil {
			return sarRejected, err
		}
		return sarAccepted, nil
	}

	// If we can't parse the payload format, reject it
	return sarRejected, internal.NewInvalidSignatureError("DSSE payload is neither a simple signing payload nor an in-toto statement")
}

// validateParsedPayload validates a parsed simple signing payload against the image.
func (pr *prSigstoreSigned) validateParsedPayload(ctx context.Context, image private.UnparsedImage, payload *internal.UntrustedSigstorePayload) error {
	// Validate the docker reference
	if !pr.SignedIdentity.matchesDockerReference(image, payload.UntrustedDockerReference()) {
		return PolicyRequirementError(fmt.Sprintf("Signature for identity %q is not accepted", payload.UntrustedDockerReference()))
	}

	// Validate the manifest digest
	m, _, err := image.Manifest(ctx)
	if err != nil {
		return err
	}
	digestMatches, err := manifest.MatchesDigest(m, payload.UntrustedDockerManifestDigest())
	if err != nil {
		return err
	}
	if !digestMatches {
		return PolicyRequirementError(fmt.Sprintf("Signature for digest %s does not match", payload.UntrustedDockerManifestDigest()))
	}

	return nil
}

// validateInTotoStatement validates an in-toto statement against the image.
func (pr *prSigstoreSigned) validateInTotoStatement(ctx context.Context, image private.UnparsedImage, statement *internal.InTotoStatement) error {
	// Get the manifest digest
	m, _, err := image.Manifest(ctx)
	if err != nil {
		return err
	}
	manifestDigest, err := manifest.Digest(m)
	if err != nil {
		return err
	}

	// Validate that one of the subjects matches the image digest
	if !statement.MatchesDigest(manifestDigest) {
		return PolicyRequirementError(fmt.Sprintf("In-toto statement does not reference digest %s", manifestDigest))
	}

	return nil
}
