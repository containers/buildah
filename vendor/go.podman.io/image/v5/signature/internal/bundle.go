package internal

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	protobundle "github.com/sigstore/protobuf-specs/gen/pb-go/bundle/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// Bundle wraps sigstore-go's bundle.Bundle with additional helper methods
// for container-libs integration.
type Bundle struct {
	*bundle.Bundle
	rawBytes []byte
}

// LoadBundle parses a sigstore bundle from JSON bytes.
// This is the primary entry point for bundle parsing.
func LoadBundle(bundleBytes []byte) (*Bundle, error) {
	b := &bundle.Bundle{
		Bundle: new(protobundle.Bundle),
	}
	if err := b.UnmarshalJSON(bundleBytes); err != nil {
		return nil, NewInvalidSignatureError(fmt.Sprintf("parsing sigstore bundle: %v", err))
	}
	return &Bundle{Bundle: b, rawBytes: bundleBytes}, nil
}

// RawBytes returns the original JSON bytes of the bundle.
func (b *Bundle) RawBytes() []byte {
	return b.rawBytes
}

// IsDSSE returns true if the bundle contains a DSSE envelope (attestation).
func (b *Bundle) IsDSSE() bool {
	return b.Bundle.GetDsseEnvelope() != nil
}

// IsMessageSignature returns true if the bundle contains a message signature.
func (b *Bundle) IsMessageSignature() bool {
	return b.Bundle.GetMessageSignature() != nil
}

// GetCertificate extracts the signing certificate from the bundle.
// Returns nil if no certificate is present (e.g., public key signed).
func (b *Bundle) GetCertificate() (*x509.Certificate, error) {
	vc, err := b.Bundle.VerificationContent()
	if err != nil {
		return nil, nil // No verification content
	}
	return vc.Certificate(), nil
}

// GetCertificatePEM returns the signing certificate as PEM-encoded bytes.
// Returns nil if no certificate is present.
func (b *Bundle) GetCertificatePEM() ([]byte, error) {
	vm := b.Bundle.GetVerificationMaterial()
	if vm == nil {
		return nil, nil
	}

	// Try X509CertificateChain first (preferred format)
	if chain := vm.GetX509CertificateChain(); chain != nil {
		certs := chain.GetCertificates()
		if len(certs) > 0 {
			return pem.EncodeToMemory(&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: certs[0].GetRawBytes(),
			}), nil
		}
	}

	// Fall back to single Certificate
	if cert := vm.GetCertificate(); cert != nil {
		return pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.GetRawBytes(),
		}), nil
	}

	return nil, nil
}

// GetIntermediateChainPEM returns the intermediate certificate chain as PEM-encoded bytes.
// Returns nil if no intermediate chain is present.
func (b *Bundle) GetIntermediateChainPEM() ([]byte, error) {
	vm := b.Bundle.GetVerificationMaterial()
	if vm == nil {
		return nil, nil
	}

	// Only X509CertificateChain has intermediates
	chain := vm.GetX509CertificateChain()
	if chain == nil {
		return nil, nil
	}
	certs := chain.GetCertificates()
	if len(certs) <= 1 {
		return nil, nil // No intermediates
	}

	// All certs after the first are intermediates
	var chainPEM []byte
	for _, cert := range certs[1:] {
		chainPEM = append(chainPEM, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.GetRawBytes(),
		})...)
	}
	return chainPEM, nil
}

// GetTlogEntryForRekor returns the transparency log entry data needed for Rekor SET verification.
// Returns the SET bytes and the canonicalized body for verification.
func (b *Bundle) GetTlogEntryForRekor() (setSET []byte, body []byte, err error) {
	vm := b.Bundle.GetVerificationMaterial()
	if vm == nil {
		return nil, nil, nil
	}
	entries := vm.GetTlogEntries()
	if len(entries) == 0 {
		return nil, nil, nil
	}
	entry := entries[0]

	// Get the SET (Signed Entry Timestamp)
	inclusionPromise := entry.GetInclusionPromise()
	if inclusionPromise != nil {
		setSET = inclusionPromise.GetSignedEntryTimestamp()
	}

	// Get the canonicalized body
	body = entry.GetCanonicalizedBody()

	return setSET, body, nil
}

