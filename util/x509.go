package util

import (
	"errors"
	"fmt"
	"os"
)

func ResolveRootCACertFile() (string, error) {
	if rv, ok := os.LookupEnv("SSL_CERT_FILE"); ok {
		return rv, nil
	}
	for _, potFile := range certFiles {
		if _, err := os.Stat(potFile); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("unexpected error resolving cert file: %w", err)
		}
		return potFile, nil
	}
	return "", errors.New("unable to resolve host cert file. Consider setting SSL_CERT_FILE environment variable")
}
