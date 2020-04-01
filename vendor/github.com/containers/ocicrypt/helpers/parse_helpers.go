package helpers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/containerd/containerd/platforms"
	"github.com/containers/ocicrypt"
	encconfig "github.com/containers/ocicrypt/config"
	encutils "github.com/containers/ocicrypt/utils"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// processRecipientKeys sorts the array of recipients by type. Recipients may be either
// x509 certificates, public keys, or PGP public keys identified by email address or name
func processRecipientKeys(recipients []string) ([][]byte, [][]byte, [][]byte, error) {
	var (
		gpgRecipients [][]byte
		pubkeys       [][]byte
		x509s         [][]byte
	)
	for _, recipient := range recipients {

		idx := strings.Index(recipient, ":")
		if idx < 0 {
			return nil, nil, nil, errors.New("Invalid recipient format")
		}

		protocol := recipient[:idx]
		value := recipient[idx+1:]

		switch protocol {
		case "pgp":
			gpgRecipients = append(gpgRecipients, []byte(value))

		case "jwe":
			tmp, err := ioutil.ReadFile(value)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "Unable to read file")
			}
			if !encutils.IsPublicKey(tmp) {
				return nil, nil, nil, errors.New("File provided is not a public key")
			}
			pubkeys = append(pubkeys, tmp)

		case "pkcs7":
			tmp, err := ioutil.ReadFile(value)
			if err != nil {
				return nil, nil, nil, errors.Wrap(err, "Unable to read file")
			}
			if !encutils.IsCertificate(tmp) {
				return nil, nil, nil, errors.New("File provided is not an x509 cert")
			}
			x509s = append(x509s, tmp)

		default:
			return nil, nil, nil, errors.New("Provided protocol not recognized")
		}
	}
	return gpgRecipients, pubkeys, x509s, nil
}

// processPwdString process a password that may be in any of the following formats:
// - file=<passwordfile>
// - pass=<password>
// - fd=<filedescriptor>
// - <password>
func processPwdString(pwdString string) ([]byte, error) {
	if strings.HasPrefix(pwdString, "file=") {
		return ioutil.ReadFile(pwdString[5:])
	} else if strings.HasPrefix(pwdString, "pass=") {
		return []byte(pwdString[5:]), nil
	} else if strings.HasPrefix(pwdString, "fd=") {
		fdStr := pwdString[3:]
		fd, err := strconv.Atoi(fdStr)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse file descriptor %s", fdStr)
		}
		f := os.NewFile(uintptr(fd), "pwdfile")
		if f == nil {
			return nil, fmt.Errorf("%s is not a valid file descriptor", fdStr)
		}
		defer f.Close()
		pwd := make([]byte, 64)
		n, err := f.Read(pwd)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read from file descriptor")
		}
		return pwd[:n], nil
	}
	return []byte(pwdString), nil
}

// processPrivateKeyFiles sorts the different types of private key files; private key files may either be
// private keys or GPG private key ring files. The private key files may include the password for the
// private key and take any of the following forms:
// - <filename>
// - <filename>:file=<passwordfile>
// - <filename>:pass=<password>
// - <filename>:fd=<filedescriptor>
// - <filename>:<password>
func processPrivateKeyFiles(keyFilesAndPwds []string) ([][]byte, [][]byte, [][]byte, [][]byte, error) {
	var (
		gpgSecretKeyRingFiles [][]byte
		gpgSecretKeyPasswords [][]byte
		privkeys              [][]byte
		privkeysPasswords     [][]byte
		err                   error
	)
	// keys needed for decryption in case of adding a recipient
	for _, keyfileAndPwd := range keyFilesAndPwds {
		var password []byte

		parts := strings.Split(keyfileAndPwd, ":")
		if len(parts) == 2 {
			password, err = processPwdString(parts[1])
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}

		keyfile := parts[0]
		tmp, err := ioutil.ReadFile(keyfile)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		isPrivKey, err := encutils.IsPrivateKey(tmp, password)
		if encutils.IsPasswordError(err) {
			return nil, nil, nil, nil, err
		}
		if isPrivKey {
			privkeys = append(privkeys, tmp)
			privkeysPasswords = append(privkeysPasswords, password)
		} else if encutils.IsGPGPrivateKeyRing(tmp) {
			gpgSecretKeyRingFiles = append(gpgSecretKeyRingFiles, tmp)
			gpgSecretKeyPasswords = append(gpgSecretKeyPasswords, password)
		} else {
			return nil, nil, nil, nil, fmt.Errorf("unidentified private key in file %s (password=%s)", keyfile, string(password))
		}
	}
	return gpgSecretKeyRingFiles, gpgSecretKeyPasswords, privkeys, privkeysPasswords, nil
}

func createGPGClient(context context.Context) (ocicrypt.GPGClient, error) {
	return ocicrypt.NewGPGClient(context.Value("gpg-version").(string), context.Value("gpg-homedir").(string))
}

func getGPGPrivateKeys(context context.Context, gpgSecretKeyRingFiles [][]byte, descs []ocispec.Descriptor, mustFindKey bool) (gpgPrivKeys [][]byte, gpgPrivKeysPwds [][]byte, err error) {
	gpgClient, err := createGPGClient(context)
	if err != nil {
		return nil, nil, err
	}

	var gpgVault ocicrypt.GPGVault
	if len(gpgSecretKeyRingFiles) > 0 {
		gpgVault = ocicrypt.NewGPGVault()
		err = gpgVault.AddSecretKeyRingDataArray(gpgSecretKeyRingFiles)
		if err != nil {
			return nil, nil, err
		}
	}
	return ocicrypt.GPGGetPrivateKey(descs, gpgClient, gpgVault, mustFindKey)
}