// GetPublicKey extracts the public key from the bundle.
// For certificate-based bundles, returns the certificate's public key.
func (b *Bundle) GetPublicKey() (crypto.PublicKey, error) {
	cert, err := b.GetCertificate()
	if err != nil {
		return nil, err
	}
	if cert != nil {
		return cert.PublicKey, nil
	}

	// Check for public key hint
	if b.Bundle.GetVerificationMaterial().GetPublicKey() != nil {
		return nil, NewInvalidSignatureError("bundle contains publicKey hint but external key lookup is required")
	}

	return nil, NewInvalidSignatureError("no public key or certificate found in bundle")
}

// GetEnvelopePayload returns the DSSE envelope payload if present.
// Returns payload bytes, payload type, and any error.
func (b *Bundle) GetEnvelopePayload() ([]byte, string, error) {
	envelope := b.Bundle.GetDsseEnvelope()
	if envelope == nil {
		return nil, "", nil
	}
	// In protobuf, Payload is already []byte
	return envelope.Payload, envelope.PayloadType, nil
}

// GetMessageSignatureBytes returns the signature and digest from a message signature bundle.
func (b *Bundle) GetMessageSignatureBytes() (signature []byte, digest []byte, err error) {
	msgSig := b.Bundle.GetMessageSignature()
	if msgSig == nil {
		return nil, nil, nil
	}
	return msgSig.GetSignature(), msgSig.GetMessageDigest().GetDigest(), nil
}

// HasTlogEntry returns true if the bundle contains transparency log entries.
func (b *Bundle) HasTlogEntry() bool {
	vm := b.Bundle.GetVerificationMaterial()
	return vm != nil && len(vm.GetTlogEntries()) > 0
}

// GetIntegratedTime returns the integrated time from the first tlog entry.
func (b *Bundle) GetIntegratedTime() (time.Time, error) {
	vm := b.Bundle.GetVerificationMaterial()
	if vm == nil || len(vm.GetTlogEntries()) == 0 {
		return time.Time{}, nil
	}
	entry := vm.GetTlogEntries()[0]
	return time.Unix(entry.GetIntegratedTime(), 0), nil
}

// BundleVerificationResult contains the results of verifying a sigstore bundle.
type BundleVerificationResult struct {
	// Certificate is the verified signing certificate, if present
	Certificate *x509.Certificate
	// PublicKey is the public key that verified the signature
	PublicKey crypto.PublicKey
	// IntegratedTime is the time the entry was integrated into the transparency log
	IntegratedTime time.Time
	// Statement is the verified in-toto statement, if this is an attestation bundle
	Statement json.RawMessage
	// EnvelopePayload is the raw DSSE envelope payload for attestations
	EnvelopePayload []byte
	// EnvelopePayloadType is the DSSE payload type
	EnvelopePayloadType string
}

// BundleVerifyOptions contains options for bundle verification.
type BundleVerifyOptions struct {
	// TrustedRoot is the trusted root for verification (Fulcio CAs, Rekor keys, etc.)
	TrustedRoot root.TrustedMaterial
	// ExpectedIdentity is the expected certificate identity (for Fulcio verification)
	ExpectedIdentity *verify.CertificateIdentity
	// ExpectedDigest is the expected artifact digest in hex
	ExpectedDigest string
	// PublicKeys are trusted public keys for non-Fulcio verification
	PublicKeys []crypto.PublicKey
	// RekorPublicKeys are the Rekor transparency log public keys
	RekorPublicKeys []*ecdsa.PublicKey
	// SkipTlogVerification skips transparency log verification
	SkipTlogVerification bool
}

// VerifyBundle verifies a sigstore bundle using the sigstore-go library.
// This is the primary verification method.
func VerifyBundle(bundleBytes []byte, opts BundleVerifyOptions) (*BundleVerificationResult, error) {
	b, err := LoadBundle(bundleBytes)
	if err != nil {
		return nil, err
	}

	// If we have a trusted root, use sigstore-go's full verification
	if opts.TrustedRoot != nil {
		return verifyWithTrustedRoot(b, opts)
	}

	// Otherwise, use public key verification
	if len(opts.PublicKeys) > 0 {
		return verifyWithPublicKeys(b, opts)
	}

	return nil, NewInvalidSignatureError("no verification method available: need either TrustedRoot or PublicKeys")
}

