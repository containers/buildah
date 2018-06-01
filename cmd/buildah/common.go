package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	is "github.com/containers/image/storage"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/idtools"
	digest "github.com/opencontainers/go-digest"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/sirupsen/logrus"
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
		// If any options setup in config, these should be dropped if user overrode the driver
		options.GraphDriverOptions = []string{}
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

func openBuilder(ctx context.Context, store storage.Store, name string) (builder *buildah.Builder, err error) {
	if name != "" {
		builder, err = buildah.OpenBuilder(store, name)
		if os.IsNotExist(err) {
			options := buildah.ImportOptions{
				Container: name,
			}
			builder, err = buildah.ImportBuilder(ctx, store, options)
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

func openImage(ctx context.Context, sc *types.SystemContext, store storage.Store, name string) (builder *buildah.Builder, err error) {
	options := buildah.ImportFromImageOptions{
		Image:         name,
		SystemContext: sc,
	}
	builder, err = buildah.ImportBuilderFromImage(ctx, store, options)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading image")
	}
	if builder == nil {
		return nil, errors.Errorf("error mocking up build configuration")
	}
	return builder, nil
}

func getDateAndDigestAndSize(ctx context.Context, image storage.Image, store storage.Store) (time.Time, string, int64, error) {
	created := time.Time{}
	is.Transport.SetStore(store)
	storeRef, err := is.Transport.ParseStoreReference(store, image.ID)
	if err != nil {
		return created, "", -1, err
	}
	img, err := storeRef.NewImage(ctx, nil)
	if err != nil {
		return created, "", -1, err
	}
	defer img.Close()
	imgSize, sizeErr := img.Size()
	if sizeErr != nil {
		imgSize = -1
	}
	manifest, _, manifestErr := img.Manifest(ctx)
	manifestDigest := ""
	if manifestErr == nil && len(manifest) > 0 {
		manifestDigest = digest.Canonical.FromBytes(manifest).String()
	}
	inspectInfo, inspectErr := img.Inspect(ctx)
	if inspectErr == nil && inspectInfo != nil {
		created = *inspectInfo.Created
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

// getContext returns a context.TODO
func getContext() context.Context {
	return context.TODO()
}

var userFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "user",
		Usage: "`user[:group]` to run the command as",
	},
}

func parseUserOptions(c *cli.Context) string {
	return c.String("user")
}

func parseNamespaceOptions(c *cli.Context) (namespaceOptions buildah.NamespaceOptions, networkPolicy buildah.NetworkConfigurationPolicy, err error) {
	options := make(buildah.NamespaceOptions, 0, 7)
	policy := buildah.NetworkDefault
	for _, what := range []string{string(specs.IPCNamespace), "net", string(specs.PIDNamespace), string(specs.UTSNamespace)} {
		if c.IsSet(what) {
			how := c.String(what)
			switch what {
			case "net", "network":
				what = string(specs.NetworkNamespace)
			}
			switch how {
			case "", "container":
				logrus.Debugf("setting %q namespace to %q", what, "")
				options.AddOrReplace(buildah.NamespaceOption{
					Name: what,
				})
			case "host":
				logrus.Debugf("setting %q namespace to host", what)
				options.AddOrReplace(buildah.NamespaceOption{
					Name: what,
					Host: true,
				})
			default:
				if what == specs.NetworkNamespace {
					if how == "none" {
						options.AddOrReplace(buildah.NamespaceOption{
							Name: what,
						})
						policy = buildah.NetworkDisabled
						logrus.Debugf("setting network to disabled")
						break
					}
					if !filepath.IsAbs(how) {
						options.AddOrReplace(buildah.NamespaceOption{
							Name: what,
							Path: how,
						})
						policy = buildah.NetworkEnabled
						logrus.Debugf("setting network configuration to %q", how)
						break
					}
				}
				if _, err := os.Stat(how); err != nil {
					return nil, buildah.NetworkDefault, errors.Wrapf(err, "error checking for %s namespace at %q", what, how)
				}
				logrus.Debugf("setting %q namespace to %q", what, how)
				options.AddOrReplace(buildah.NamespaceOption{
					Name: what,
					Path: how,
				})
			}
		}
	}
	return options, policy, nil
}

func parseIDMappingOptions(c *cli.Context) (usernsOptions buildah.NamespaceOptions, idmapOptions *buildah.IDMappingOptions, err error) {
	user := c.String("userns-uid-map-user")
	group := c.String("userns-gid-map-group")
	// If only the user or group was specified, use the same value for the
	// other, since we need both in order to initialize the maps using the
	// names.
	if user == "" && group != "" {
		user = group
	}
	if group == "" && user != "" {
		group = user
	}
	// Either start with empty maps or the name-based maps.
	mappings := idtools.NewIDMappingsFromMaps(nil, nil)
	if user != "" && group != "" {
		submappings, err := idtools.NewIDMappings(user, group)
		if err != nil {
			return nil, nil, err
		}
		mappings = submappings
	}
	// We'll parse the UID and GID mapping options the same way.
	buildIDMap := func(basemap []idtools.IDMap, option string) ([]specs.LinuxIDMapping, error) {
		outmap := make([]specs.LinuxIDMapping, 0, len(basemap))
		// Start with the name-based map entries.
		for _, m := range basemap {
			outmap = append(outmap, specs.LinuxIDMapping{
				ContainerID: uint32(m.ContainerID),
				HostID:      uint32(m.HostID),
				Size:        uint32(m.Size),
			})
		}
		// Parse the flag's value as one or more triples (if it's even
		// been set), and append them.
		idmap, err := parseIDMap(c.StringSlice(option))
		if err != nil {
			return nil, err
		}
		for _, m := range idmap {
			outmap = append(outmap, specs.LinuxIDMapping{
				ContainerID: m[0],
				HostID:      m[1],
				Size:        m[2],
			})
		}
		return outmap, nil
	}
	uidmap, err := buildIDMap(mappings.UIDs(), "userns-uid-map")
	if err != nil {
		return nil, nil, err
	}
	gidmap, err := buildIDMap(mappings.GIDs(), "userns-gid-map")
	if err != nil {
		return nil, nil, err
	}
	// If we only have one map or the other populated at this point, then
	// use the same mapping for both, since we know that no user or group
	// name was specified, but a specific mapping was for one or the other.
	if len(uidmap) == 0 && len(gidmap) != 0 {
		uidmap = gidmap
	}
	if len(gidmap) == 0 && len(uidmap) != 0 {
		gidmap = uidmap
	}
	// By default, having mappings configured means we use a user
	// namespace.  Otherwise, we don't.
	usernsOption := buildah.NamespaceOption{
		Name: string(specs.UserNamespace),
		Host: len(uidmap) == 0 && len(gidmap) == 0,
	}
	// If the user specifically requested that we either use or don't use
	// user namespaces, override that default.
	if c.IsSet("userns") {
		how := c.String("userns")
		switch how {
		case "", "container":
			usernsOption.Host = false
		case "host":
			usernsOption.Host = true
		default:
			if _, err := os.Stat(how); err != nil {
				return nil, nil, errors.Wrapf(err, "error checking for %s namespace at %q", string(specs.UserNamespace), how)
			}
			logrus.Debugf("setting %q namespace to %q", string(specs.UserNamespace), how)
			usernsOption.Path = how
		}
	}
	usernsOptions = buildah.NamespaceOptions{usernsOption}
	if !c.IsSet("net") {
		usernsOptions = append(usernsOptions, buildah.NamespaceOption{
			Name: string(specs.NetworkNamespace),
			Host: usernsOption.Host,
		})
	}
	// If the user requested that we use the host namespace, but also that
	// we use mappings, that's not going to work.
	if (len(uidmap) != 0 || len(gidmap) != 0) && usernsOption.Host {
		return nil, nil, errors.Errorf("can not specify ID mappings while using host's user namespace")
	}
	return usernsOptions, &buildah.IDMappingOptions{
		HostUIDMapping: usernsOption.Host,
		HostGIDMapping: usernsOption.Host,
		UIDMap:         uidmap,
		GIDMap:         gidmap,
	}, nil
}

func parseIDMap(spec []string) (m [][3]uint32, err error) {
	for _, s := range spec {
		args := strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsDigit(r) })
		if len(args)%3 != 0 {
			return nil, fmt.Errorf("mapping %q is not in the form containerid:hostid:size[,...]", s)
		}
		for len(args) >= 3 {
			cid, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("error parsing container ID %q from mapping %q as a number: %v", args[0], s, err)
			}
			hostid, err := strconv.ParseUint(args[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("error parsing host ID %q from mapping %q as a number: %v", args[1], s, err)
			}
			size, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("error parsing %q from mapping %q as a number: %v", args[2], s, err)
			}
			m = append(m, [3]uint32{uint32(cid), uint32(hostid), uint32(size)})
			args = args[3:]
		}
	}
	return m, nil
}

func defaultFormat() string {
	format := os.Getenv("BUILDAH_FORMAT")
	if format != "" {
		return format
	}
	return "oci"
}
