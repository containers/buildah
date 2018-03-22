package buildah

import (
	"strings"

	cp "github.com/containers/image/copy"
	"github.com/containers/image/docker/reference"
	tarfile "github.com/containers/image/docker/tarfile"
	ociarchive "github.com/containers/image/oci/archive"
	"github.com/containers/image/signature"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/util"
	"github.com/sirupsen/logrus"
)

func localImageNameForReference(store storage.Store, srcRef types.ImageReference, spec string) (string, error) {
	if srcRef == nil {
		return "", errors.Errorf("reference to image is empty")
	}
	split := strings.SplitN(spec, ":", 2)
	file := split[len(split)-1]
	var name string
	switch srcRef.Transport().Name() {
	case util.DockerArchive:
		tarSource, err := tarfile.NewSourceFromFile(file)
		if err != nil {
			return "", err
		}
		manifest, err := tarSource.LoadTarManifest()
		if err != nil {
			return "", errors.Errorf("error retrieving manifest.json: %v", err)
		}
		// to pull the first image stored in the tar file
		if len(manifest) == 0 {
			// use the hex of the digest if no manifest is found
			name, err = getImageDigest(srcRef, nil)
			if err != nil {
				return "", err
			}
		} else {
			if len(manifest[0].RepoTags) > 0 {
				name = manifest[0].RepoTags[0]
			} else {
				// If the input image has no repotags, we need to feed it a dest anyways
				name, err = getImageDigest(srcRef, nil)
				if err != nil {
					return "", err
				}
			}
		}
	case util.OCIArchive:
		// retrieve the manifest from index.json to access the image name
		manifest, err := ociarchive.LoadManifestDescriptor(srcRef)
		if err != nil {
			return "", errors.Wrapf(err, "error loading manifest for %q", srcRef)
		}
		if manifest.Annotations == nil || manifest.Annotations["org.opencontainers.image.ref.name"] == "" {
			return "", errors.Errorf("error, archive doesn't have a name annotation. Cannot store image with no name")
		}
		name = manifest.Annotations["org.opencontainers.image.ref.name"]
	case util.DirTransport:
		// supports pull from a directory
		name = split[1]
		// remove leading "/"
		if name[:1] == "/" {
			name = name[1:]
		}
	default:
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

		if named, ok := ref.(reference.Named); ok {
			name = named.Name()
			if namedTagged, ok := ref.(reference.NamedTagged); ok {
				name = name + ":" + namedTagged.Tag()
			}
			if canonical, ok := ref.(reference.Canonical); ok {
				name = name + "@" + canonical.Digest().String()
			}
		}
	}

	if _, err := is.Transport.ParseStoreReference(store, name); err != nil {
		return "", errors.Wrapf(err, "error parsing computed local image name %q", name)
	}
	return name, nil
}

func pullImage(store storage.Store, imageName string, options BuilderOptions, sc *types.SystemContext) (types.ImageReference, error) {
	spec := imageName
	srcRef, err := alltransports.ParseImageName(spec)
	if err != nil {
		if options.Transport == "" {
			return nil, errors.Wrapf(err, "error parsing image name %q", spec)
		}
		transport := options.Transport
		if transport != DefaultTransport {
			transport = transport + ":"
		}
		spec = transport + spec
		srcRef2, err2 := alltransports.ParseImageName(spec)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "error parsing image name %q", spec)
		}
		srcRef = srcRef2
	}

	destName, err := localImageNameForReference(store, srcRef, spec)
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

	img, err := srcRef.NewImageSource(sc)
	if err != nil {
		return nil, errors.Wrapf(err, "error initializing %q as an image source", spec)
	}
	img.Close()

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
			logrus.Debugf("error destroying signature policy context: %v", err2)
		}
	}()

	logrus.Debugf("copying %q to %q", spec, destName)

	err = cp.Image(policyContext, destRef, srcRef, getCopyOptions(options.ReportWriter, options.SystemContext, nil, ""))
	if err == nil {
		return destRef, nil
	}
	return nil, err
}

// getImageDigest creates an image object and uses the hex value of the digest as the image ID
// for parsing the store reference
func getImageDigest(src types.ImageReference, ctx *types.SystemContext) (string, error) {
	newImg, err := src.NewImage(ctx)
	if err != nil {
		return "", err
	}
	defer newImg.Close()

	digest := newImg.ConfigInfo().Digest
	if err = digest.Validate(); err != nil {
		return "", errors.Wrapf(err, "error getting config info")
	}
	return "@" + digest.Hex(), nil
}