// verifyWithTrustedRoot verifies using sigstore-go's full verification with a trusted root.
func verifyWithTrustedRoot(b *Bundle, opts BundleVerifyOptions) (*BundleVerificationResult, error) {
	verifierOpts := []verify.VerifierOption{}

	if !opts.SkipTlogVerification {
		verifierOpts = append(verifierOpts, verify.WithTransparencyLog(1))
	}

	verifier, err := verify.NewSignedEntityVerifier(opts.TrustedRoot, verifierOpts...)
	if err != nil {
		return nil, NewInvalidSignatureError(fmt.Sprintf("creating verifier: %v", err))
	}

	// Build policy - NewPolicy requires an ArtifactPolicyOption as first argument
	var artifactPolicy verify.ArtifactPolicyOption
	if opts.ExpectedDigest != "" {
		digestBytes, err := hex.DecodeString(opts.ExpectedDigest)
		if err != nil {
			return nil, NewInvalidSignatureError(fmt.Sprintf("decoding expected digest: %v", err))
		}
		artifactPolicy = verify.WithArtifactDigest("sha256", digestBytes)
	} else {
		// If no expected digest, we still need an artifact policy
		// Use WithoutArtifactUnsafe for cases where digest matching is done elsewhere
		artifactPolicy = verify.WithoutArtifactUnsafe()
	}

	policyOpts := []verify.PolicyOption{}
	if opts.ExpectedIdentity != nil {
		policyOpts = append(policyOpts, verify.WithCertificateIdentity(*opts.ExpectedIdentity))
	}

	policy := verify.NewPolicy(artifactPolicy, policyOpts...)

	result, err := verifier.Verify(b.Bundle, policy)
	if err != nil {
		return nil, NewInvalidSignatureError(fmt.Sprintf("verification failed: %v", err))
	}

	return extractVerificationResult(b, result)
}

// verifyWithPublicKeys verifies the bundle signature using provided public keys.
func verifyWithPublicKeys(b *Bundle, opts BundleVerifyOptions) (*BundleVerificationResult, error) {
	if b.IsDSSE() {
		return verifyDSSEWithPublicKeys(b, opts.PublicKeys)
	}

	return verifyMessageSignatureWithPublicKeys(b, opts)
}

// verifyDSSEWithPublicKeys verifies a DSSE envelope using provided public keys.
func verifyDSSEWithPublicKeys(b *Bundle, publicKeys []crypto.PublicKey) (*BundleVerificationResult, error) {
	envelope := b.Bundle.GetDsseEnvelope()
	if envelope == nil {
		return nil, NewInvalidSignatureError("bundle does not contain a DSSE envelope")
	}

	if len(envelope.Signatures) == 0 {
		return nil, NewInvalidSignatureError("DSSE envelope has no signatures")
	}

	// In protobuf, Payload is already []byte
	payload := envelope.Payload

	// Compute PAE
	paeBytes := ComputePAE(envelope.PayloadType, payload)

	// Try each public key
	for _, pk := range publicKeys {
		for _, sig := range envelope.Signatures {
			// In protobuf, Sig is already []byte
			sigBytes := sig.Sig

			if err := verifySignature(pk, paeBytes, sigBytes); err == nil {
				return &BundleVerificationResult{
					PublicKey:           pk,
					EnvelopePayload:     payload,
					EnvelopePayloadType: envelope.PayloadType,
				}, nil
			}
		}
	}

	return nil, NewInvalidSignatureError("DSSE signature verification failed with all provided keys")
}

// verifyMessageSignatureWithPublicKeys verifies a message signature using public keys.
func verifyMessageSignatureWithPublicKeys(b *Bundle, opts BundleVerifyOptions) (*BundleVerificationResult, error) {
	msgSig := b.Bundle.GetMessageSignature()
	if msgSig == nil {
		return nil, NewInvalidSignatureError("bundle does not contain a message signature")
	}

	signature := msgSig.GetSignature()
	digest := msgSig.GetMessageDigest().GetDigest()

	for _, pk := range opts.PublicKeys {
		if err := verifySignature(pk, digest, signature); err == nil {
			intTime, _ := b.GetIntegratedTime()
			return &BundleVerificationResult{
				PublicKey:      pk,
				IntegratedTime: intTime,
			}, nil
		}
	}

	return nil, NewInvalidSignatureError("message signature verification failed with all provided keys")
}

