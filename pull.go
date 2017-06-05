package buildah

import (
	"github.com/Sirupsen/logrus"
	cp "github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
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
			return errors.Wrapf(err2, "error parsing image name %q", spec)
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
		return errors.Wrapf(err, "error parsing full image name %q", name)
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

	err = cp.Image(policyContext, destRef, srcRef, getCopyOptions(options.ReportWriter))
	return err
}
