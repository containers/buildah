package utils

import (
	"os"

	"github.com/containers/image/pkg/sysregistries"
	"github.com/containers/image/signature"
	"github.com/containers/libpod/pkg/rootless"
	"github.com/containers/storage"
	"github.com/pkg/errors"
)

// CheckConfigFiles log messages if one or more files to set up buildah do not exist
func CheckConfigFiles() error {
	registriesConfPath := sysregistries.RegistriesConfPath(nil)
	_, err := os.Stat(registriesConfPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf("required file %q not found.", registriesConfPath)
		}
		return err
	}

	if !rootless.IsRootless() {
		storageDefaultPath := storage.DefaultConfigFile
		_, err = os.Stat(storageDefaultPath)
		if err != nil {
			if os.IsNotExist(err) {
				return errors.Errorf("required file %q not found.", storageDefaultPath)
			}
			return err
		}
	}

	if _, err = signature.DefaultPolicy(nil); err != nil {
		return err
	}
	return nil
}
