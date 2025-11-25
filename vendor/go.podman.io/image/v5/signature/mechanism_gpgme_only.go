//go:build !containers_image_openpgp && !containers_image_sequoia

package signature

import (
	"bytes"
	"context"
	"io"
	"os"

	"github.com/proglottis/gpgme"
)

// newEphemeralGPGSigningMechanism returns a new GPG/OpenPGP signing mechanism which
// recognizes _only_ public keys from the supplied blobs, and returns the identities
// of these keys.
// The caller must call .Close() on the returned SigningMechanism.
func newEphemeralGPGSigningMechanism(ctx context.Context, blobs [][]byte) (signingMechanismWithPassphrase, []string, error) {
	dir, err := os.MkdirTemp("", "containers-ephemeral-gpg-")
	if err != nil {
		return nil, nil, err
	}
	removeDir := true
	defer func() {
		if removeDir {
			os.RemoveAll(dir)
		}
	}()
	gpgmeCtx, err := newGPGMEContext(dir)
	if err != nil {
		return nil, nil, err
	}
	keyIdentities := []string{}
	for _, blob := range blobs {
		ki, err := importKeysFromBytes(ctx, gpgmeCtx, blob)
		if err != nil {
			return nil, nil, err
		}
		keyIdentities = append(keyIdentities, ki...)
	}

	mech := newGPGMESigningMechanism(gpgmeCtx, dir)
	removeDir = false
	return mech, keyIdentities, nil
}

// cancelableReader wraps an io.Reader and checks context cancellation on each Read call.
type cancelableReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *cancelableReader) Read(p []byte) (int, error) {
	// Check if context is cancelled before each read
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	// Check again after read in case cancellation happened during the read
	if err == nil && r.ctx.Err() != nil {
		return n, r.ctx.Err()
	}
	return n, err
}

// importKeysFromBytes imports public keys from the supplied blob and returns their identities.
// The blob is assumed to have an appropriate format (the caller is expected to know which one).
// NOTE: This may modify long-term state (e.g. key storage in a directory underlying the mechanism);
// but we do not make this public, it can only be used through newEphemeralGPGSigningMechanism.
// The context can be used to cancel the operation; if cancelled, the reader will return an error
// which may allow the GPGME operation to abort (though there's no guarantee the C library will
// respect this immediately).
func importKeysFromBytes(ctx context.Context, gpgmeCtx *gpgme.Context, blob []byte) ([]string, error) {
	// Create a cancelable reader that checks context on each Read call
	reader := &cancelableReader{
		ctx:    ctx,
		reader: bytes.NewReader(blob),
	}
	inputData, err := gpgme.NewDataReader(reader)
	if err != nil {
		return nil, err
	}
	res, err := gpgmeCtx.Import(inputData)
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
