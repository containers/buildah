package buildah

import (
	"strings"

	cp "github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func localImageNameForReference(store storage.Store, srcRef types.ImageReference) (string, error) {
	if srcRef == nil {
		return "", errors.Errorf("reference to image is empty")
	}
	ref := srcRef.DockerReference()
	if ref == nil {
		name := srcRef.StringWithinTransport()
		_, err := is.Transport.ParseStoreReference(store, name)
		if err == nil {
			return name, nil
		}
		if strings.LastIndex(name, "/") != -1 {
			name = name[strings.LastIndex(name, "/")+1:]
			_, err = is.Transport.ParseStoreReference(store, name)
			if err == nil {
				return name, nil
			}
		}
		return "", errors.Errorf("reference to image %q is not a named reference", transports.ImageName(srcRef))
	}

	name := ""
	if named, ok := ref.(reference.Named); ok {
		name = named.Name()
		if namedTagged, ok := ref.(reference.NamedTagged); ok {
			name = name + ":" + namedTagged.Tag()
		}
		if canonical, ok := ref.(reference.Canonical); ok {
			name = name + "@" + canonical.Digest().String()
		}
	}

	if _, err := is.Transport.ParseStoreReference(store, name); err != nil {
		return "", errors.Wrapf(err, "error parsing computed local image name %q", name)
	}
	return name, nil
}

func pullImage(store storage.Store, options BuilderOptions, sc *types.SystemContext) (types.ImageReference, error) {
	name := options.FromImage

	spec := name
	if options.Registry != "" {
		spec = options.Registry + spec
	}
	spec2 := spec
	if options.Transport != "" {
		spec2 = options.Transport + spec
	}

	srcRef, err := alltransports.ParseImageName(name)
	if err != nil {
		srcRef2, err2 := alltransports.ParseImageName(spec)
		if err2 != nil {
			srcRef3, err3 := alltransports.ParseImageName(spec2)
			if err3 != nil {
				return nil, errors.Wrapf(err3, "error parsing image name %q", spec2)
			}
			srcRef2 = srcRef3
		}
		srcRef = srcRef2
	}

	destName, err := localImageNameForReference(store, srcRef)
	if err != nil {
		return nil, errors.Wrapf(err, "error computing local image name for %q", transports.ImageName(srcRef))
	}
	if destName == "" {
		return nil, errors.Errorf("error computing local image name for %q", transports.ImageName(srcRef))
	}

	destRef, err := is.Transport.ParseStoreReference(store, destName)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing image name %q", destName)
	}

	policy, err := signature.DefaultPolicy(sc)
	if err != nil {
		return nil, errors.Wrapf(err, "error obtaining default signature policy")
	}

	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating new signature policy context")
	}

	defer func() {
		if err2 := policyContext.Destroy(); err2 != nil {
			logrus.Debugf("error destroying signature polcy context: %v", err2)
		}
	}()

	logrus.Debugf("copying %q to %q", spec, name)

	err = cp.Image(policyContext, destRef, srcRef, getCopyOptions(options.ReportWriter, options.SystemContext, nil))
	return destRef, err
}