// CreateDecryptCryptoConfig creates the CryptoConfig object that contains the necessary
// information to perform decryption from command line options and possibly
// LayerInfos describing the image and helping us to query for the PGP decryption keys
func CreateDecryptCryptoConfig(keys []string, decRecipients []string) (encconfig.CryptoConfig, error) {
	ccs := []encconfig.CryptoConfig{}

	// x509 cert is needed for PKCS7 decryption
	_, _, x509s, err := processRecipientKeys(decRecipients)
	if err != nil {
		return encconfig.CryptoConfig{}, err
	}

	gpgSecretKeyRingFiles, gpgSecretKeyPasswords, privKeys, privKeysPasswords, err := processPrivateKeyFiles(keys)
	if err != nil {
		return encconfig.CryptoConfig{}, err
	}

	if len(gpgSecretKeyRingFiles) > 0 {
		gpgCc, err := encconfig.DecryptWithGpgPrivKeys(gpgSecretKeyRingFiles, gpgSecretKeyPasswords)
		if err != nil {
			return encconfig.CryptoConfig{}, err
		}
		ccs = append(ccs, gpgCc)
	}

	/* TODO: Add in GPG client query for secret keys in the future.
	_, err = createGPGClient(context)
	gpgInstalled := err == nil
	if gpgInstalled {
		if len(gpgSecretKeyRingFiles) == 0 && len(privKeys) == 0 && descs != nil {
			// Get pgp private keys from keyring only if no private key was passed
			gpgPrivKeys, gpgPrivKeyPasswords, err := getGPGPrivateKeys(context, gpgSecretKeyRingFiles, descs, true)
			if err != nil {
				return encconfig.CryptoConfig{}, err
			}

			gpgCc, err := encconfig.DecryptWithGpgPrivKeys(gpgPrivKeys, gpgPrivKeyPasswords)
			if err != nil {
				return encconfig.CryptoConfig{}, err
			}
			ccs = append(ccs, gpgCc)

		} else if len(gpgSecretKeyRingFiles) > 0 {
			gpgCc, err := encconfig.DecryptWithGpgPrivKeys(gpgSecretKeyRingFiles, gpgSecretKeyPasswords)
			if err != nil {
				return encconfig.CryptoConfig{}, err
			}
			ccs = append(ccs, gpgCc)

		}
	}
	*/

	x509sCc, err := encconfig.DecryptWithX509s(x509s)
	if err != nil {
		return encconfig.CryptoConfig{}, err
	}
	ccs = append(ccs, x509sCc)

	privKeysCc, err := encconfig.DecryptWithPrivKeys(privKeys, privKeysPasswords)
	if err != nil {
		return encconfig.CryptoConfig{}, err
	}
	ccs = append(ccs, privKeysCc)

	return encconfig.CombineCryptoConfigs(ccs), nil
}

// parsePlatformArray parses an array of specifiers and converts them into an array of specs.Platform
func parsePlatformArray(specifiers []string) ([]ocispec.Platform, error) {
	var speclist []ocispec.Platform

	for _, specifier := range specifiers {
		spec, err := platforms.Parse(specifier)
		if err != nil {
			return []ocispec.Platform{}, err
		}
		speclist = append(speclist, spec)
	}
	return speclist, nil
}

// CreateCryptoConfig from the list of recipient strings and list of key paths of private keys
func CreateCryptoConfig(recipients []string, keys []string) (encconfig.CryptoConfig, error) {
	var decryptCc *encconfig.CryptoConfig
	ccs := []encconfig.CryptoConfig{}
	if len(keys) > 0 {
		dcc, err := CreateDecryptCryptoConfig(keys, []string{})
		if err != nil {
			return encconfig.CryptoConfig{}, err
		}
		decryptCc = &dcc
		ccs = append(ccs, dcc)
	}

	if len(recipients) > 0 {
		gpgRecipients, pubKeys, x509s, err := processRecipientKeys(recipients)
		if err != nil {
			return encconfig.CryptoConfig{}, err
		}
		encryptCcs := []encconfig.CryptoConfig{}

		// Create GPG client with guessed GPG version and default homedir
		gpgClient, err := ocicrypt.NewGPGClient("", "")
		gpgInstalled := err == nil
		if len(gpgRecipients) > 0 && gpgInstalled {
			gpgPubRingFile, err := gpgClient.ReadGPGPubRingFile()
			if err != nil {
				return encconfig.CryptoConfig{}, err
			}

			gpgCc, err := encconfig.EncryptWithGpg(gpgRecipients, gpgPubRingFile)
			if err != nil {
				return encconfig.CryptoConfig{}, err
			}
			encryptCcs = append(encryptCcs, gpgCc)
		}

		// Create Encryption Crypto Config
		pkcs7Cc, err := encconfig.EncryptWithPkcs7(x509s)
		if err != nil {
			return encconfig.CryptoConfig{}, err
		}
		encryptCcs = append(encryptCcs, pkcs7Cc)

		jweCc, err := encconfig.EncryptWithJwe(pubKeys)
		if err != nil {
			return encconfig.CryptoConfig{}, err
		}
		encryptCcs = append(encryptCcs, jweCc)
		ecc := encconfig.CombineCryptoConfigs(encryptCcs)
		if decryptCc != nil {
			ecc.EncryptConfig.AttachDecryptConfig(decryptCc.DecryptConfig)
		}
		ccs = append(ccs, ecc)
	}

	if len(ccs) > 0 {
		return encconfig.CombineCryptoConfigs(ccs), nil
	} else {
		return encconfig.CryptoConfig{}, nil
	}
}
