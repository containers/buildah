package signature

import (
	"bytes"
	"encoding/json"
	"maps"
	"strings"
)

const (
	// from sigstore/cosign/pkg/types.SimpleSigningMediaType
	SigstoreSignatureMIMEType = "application/vnd.dev.cosign.simplesigning.v1+json"
	// from sigstore/cosign/pkg/oci/static.SignatureAnnotationKey
	SigstoreSignatureAnnotationKey = "dev.cosignproject.cosign/signature"
	// from sigstore/cosign/pkg/oci/static.BundleAnnotationKey
	SigstoreSETAnnotationKey = "dev.sigstore.cosign/bundle"
	// from sigstore/cosign/pkg/oci/static.CertificateAnnotationKey
	SigstoreCertificateAnnotationKey = "dev.sigstore.cosign/certificate"
	// from sigstore/cosign/pkg/oci/static.ChainAnnotationKey
	SigstoreIntermediateCertificateChainAnnotationKey = "dev.sigstore.cosign/chain"

	// Sigstore bundle format media types
	// See https://github.com/sigstore/protobuf-specs for the bundle specification
	SigstoreBundleMIMEType = "application/vnd.dev.sigstore.bundle.v0.3+json"
	// SigstoreBundleMediaTypePrefix is the prefix for all sigstore bundle versions
	SigstoreBundleMediaTypePrefix = "application/vnd.dev.sigstore.bundle"

	// DSSEPayloadType is the payload type used in DSSE envelopes for in-toto attestations
	DSSEPayloadType = "application/vnd.in-toto+json"
)

// IsSigstoreBundleMediaType returns true if the media type indicates a Cosign v3 sigstore bundle format
func IsSigstoreBundleMediaType(mediaType string) bool {
	return strings.HasPrefix(mediaType, SigstoreBundleMediaTypePrefix)
}

// IsSigstoreSignatureMediaType returns true if the media type indicates any sigstore signature format
// (either legacy simple signing or new bundle format)
func IsSigstoreSignatureMediaType(mediaType string) bool {
	return mediaType == SigstoreSignatureMIMEType || IsSigstoreBundleMediaType(mediaType)
}

// Sigstore is a github.com/cosign/cosign signature.
// For the persistent-storage format used for blobChunk(), we want
// a degree of forward compatibility against unexpected field changes
// (as has happened before), which is why this data type
// contains just a payload + annotations (including annotations
// that we don’t recognize or support), instead of individual fields
// for the known annotations.
type Sigstore struct {
	untrustedMIMEType    string
	untrustedPayload     []byte
	untrustedAnnotations map[string]string
}

// sigstoreJSONRepresentation needs the files to be public, which we don’t want for
// the main Sigstore type.
type sigstoreJSONRepresentation struct {
	UntrustedMIMEType    string            `json:"mimeType"`
	UntrustedPayload     []byte            `json:"payload"`
	UntrustedAnnotations map[string]string `json:"annotations"`
}

// SigstoreFromComponents returns a Sigstore object from its components.
func SigstoreFromComponents(untrustedMimeType string, untrustedPayload []byte, untrustedAnnotations map[string]string) Sigstore {
	return Sigstore{
		untrustedMIMEType:    untrustedMimeType,
		untrustedPayload:     bytes.Clone(untrustedPayload),
		untrustedAnnotations: maps.Clone(untrustedAnnotations),
	}
}

// sigstoreFromBlobChunk converts a Sigstore signature, as returned by Sigstore.blobChunk, into a Sigstore object.
func sigstoreFromBlobChunk(blobChunk []byte) (Sigstore, error) {
	var v sigstoreJSONRepresentation
	if err := json.Unmarshal(blobChunk, &v); err != nil {
		return Sigstore{}, err
	}
	return SigstoreFromComponents(v.UntrustedMIMEType,
		v.UntrustedPayload,
		v.UntrustedAnnotations), nil
}

func (s Sigstore) FormatID() FormatID {
	return SigstoreFormat
}

// blobChunk returns a representation of signature as a []byte, suitable for long-term storage.
// Almost everyone should use signature.Blob() instead.
func (s Sigstore) blobChunk() ([]byte, error) {
	return json.Marshal(sigstoreJSONRepresentation{
		UntrustedMIMEType:    s.UntrustedMIMEType(),
		UntrustedPayload:     s.UntrustedPayload(),
		UntrustedAnnotations: s.UntrustedAnnotations(),
	})
}

func (s Sigstore) UntrustedMIMEType() string {
	return s.untrustedMIMEType
}

func (s Sigstore) UntrustedPayload() []byte {
	return bytes.Clone(s.untrustedPayload)
}

func (s Sigstore) UntrustedAnnotations() map[string]string {
	return maps.Clone(s.untrustedAnnotations)
}
