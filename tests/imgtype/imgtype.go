package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/docker"
	"github.com/containers/buildah/util"
	"github.com/containers/image/manifest"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

func main() {
	if buildah.InitReexec() {
		return
	}

	expectedManifestType := ""
	expectedConfigType := ""

	storeOptions, _ := storage.DefaultStoreOptions(false, 0)
	debug := flag.Bool("debug", false, "turn on debug logging")
	root := flag.String("root", storeOptions.GraphRoot, "storage root directory")
	runroot := flag.String("runroot", storeOptions.RunRoot, "storage runtime directory")
	driver := flag.String("storage-driver", storeOptions.GraphDriverName, "storage driver")
	opts := flag.String("storage-opts", "", "storage option list (comma separated)")
	policy := flag.String("signature-policy", "", "signature policy file")
	mtype := flag.String("expected-manifest-type", buildah.OCIv1ImageManifest, "expected manifest type")
	showm := flag.Bool("show-manifest", false, "output the manifest JSON")
	rebuildm := flag.Bool("rebuild-manifest", false, "rebuild the manifest JSON")
	showc := flag.Bool("show-config", false, "output the configuration JSON")
	rebuildc := flag.Bool("rebuild-config", false, "rebuild the configuration JSON")
	flag.Parse()
	logrus.SetLevel(logrus.ErrorLevel)
	if debug != nil && *debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
	switch *mtype {
	case buildah.OCIv1ImageManifest:
		expectedManifestType = *mtype
		expectedConfigType = v1.MediaTypeImageConfig
	case buildah.Dockerv2ImageManifest:
		expectedManifestType = *mtype
		expectedConfigType = manifest.DockerV2Schema2ConfigMediaType
	case "*":
		expectedManifestType = ""
		expectedConfigType = ""
	default:
		logrus.Errorf("unknown -expected-manifest-type value, expected either %q or %q or %q",
			buildah.OCIv1ImageManifest, buildah.Dockerv2ImageManifest, "*")
		return
	}
	if root != nil {
		storeOptions.GraphRoot = *root
	}
	if runroot != nil {
		storeOptions.RunRoot = *runroot
	}
	if driver != nil {
		storeOptions.GraphDriverName = *driver
	}
	if opts != nil && *opts != "" {
		storeOptions.GraphDriverOptions = strings.Split(*opts, ",")
	}
	systemContext := &types.SystemContext{
		SignaturePolicyPath: *policy,
	}
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}
	store, err := storage.GetStore(storeOptions)
	if err != nil {
		logrus.Errorf("error opening storage: %v", err)
		return
	}
	is.Transport.SetStore(store)

	errors := false
	defer func() {
		store.Shutdown(false) // nolint:errcheck
		if errors {
			os.Exit(1)
		}
	}()
	for _, image := range args {
		var ref types.ImageReference
		oImage := v1.Image{}
		dImage := docker.V2Image{}
		oManifest := v1.Manifest{}
		dManifest := docker.V2S2Manifest{}
		manifestType := ""
		configType := ""

		ref, _, err := util.FindImage(store, "", systemContext, image)
		if err != nil {
			ref2, err2 := alltransports.ParseImageName(image)
			if err2 != nil {
				logrus.Errorf("error parsing reference %q to an image: %v", image, err)
				errors = true
				continue
			}
			ref = ref2
		}

		ctx := context.Background()
		img, err := ref.NewImage(ctx, systemContext)
		if err != nil {
			logrus.Errorf("error opening image %q: %v", image, err)
			errors = true
			continue
		}
		defer img.Close()

		config, err := img.ConfigBlob(ctx)
		if err != nil {
			logrus.Errorf("error reading configuration from %q: %v", image, err)
			errors = true
			continue
		}

		manifest, manifestType, err := img.Manifest(ctx)
		if err != nil {
			logrus.Errorf("error reading manifest from %q: %v", image, err)
			errors = true
			continue
		}

		switch expectedManifestType {
		case buildah.OCIv1ImageManifest:
			err = json.Unmarshal(manifest, &oManifest)
			if err != nil {
				logrus.Errorf("error parsing manifest from %q: %v", image, err)
				errors = true
				continue
			}
			err = json.Unmarshal(config, &oImage)
			if err != nil {
				logrus.Errorf("error parsing config from %q: %v", image, err)
				errors = true
				continue
			}
			manifestType = v1.MediaTypeImageManifest
			configType = oManifest.Config.MediaType
		case buildah.Dockerv2ImageManifest:
			err = json.Unmarshal(manifest, &dManifest)
			if err != nil {
				logrus.Errorf("error parsing manifest from %q: %v", image, err)
				errors = true
				continue
			}
			err = json.Unmarshal(config, &dImage)
			if err != nil {
				logrus.Errorf("error parsing config from %q: %v", image, err)
				errors = true
				continue
			}
			manifestType = dManifest.MediaType
			configType = dManifest.Config.MediaType
		}
		if expectedManifestType != "" && manifestType != expectedManifestType {
			logrus.Errorf("expected manifest type %q in %q, got %q", expectedManifestType, image, manifestType)
			errors = true
			continue
		}
		switch manifestType {
		case buildah.OCIv1ImageManifest:
			if rebuildm != nil && *rebuildm {
				err = json.Unmarshal(manifest, &oManifest)
				if err != nil {
					logrus.Errorf("error parsing manifest from %q: %v", image, err)
					errors = true
					continue
				}
				manifest, err = json.Marshal(oManifest)
				if err != nil {
					logrus.Errorf("error rebuilding manifest from %q: %v", image, err)
					errors = true
					continue
				}
			}
			if rebuildc != nil && *rebuildc {
				err = json.Unmarshal(config, &oImage)
				if err != nil {
					logrus.Errorf("error parsing config from %q: %v", image, err)
					errors = true
					continue
				}
				config, err = json.Marshal(oImage)
				if err != nil {
					logrus.Errorf("error rebuilding config from %q: %v", image, err)
					errors = true
					continue
				}
			}
		case buildah.Dockerv2ImageManifest:
			if rebuildm != nil && *rebuildm {
				err = json.Unmarshal(manifest, &dManifest)
				if err != nil {
					logrus.Errorf("error parsing manifest from %q: %v", image, err)
					errors = true
					continue
				}
				manifest, err = json.Marshal(dManifest)
				if err != nil {
					logrus.Errorf("error rebuilding manifest from %q: %v", image, err)
					errors = true
					continue
				}
			}
			if rebuildc != nil && *rebuildc {
				err = json.Unmarshal(config, &dImage)
				if err != nil {
					logrus.Errorf("error parsing config from %q: %v", image, err)
					errors = true
					continue
				}
				config, err = json.Marshal(dImage)
				if err != nil {
					logrus.Errorf("error rebuilding config from %q: %v", image, err)
					errors = true
					continue
				}
			}
		}
		if expectedConfigType != "" && configType != expectedConfigType {
			logrus.Errorf("expected config type %q in %q, got %q", expectedConfigType, image, configType)
			errors = true
			continue
		}
		if showm != nil && *showm {
			fmt.Println(string(manifest))
		}
		if showc != nil && *showc {
			fmt.Println(string(config))
		}
	}
}