// extractVerificationResult extracts results from sigstore-go verification.
func extractVerificationResult(b *Bundle, result *verify.VerificationResult) (*BundleVerificationResult, error) {
	vr := &BundleVerificationResult{}

	// Extract certificate
	vc, err := b.Bundle.VerificationContent()
	if err == nil && vc != nil {
		if cert := vc.Certificate(); cert != nil {
			vr.Certificate = cert
			vr.PublicKey = cert.PublicKey
		}
	}

	// Extract timestamps
	if result.VerifiedTimestamps != nil && len(result.VerifiedTimestamps) > 0 {
		vr.IntegratedTime = result.VerifiedTimestamps[0].Timestamp
	}

	// Extract statement for attestations
	if result.Statement != nil {
		statementBytes, err := json.Marshal(result.Statement)
		if err == nil {
			vr.Statement = statementBytes
		}
	}

	// Extract envelope payload
	if b.IsDSSE() {
		payload, payloadType, _ := b.GetEnvelopePayload()
		vr.EnvelopePayload = payload
		vr.EnvelopePayloadType = payloadType
	}

	return vr, nil
}

// verifySignature verifies a raw signature over data using a public key.
func verifySignature(publicKey crypto.PublicKey, data, signature []byte) error {
	switch pk := publicKey.(type) {
	case *ecdsa.PublicKey:
		return verifyECDSA(pk, data, signature)
	default:
		return NewInvalidSignatureError(fmt.Sprintf("unsupported key type: %T", publicKey))
	}
}

// verifyECDSA verifies an ECDSA signature.
func verifyECDSA(publicKey *ecdsa.PublicKey, data, signature []byte) error {
	// Try direct verification first (data is already a hash)
	if ecdsa.VerifyASN1(publicKey, data, signature) {
		return nil
	}

	// Try hashing the data
	hash := sha256Hash(data)
	if ecdsa.VerifyASN1(publicKey, hash, signature) {
		return nil
	}

	return NewInvalidSignatureError("ECDSA signature verification failed")
}

// sha256Hash computes SHA256 hash.
func sha256Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// ComputePAE computes the Pre-Authentication Encoding for DSSE verification.
func ComputePAE(payloadType string, payload []byte) []byte {
	return []byte(fmt.Sprintf("DSSEv1 %d %s %d %s",
		len(payloadType), payloadType,
		len(payload), string(payload)))
}

// ConvertBundleToLegacyFormat extracts components for legacy verification.
// This is used for backwards compatibility with existing verification code.
func ConvertBundleToLegacyFormat(b *Bundle) (keyOrCertPEM []byte, base64Sig string, payload []byte, err error) {
	// Get certificate
	cert, err := b.GetCertificate()
	if err != nil {
		return nil, "", nil, err
	}
	if cert != nil {
		keyOrCertPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
	}

	// Get signature
	if b.IsMessageSignature() {
		msgSig := b.Bundle.GetMessageSignature()
		base64Sig = base64.StdEncoding.EncodeToString(msgSig.GetSignature())
	} else if b.IsDSSE() {
		envelope := b.Bundle.GetDsseEnvelope()
		if len(envelope.Signatures) > 0 {
			// Sig is []byte in protobuf, encode to base64 string
			base64Sig = base64.StdEncoding.EncodeToString(envelope.Signatures[0].Sig)
		}
	}

	// Get payload
	if b.IsDSSE() {
		payload, _, err = b.GetEnvelopePayload()
	} else if b.IsMessageSignature() {
		_, payload, err = b.GetMessageSignatureBytes()
	}

	return keyOrCertPEM, base64Sig, payload, err
}

// String returns a debug representation of the bundle.
func (b *Bundle) String() string {
	hasCert := false
	if cert, _ := b.GetCertificate(); cert != nil {
		hasCert = true
	}
	return fmt.Sprintf("Bundle{isDSSE=%v, hasCert=%v, hasTlog=%v}",
		b.IsDSSE(), hasCert, b.HasTlogEntry())
}
