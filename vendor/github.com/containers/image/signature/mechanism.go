// Note: Consider the API unstable until the code supports at least three different image formats or transports.

package signature

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/mtrmac/gpgme"
	"golang.org/x/crypto/openpgp"
)

// SigningMechanism abstracts a way to sign binary blobs and verify their signatures.
// FIXME: Eventually expand on keyIdentity (namespace them between mechanisms to
// eliminate ambiguities, support CA signatures and perhaps other key properties)
type SigningMechanism interface {
	// ImportKeysFromBytes imports public keys from the supplied blob and returns their identities.
	// The blob is assumed to have an appropriate format (the caller is expected to know which one).
	// NOTE: This may modify long-term state (e.g. key storage in a directory underlying the mechanism).
	ImportKeysFromBytes(blob []byte) ([]string, error)
	// Sign creates a (non-detached) signature of input using keyidentity
	Sign(input []byte, keyIdentity string) ([]byte, error)
	// Verify parses unverifiedSignature and returns the content and the signer's identity
	Verify(unverifiedSignature []byte) (contents []byte, keyIdentity string, err error)
	// UntrustedSignatureContents returns UNTRUSTED contents of the signature WITHOUT ANY VERIFICATION,
	// along with a short identifier of the key used for signing.
	// WARNING: The short key identifier (which correponds to "Key ID" for OpenPGP keys)
	// is NOT the same as a "key identity" used in other calls ot this interface, and
	// the values may have no recognizable relationship if the public key is not available.
	UntrustedSignatureContents(untrustedSignature []byte) (untrustedContents []byte, shortKeyIdentifier string, err error)
}

// A GPG/OpenPGP signing mechanism.
type gpgSigningMechanism struct {
	ctx *gpgme.Context
}

// NewGPGSigningMechanism returns a new GPG/OpenPGP signing mechanism.
func NewGPGSigningMechanism() (SigningMechanism, error) {
	return newGPGSigningMechanismInDirectory("")
}

// newGPGSigningMechanismInDirectory returns a new GPG/OpenPGP signing mechanism, using optionalDir if not empty.
func newGPGSigningMechanismInDirectory(optionalDir string) (SigningMechanism, error) {
	ctx, err := gpgme.New()
	if err != nil {
		return nil, err
	}
	if err = ctx.SetProtocol(gpgme.ProtocolOpenPGP); err != nil {
		return nil, err
	}
	if optionalDir != "" {
		err := ctx.SetEngineInfo(gpgme.ProtocolOpenPGP, "", optionalDir)
		if err != nil {
			return nil, err
		}
	}
	ctx.SetArmor(false)
	ctx.SetTextMode(false)
	return gpgSigningMechanism{ctx: ctx}, nil
}

// ImportKeysFromBytes implements SigningMechanism.ImportKeysFromBytes
func (m gpgSigningMechanism) ImportKeysFromBytes(blob []byte) ([]string, error) {
	inputData, err := gpgme.NewDataBytes(blob)
	if err != nil {
		return nil, err
	}
	res, err := m.ctx.Import(inputData)
	if err != nil {
		return nil, err
	}
	keyIdentities := []string{}
	for _, i := range res.Imports {
		if i.Result == nil {
			keyIdentities = append(keyIdentities, i.Fingerprint)
		}
	}
	return keyIdentities, nil
}

// Sign implements SigningMechanism.Sign
func (m gpgSigningMechanism) Sign(input []byte, keyIdentity string) ([]byte, error) {
	key, err := m.ctx.GetKey(keyIdentity, true)
	if err != nil {
		return nil, err
	}
	inputData, err := gpgme.NewDataBytes(input)
	if err != nil {
		return nil, err
	}
	var sigBuffer bytes.Buffer
	sigData, err := gpgme.NewDataWriter(&sigBuffer)
	if err != nil {
		return nil, err
	}
	if err = m.ctx.Sign([]*gpgme.Key{key}, inputData, sigData, gpgme.SigModeNormal); err != nil {
		return nil, err
	}
	return sigBuffer.Bytes(), nil
}

// Verify implements SigningMechanism.Verify
func (m gpgSigningMechanism) Verify(unverifiedSignature []byte) (contents []byte, keyIdentity string, err error) {
	signedBuffer := bytes.Buffer{}
	signedData, err := gpgme.NewDataWriter(&signedBuffer)
	if err != nil {
		return nil, "", err
	}
	unverifiedSignatureData, err := gpgme.NewDataBytes(unverifiedSignature)
	if err != nil {
		return nil, "", err
	}
	_, sigs, err := m.ctx.Verify(unverifiedSignatureData, nil, signedData)
	if err != nil {
		return nil, "", err
	}
	if len(sigs) != 1 {
		return nil, "", InvalidSignatureError{msg: fmt.Sprintf("Unexpected GPG signature count %d", len(sigs))}
	}
	sig := sigs[0]
	// This is sig.Summary == gpgme.SigSumValid except for key trust, which we handle ourselves
	if sig.Status != nil || sig.Validity == gpgme.ValidityNever || sig.ValidityReason != nil || sig.WrongKeyUsage {
		// FIXME: Better error reporting eventually
		return nil, "", InvalidSignatureError{msg: fmt.Sprintf("Invalid GPG signature: %#v", sig)}
	}
	return signedBuffer.Bytes(), sig.Fingerprint, nil
}

// UntrustedSignatureContents returns UNTRUSTED contents of the signature WITHOUT ANY VERIFICATION,
// along with a short identifier of the key used for signing.
// WARNING: The short key identifier (which correponds to "Key ID" for OpenPGP keys)
// is NOT the same as a "key identity" used in other calls ot this interface, and
// the values may have no recognizable relationship if the public key is not available.
func (m gpgSigningMechanism) UntrustedSignatureContents(untrustedSignature []byte) (untrustedContents []byte, shortKeyIdentifier string, err error) {
	// This uses the Golang-native OpenPGP implementation instead of gpgme because we are not doing any cryptography.
	md, err := openpgp.ReadMessage(bytes.NewReader(untrustedSignature), openpgp.EntityList{}, nil, nil)
	if err != nil {
		return nil, "", err
	}
	if !md.IsSigned {
		return nil, "", errors.New("The input is not a signature")
	}
	content, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		// Coverage: An error during reading the body can happen only if
		// 1) the message is encrypted, which is not our case (and we don’t give ReadMessage the key
		// to decrypt the contents anyway), or
		// 2) the message is signed AND we give ReadMessage a correspnding public key, which we don’t.
		return nil, "", err
	}

	// Uppercase the key ID for minimal consistency with the gpgme-returned fingerprints
	// (but note that key ID is a suffix of the fingerprint only for V4 keys, not V3)!
	return content, strings.ToUpper(fmt.Sprintf("%016X", md.SignedByKeyId)), nil
}
