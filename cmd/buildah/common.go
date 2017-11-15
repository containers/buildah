package main

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"time"

	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	digest "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var needToShutdownStore = false

func getStore(c *cli.Context) (storage.Store, error) {
	options := storage.DefaultStoreOptions
	if c.GlobalIsSet("root") || c.GlobalIsSet("runroot") {
		options.GraphRoot = c.GlobalString("root")
		options.RunRoot = c.GlobalString("runroot")
	}
	if c.GlobalIsSet("storage-driver") {
		options.GraphDriverName = c.GlobalString("storage-driver")
	}
	if c.GlobalIsSet("storage-opt") {
		opts := c.GlobalStringSlice("storage-opt")
		if len(opts) > 0 {
			options.GraphDriverOptions = opts
		}
	}
	store, err := storage.GetStore(options)
	if store != nil {
		is.Transport.SetStore(store)
	}
	needToShutdownStore = true
	return store, err
}

func openBuilder(store storage.Store, name string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(err) {
			options := buildah.ImportOptions{
				Container: name,
			}
			builder, err = buildah.ImportBuilder(store, options)
		}
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error reading build container")
	}
	if builder == nil {
		return nil, errors.Errorf("error finding build container")
	}
	return builder, nil
}

func openBuilders(store storage.Store) (builders []*buildah.Builder, err error) {
	return buildah.OpenAllBuilders(store)
}

func openImage(store storage.Store, name string) (builder *buildah.Builder, err error) {
	options := buildah.ImportFromImageOptions{
		Image: name,
	}
	builder, err = buildah.ImportBuilderFromImage(store, options)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading image")
	}
	if builder == nil {
		return nil, errors.Errorf("error mocking up build configuration")
	}
	return builder, nil
}

func getDateAndDigestAndSize(image storage.Image, store storage.Store) (time.Time, string, int64, error) {
	created := time.Time{}
	is.Transport.SetStore(store)
	storeRef, err := is.Transport.ParseStoreReference(store, "@"+image.ID)
	if err != nil {
		return created, "", -1, err
	}
	img, err := storeRef.NewImage(nil)
	if err != nil {
		return created, "", -1, err
	}
	defer img.Close()
	imgSize, sizeErr := img.Size()
	if sizeErr != nil {
		imgSize = -1
	}
	manifest, _, manifestErr := img.Manifest()
	manifestDigest := ""
	if manifestErr == nil && len(manifest) > 0 {
		manifestDigest = digest.Canonical.FromBytes(manifest).String()
	}
	inspectInfo, inspectErr := img.Inspect()
	if inspectErr == nil && inspectInfo != nil {
		created = inspectInfo.Created
	}
	if sizeErr != nil {
		err = sizeErr
	} else if manifestErr != nil {
		err = manifestErr
	} else if inspectErr != nil {
		err = inspectErr
	}
	return created, manifestDigest, imgSize, err
}

// systemContextFromOptions returns a SystemContext populated with values
// per the input parameters provided by the caller for the use in authentication.
func systemContextFromOptions(c *cli.Context) (*types.SystemContext, error) {
	ctx := &types.SystemContext{
		DockerCertPath: c.String("cert-dir"),
	}
	if c.IsSet("tls-verify") {
		ctx.DockerInsecureSkipTLSVerify = !c.BoolT("tls-verify")
	}
	if c.IsSet("creds") {
		var err error
		ctx.DockerAuthConfig, err = getDockerAuth(c.String("creds"))
		if err != nil {
			return nil, err
		}
	}
	if c.IsSet("signature-policy") {
		ctx.SignaturePolicyPath = c.String("signature-policy")
	}
	if c.IsSet("authfile") {
		ctx.AuthFilePath = c.String("authfile")
	}
	return ctx, nil
}

func parseCreds(creds string) (string, string, error) {
	if creds == "" {
		return "", "", errors.Wrapf(syscall.EINVAL, "credentials can't be empty")
	}
	up := strings.SplitN(creds, ":", 2)
	if len(up) == 1 {
		return up[0], "", nil
	}
	if up[0] == "" {
		return "", "", errors.Wrapf(syscall.EINVAL, "username can't be empty")
	}
	return up[0], up[1], nil
}

func getDockerAuth(creds string) (*types.DockerAuthConfig, error) {
	username, password, err := parseCreds(creds)
	if err != nil {
		return nil, err
	}
	return &types.DockerAuthConfig{
		Username: username,
		Password: password,
	}, nil
}

// validateFlags searches for StringFlags or StringSlice flags that never had
// a value set.  This commonly occurs when the CLI mistakenly takes the next
// option and uses it as a value.
func validateFlags(c *cli.Context, flags []cli.Flag) error {
	re, err := regexp.Compile("^-.+")
	if err != nil {
		return errors.Wrap(err, "compiling regex failed")
	}

	for _, flag := range flags {
		switch reflect.TypeOf(flag).String() {
		case "cli.StringSliceFlag":
			{
				f := flag.(cli.StringSliceFlag)
				name := strings.Split(f.Name, ",")
				val := c.StringSlice(name[0])
				for _, v := range val {
					if ok := re.MatchString(v); ok {
						return errors.Errorf("option --%s requires a value", name[0])
					}
				}
			}
		case "cli.StringFlag":
			{
				f := flag.(cli.StringFlag)
				name := strings.Split(f.Name, ",")
				val := c.String(name[0])
				if ok := re.MatchString(val); ok {
					return errors.Errorf("option --%s requires a value", name[0])
				}
			}
		}
	}
	return nil
}
