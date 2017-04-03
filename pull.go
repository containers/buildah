package buildah

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage/storage"
)

func pullImage(store storage.Store, options BuilderOptions, sc *types.SystemContext) error {
	name := options.FromImage

	spec := name
	if options.Registry != "" {
		spec = options.Registry + spec
	}

	srcRef, err := alltransports.ParseImageName(name)
	if err != nil {
		srcRef2, err2 := alltransports.ParseImageName(spec)
		if err2 != nil {
			return fmt.Errorf("error parsing image name %q: %v", spec, err2)
		}
		srcRef = srcRef2
	}

	if ref := srcRef.DockerReference(); ref != nil {
		name = srcRef.DockerReference().Name()
		if tagged, ok := srcRef.DockerReference().(reference.NamedTagged); ok {
			name = name + ":" + tagged.Tag()
		}
	}

	destRef, err := is.Transport.ParseStoreReference(store, name)
	if err != nil {
		return fmt.Errorf("error parsing full image name %q: %v", name, err)
	}

	policy, err := signature.DefaultPolicy(sc)
	if err != nil {
		return err
	}

	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return err
	}

	logrus.Debugf("copying %q to %q", spec, name)

	err = copy.Image(policyContext, destRef, srcRef, getCopyOptions())
	if err != nil {
		return err
	}

	// Go find the image, and attach the requested name to it, so that we
	// can more easily find it later, even if the destination reference
	// looks different.
	destImage, err := is.Transport.GetStoreImage(store, destRef)
	if err != nil {
		return err
	}

	names := append(destImage.Names, options.FromImage, name)
	err = store.SetNames(destImage.ID, names)
	return err
}
