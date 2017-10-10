package imagebuildah

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/stringid"
	"github.com/docker/docker/builder/dockerfile/parser"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/openshift/imagebuilder"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/sirupsen/logrus"
)

const (
	PullIfMissing       = buildah.PullIfMissing
	PullAlways          = buildah.PullAlways
	PullNever           = buildah.PullNever
	DefaultRuntime      = buildah.DefaultRuntime
	OCIv1ImageFormat    = buildah.OCIv1ImageManifest
	Dockerv2ImageFormat = buildah.Dockerv2ImageManifest

	Gzip         = archive.Gzip
	Bzip2        = archive.Bzip2
	Xz           = archive.Xz
	Uncompressed = archive.Uncompressed
)

// Mount is a mountpoint for the build container.
type Mount specs.Mount

// BuildOptions can be used to alter how an image is built.
type BuildOptions struct {
	// ContextDirectory is the default source location for COPY and ADD
	// commands.
	ContextDirectory string
	// PullPolicy controls whether or not we pull images.  It should be one
	// of PullIfMissing, PullAlways, or PullNever.
	PullPolicy int
	// Registry is a value which is prepended to the image's name, if it
	// needs to be pulled and the image name alone can not be resolved to a
	// reference to a source image.  No separator is implicitly added.
	Registry string
	// Transport is a value which is prepended to the image's name, if it
	// needs to be pulled and the image name alone, or the image name and
	// the registry together, can not be resolved to a reference to a
	// source image.  No separator is implicitly added.
	Transport string
	// IgnoreUnrecognizedInstructions tells us to just log instructions we
	// don't recognize, and try to keep going.
	IgnoreUnrecognizedInstructions bool
	// Quiet tells us whether or not to announce steps as we go through them.
	Quiet bool
	// Runtime is the name of the command to run for RUN instructions.  It
	// should accept the same arguments and flags that runc does.
	Runtime string
	// RuntimeArgs adds global arguments for the runtime.
	RuntimeArgs []string
	// TransientMounts is a list of mounts that won't be kept in the image.
	TransientMounts []Mount
	// Compression specifies the type of compression which is applied to
	// layer blobs.  The default is to not use compression, but
	// archive.Gzip is recommended.
	Compression archive.Compression
	// Arguments which can be interpolated into Dockerfiles
	Args map[string]string
	// Name of the image to write to.
	Output string
	// Additional tags to add to the image that we write, if we know of a
	// way to add them.
	AdditionalTags []string
	// Log is a callback that will print a progress message.  If no value
	// is supplied, the message will be sent to Err (or os.Stderr, if Err
	// is nil) by default.
	Log func(format string, args ...interface{})
	// Out is a place where non-error log messages are sent.
	Out io.Writer
	// Err is a place where error log messages should be sent.
	Err io.Writer
	// SignaturePolicyPath specifies an override location for the signature
	// policy which should be used for verifying the new image as it is
	// being written.  Except in specific circumstances, no value should be
	// specified, indicating that the shared, system-wide default policy
	// should be used.
	SignaturePolicyPath string
	// ReportWriter is an io.Writer which will be used to report the
	// progress of the (possible) pulling of the source image and the
	// writing of the new image.
	ReportWriter io.Writer
	// OutputFormat is the format of the output image's manifest and
	// configuration data.
	// Accepted values are OCIv1ImageFormat and Dockerv2ImageFormat.
	OutputFormat string
}

// Executor is a buildah-based implementation of the imagebuilder.Executor
// interface.
type Executor struct {
	store                          storage.Store
	contextDir                     string
	builder                        *buildah.Builder
	pullPolicy                     int
	registry                       string
	transport                      string
	ignoreUnrecognizedInstructions bool
	quiet                          bool
	runtime                        string
	runtimeArgs                    []string
	transientMounts                []Mount
	compression                    archive.Compression
	output                         string
	outputFormat                   string
	additionalTags                 []string
	log                            func(format string, args ...interface{})
	out                            io.Writer
	err                            io.Writer
	signaturePolicyPath            string
	systemContext                  *types.SystemContext
	mountPoint                     string
	preserved                      int
	volumes                        imagebuilder.VolumeSet
	volumeCache                    map[string]string
	volumeCacheInfo                map[string]os.FileInfo
	reportWriter                   io.Writer
}

func makeSystemContext(signaturePolicyPath string) *types.SystemContext {
	sc := &types.SystemContext{}
	if signaturePolicyPath != "" {
		sc.SignaturePolicyPath = signaturePolicyPath
	}
	return sc
}

// Preserve informs the executor that from this point on, it needs to ensure
// that only COPY and ADD instructions can modify the contents of this
// directory or anything below it.
// The Executor handles this by caching the contents of directories which have
// been marked this way before executing a RUN instruction, invalidating that
// cache when an ADD or COPY instruction sets any location under the directory
// as the destination, and using the cache to reset the contents of the
// directory tree after processing each RUN instruction.
// It would be simpler if we could just mark the directory as a read-only bind
// mount of itself during Run(), but the directory is expected to be remain
// writeable, even if any changes within it are ultimately discarded.
func (b *Executor) Preserve(path string) error {
	logrus.Debugf("PRESERVE %q", path)
	if b.volumes.Covers(path) {
		// This path is already a subdirectory of a volume path that
		// we're already preserving, so there's nothing new to be done
		// except ensure that it exists.
		archivedPath := filepath.Join(b.mountPoint, path)
		if err := os.MkdirAll(archivedPath, 0755); err != nil {
			return errors.Wrapf(err, "error ensuring volume path %q exists", archivedPath)
		}
		if err := b.volumeCacheInvalidate(path); err != nil {
			return errors.Wrapf(err, "error ensuring volume path %q is preserved", archivedPath)
		}
		return nil
	}
	// Figure out where the cache for this volume would be stored.
	b.preserved++
	cacheDir, err := b.store.ContainerDirectory(b.builder.ContainerID)
	if err != nil {
		return errors.Errorf("unable to locate temporary directory for container")
	}
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("volume%d.tar", b.preserved))
	// Save info about the top level of the location that we'll be archiving.
	archivedPath := filepath.Join(b.mountPoint, path)
	st, err := os.Stat(archivedPath)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(archivedPath, 0755); err != nil {
			return errors.Wrapf(err, "error ensuring volume path %q exists", archivedPath)
		}
		st, err = os.Stat(archivedPath)
	}
	if err != nil {
		logrus.Debugf("error reading info about %q: %v", archivedPath, err)
		return errors.Wrapf(err, "error reading info about volume path %q", archivedPath)
	}
	b.volumeCacheInfo[path] = st
	if !b.volumes.Add(path) {
		// This path is not a subdirectory of a volume path that we're
		// already preserving, so adding it to the list should work.
		return errors.Errorf("error adding %q to the volume cache", path)
	}
	b.volumeCache[path] = cacheFile
	// Now prune cache files for volumes that are now supplanted by this one.
	removed := []string{}
	for cachedPath := range b.volumeCache {
		// Walk our list of cached volumes, and check that they're
		// still in the list of locations that we need to cache.
		found := false
		for _, volume := range b.volumes {
			if volume == cachedPath {
				// We need to keep this volume's cache.
				found = true
				break
			}
		}
		if !found {
			// We don't need to keep this volume's cache.  Make a
			// note to remove it.
			removed = append(removed, cachedPath)
		}
	}
	// Actually remove the caches that we decided to remove.
	for _, cachedPath := range removed {
		archivedPath := filepath.Join(b.mountPoint, cachedPath)
		logrus.Debugf("no longer need cache of %q in %q", archivedPath, b.volumeCache[cachedPath])
		if err := os.Remove(b.volumeCache[cachedPath]); err != nil {
			return errors.Wrapf(err, "error removing %q", b.volumeCache[cachedPath])
		}
		delete(b.volumeCache, cachedPath)
	}
	return nil
}

// Remove any volume cache item which will need to be re-saved because we're
// writing to part of it.
func (b *Executor) volumeCacheInvalidate(path string) error {
	invalidated := []string{}
	for cachedPath := range b.volumeCache {
		if strings.HasPrefix(path, cachedPath+string(os.PathSeparator)) {
			invalidated = append(invalidated, cachedPath)
		}
	}
	for _, cachedPath := range invalidated {
		if err := os.Remove(b.volumeCache[cachedPath]); err != nil {
			return errors.Wrapf(err, "error removing volume cache %q", b.volumeCache[cachedPath])
		}
		archivedPath := filepath.Join(b.mountPoint, cachedPath)
		logrus.Debugf("invalidated volume cache for %q from %q", archivedPath, b.volumeCache[cachedPath])
		delete(b.volumeCache, cachedPath)
	}
	return nil
}

// Save the contents of each of the executor's list of volumes for which we
// don't already have a cache file.
func (b *Executor) volumeCacheSave() error {
	for cachedPath, cacheFile := range b.volumeCache {
		archivedPath := filepath.Join(b.mountPoint, cachedPath)
		_, err := os.Stat(cacheFile)
		if err == nil {
			logrus.Debugf("contents of volume %q are already cached in %q", archivedPath, cacheFile)
			continue
		}
		if !os.IsNotExist(err) {
			return errors.Wrapf(err, "error checking for cache of %q in %q", archivedPath, cacheFile)
		}
		if err := os.MkdirAll(archivedPath, 0755); err != nil {
			return errors.Wrapf(err, "error ensuring volume path %q exists", archivedPath)
		}
		logrus.Debugf("caching contents of volume %q in %q", archivedPath, cacheFile)
		cache, err := os.Create(cacheFile)
		if err != nil {
			return errors.Wrapf(err, "error creating archive at %q", cacheFile)
		}
		defer cache.Close()
		rc, err := archive.Tar(archivedPath, archive.Uncompressed)
		if err != nil {
			return errors.Wrapf(err, "error archiving %q", archivedPath)
		}
		defer rc.Close()
		_, err = io.Copy(cache, rc)
		if err != nil {
			return errors.Wrapf(err, "error archiving %q to %q", archivedPath, cacheFile)
		}
	}
	return nil
}

// Restore the contents of each of the executor's list of volumes.
func (b *Executor) volumeCacheRestore() error {
	for cachedPath, cacheFile := range b.volumeCache {
		archivedPath := filepath.Join(b.mountPoint, cachedPath)
		logrus.Debugf("restoring contents of volume %q from %q", archivedPath, cacheFile)
		cache, err := os.Open(cacheFile)
		if err != nil {
			return errors.Wrapf(err, "error opening archive at %q", cacheFile)
		}
		defer cache.Close()
		if err := os.RemoveAll(archivedPath); err != nil {
			return errors.Wrapf(err, "error clearing volume path %q", archivedPath)
		}
		if err := os.MkdirAll(archivedPath, 0755); err != nil {
			return errors.Wrapf(err, "error recreating volume path %q", archivedPath)
		}
		err = archive.Untar(cache, archivedPath, nil)
		if err != nil {
			return errors.Wrapf(err, "error extracting archive at %q", archivedPath)
		}
		if st, ok := b.volumeCacheInfo[cachedPath]; ok {
			if err := os.Chmod(archivedPath, st.Mode()); err != nil {
				return errors.Wrapf(err, "error restoring permissions on %q", archivedPath)
			}
			if err := os.Chown(archivedPath, 0, 0); err != nil {
				return errors.Wrapf(err, "error setting ownership on %q", archivedPath)
			}
			if err := os.Chtimes(archivedPath, st.ModTime(), st.ModTime()); err != nil {
				return errors.Wrapf(err, "error restoring datestamps on %q", archivedPath)
			}
		}
	}
	return nil
}

// Copy copies data into the working tree.  The "Download" field is how
// imagebuilder tells us the instruction was "ADD" and not "COPY".
func (b *Executor) Copy(excludes []string, copies ...imagebuilder.Copy) error {
	for _, copy := range copies {
		logrus.Debugf("COPY %#v, %#v", excludes, copy)
		if err := b.volumeCacheInvalidate(copy.Dest); err != nil {
			return err
		}
		sources := []string{}
		for _, src := range copy.Src {
			if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
				sources = append(sources, src)
			} else {
				sources = append(sources, filepath.Join(b.contextDir, src))
			}
		}
		if err := b.builder.Add(copy.Dest, copy.Download, sources...); err != nil {
			return err
		}
	}
	return nil
}

func convertMounts(mounts []Mount) []specs.Mount {
	specmounts := []specs.Mount{}
	for _, m := range mounts {
		s := specs.Mount{
			Destination: m.Destination,
			Type:        m.Type,
			Source:      m.Source,
			Options:     m.Options,
		}
		specmounts = append(specmounts, s)
	}
	return specmounts
}

// Run executes a RUN instruction using the working container as a root
// directory.
func (b *Executor) Run(run imagebuilder.Run, config docker.Config) error {
	logrus.Debugf("RUN %#v, %#v", run, config)
	if b.builder == nil {
		return errors.Errorf("no build container available")
	}
	options := buildah.RunOptions{
		Hostname:        config.Hostname,
		Runtime:         b.runtime,
		Args:            b.runtimeArgs,
		Mounts:          convertMounts(b.transientMounts),
		Env:             config.Env,
		User:            config.User,
		WorkingDir:      config.WorkingDir,
		Entrypoint:      config.Entrypoint,
		Cmd:             config.Cmd,
		NetworkDisabled: config.NetworkDisabled,
	}

	args := run.Args
	if run.Shell {
		args = append([]string{"/bin/sh", "-c"}, args...)
	}
	if err := b.volumeCacheSave(); err != nil {
		return err
	}
	err := b.builder.Run(args, options)
	if err2 := b.volumeCacheRestore(); err2 != nil {
		if err == nil {
			return err2
		}
	}
	return err
}

// UnrecognizedInstruction is called when we encounter an instruction that the
// imagebuilder parser didn't understand.
func (b *Executor) UnrecognizedInstruction(step *imagebuilder.Step) error {
	if !b.ignoreUnrecognizedInstructions {
		logrus.Debugf("+(UNIMPLEMENTED?) %#v", step)
		return nil
	}
	logrus.Errorf("+(UNIMPLEMENTED?) %#v", step)
	return errors.Errorf("Unrecognized instruction: %#v", step)
}

// NewExecutor creates a new instance of the imagebuilder.Executor interface.
func NewExecutor(store storage.Store, options BuildOptions) (*Executor, error) {
	exec := Executor{
		store:                          store,
		contextDir:                     options.ContextDirectory,
		pullPolicy:                     options.PullPolicy,
		registry:                       options.Registry,
		transport:                      options.Transport,
		ignoreUnrecognizedInstructions: options.IgnoreUnrecognizedInstructions,
		quiet:               options.Quiet,
		runtime:             options.Runtime,
		runtimeArgs:         options.RuntimeArgs,
		transientMounts:     options.TransientMounts,
		compression:         options.Compression,
		output:              options.Output,
		outputFormat:        options.OutputFormat,
		additionalTags:      options.AdditionalTags,
		signaturePolicyPath: options.SignaturePolicyPath,
		systemContext:       makeSystemContext(options.SignaturePolicyPath),
		volumeCache:         make(map[string]string),
		volumeCacheInfo:     make(map[string]os.FileInfo),
		log:                 options.Log,
		out:                 options.Out,
		err:                 options.Err,
		reportWriter:        options.ReportWriter,
	}
	if exec.err == nil {
		exec.err = os.Stderr
	}
	if exec.out == nil {
		exec.out = os.Stdout
	}
	if exec.log == nil {
		stepCounter := 0
		exec.log = func(format string, args ...interface{}) {
			stepCounter++
			prefix := fmt.Sprintf("STEP %d: ", stepCounter)
			suffix := "\n"
			fmt.Fprintf(exec.err, prefix+format+suffix, args...)
		}
	}
	return &exec, nil
}

// Prepare creates a working container based on specified image, or if one
// isn't specified, the first FROM instruction we can find in the parsed tree.
func (b *Executor) Prepare(ib *imagebuilder.Builder, node *parser.Node, from string) error {
	if from == "" {
		base, err := ib.From(node)
		if err != nil {
			logrus.Debugf("Prepare(node.Children=%#v)", node.Children)
			return errors.Wrapf(err, "error determining starting point for build")
		}
		from = base
	}
	logrus.Debugf("FROM %#v", from)
	if !b.quiet {
		b.log("FROM %s", from)
	}
	builderOptions := buildah.BuilderOptions{
		FromImage:           from,
		PullPolicy:          b.pullPolicy,
		Registry:            b.registry,
		Transport:           b.transport,
		SignaturePolicyPath: b.signaturePolicyPath,
		ReportWriter:        b.reportWriter,
	}
	builder, err := buildah.NewBuilder(b.store, builderOptions)
	if err != nil {
		return errors.Wrapf(err, "error creating build container")
	}
	volumes := map[string]struct{}{}
	for _, v := range builder.Volumes() {
		volumes[v] = struct{}{}
	}
	dConfig := docker.Config{
		Hostname:   builder.Hostname(),
		Domainname: builder.Domainname(),
		User:       builder.User(),
		Env:        builder.Env(),
		Cmd:        builder.Cmd(),
		Image:      from,
		Volumes:    volumes,
		WorkingDir: builder.WorkDir(),
		Entrypoint: builder.Entrypoint(),
		Labels:     builder.Labels(),
	}
	var rootfs *docker.RootFS
	if builder.Docker.RootFS != nil {
		rootfs = &docker.RootFS{
			Type: builder.Docker.RootFS.Type,
		}
		for _, id := range builder.Docker.RootFS.DiffIDs {
			rootfs.Layers = append(rootfs.Layers, id.String())
		}
	}
	dImage := docker.Image{
		Parent:          builder.FromImage,
		ContainerConfig: dConfig,
		Container:       builder.Container,
		Author:          builder.Maintainer(),
		Architecture:    builder.Architecture(),
		RootFS:          rootfs,
	}
	dImage.Config = &dImage.ContainerConfig
	err = ib.FromImage(&dImage, node)
	if err != nil {
		if err2 := builder.Delete(); err2 != nil {
			logrus.Debugf("error deleting container which we failed to update: %v", err2)
		}
		return errors.Wrapf(err, "error updating build context")
	}
	mountPoint, err := builder.Mount("")
	if err != nil {
		if err2 := builder.Delete(); err2 != nil {
			logrus.Debugf("error deleting container which we failed to mount: %v", err2)
		}
		return errors.Wrapf(err, "error mounting new container")
	}
	b.mountPoint = mountPoint
	b.builder = builder
	return nil
}

// Delete deletes the working container, if we have one.  The Executor object
// should not be used to build another image, as the name of the output image
// isn't resettable.
func (b *Executor) Delete() (err error) {
	if b.builder != nil {
		err = b.builder.Delete()
		b.builder = nil
	}
	return err
}

// Execute runs each of the steps in the parsed tree, in turn.
func (b *Executor) Execute(ib *imagebuilder.Builder, node *parser.Node) error {
	for i, node := range node.Children {
		step := ib.Step()
		if err := step.Resolve(node); err != nil {
			return errors.Wrapf(err, "error resolving step %+v", *node)
		}
		logrus.Debugf("Parsed Step: %+v", *step)
		if !b.quiet {
			b.log("%s", step.Original)
		}
		requiresStart := false
		if i < len(node.Children)-1 {
			requiresStart = ib.RequiresStart(&parser.Node{Children: node.Children[i+1:]})
		}
		err := ib.Run(step, b, requiresStart)
		if err != nil {
			return errors.Wrapf(err, "error building at step %+v", *step)
		}
	}
	return nil
}

// Commit writes the container's contents to an image, using a passed-in tag as
// the name if there is one, generating a unique ID-based one otherwise.
func (b *Executor) Commit(ib *imagebuilder.Builder) (err error) {
	var imageRef types.ImageReference
	if b.output != "" {
		imageRef, err = alltransports.ParseImageName(b.output)
		if err != nil {
			imageRef2, err2 := is.Transport.ParseStoreReference(b.store, b.output)
			if err2 == nil {
				imageRef = imageRef2
				err = nil
			}
		}
	} else {
		imageRef, err = is.Transport.ParseStoreReference(b.store, "@"+stringid.GenerateRandomID())
	}
	if err != nil {
		return errors.Wrapf(err, "error parsing reference for image to be written")
	}
	config := ib.Config()
	b.builder.SetHostname(config.Hostname)
	b.builder.SetDomainname(config.Domainname)
	b.builder.SetUser(config.User)
	b.builder.ClearPorts()
	for p := range config.ExposedPorts {
		b.builder.SetPort(string(p))
	}
	b.builder.ClearEnv()
	for _, envSpec := range config.Env {
		spec := strings.SplitN(envSpec, "=", 2)
		b.builder.SetEnv(spec[0], spec[1])
	}
	b.builder.SetCmd(config.Cmd)
	b.builder.ClearVolumes()
	for v := range config.Volumes {
		b.builder.AddVolume(v)
	}
	b.builder.SetWorkDir(config.WorkingDir)
	b.builder.SetEntrypoint(config.Entrypoint)
	b.builder.ClearLabels()
	for k, v := range config.Labels {
		b.builder.SetLabel(k, v)
	}
	if imageRef != nil {
		logName := transports.ImageName(imageRef)
		logrus.Debugf("COMMIT %q", logName)
		if !b.quiet {
			b.log("COMMIT %s", logName)
		}
	} else {
		logrus.Debugf("COMMIT")
		if !b.quiet {
			b.log("COMMIT")
		}
	}
	options := buildah.CommitOptions{
		Compression:           b.compression,
		SignaturePolicyPath:   b.signaturePolicyPath,
		AdditionalTags:        b.additionalTags,
		ReportWriter:          b.reportWriter,
		PreferredManifestType: b.outputFormat,
	}
	return b.builder.Commit(imageRef, options)
}

// Build takes care of the details of running Prepare/Execute/Commit/Delete
// over each of the one or more parsed Dockerfiles.
func (b *Executor) Build(ib *imagebuilder.Builder, node []*parser.Node) (err error) {
	if len(node) == 0 {
		return errors.Wrapf(err, "error building: no build instructions")
	}
	first := node[0]
	from, err := ib.From(first)
	if err != nil {
		logrus.Debugf("Build(first.Children=%#v)", first.Children)
		return errors.Wrapf(err, "error determining starting point for build")
	}
	if err = b.Prepare(ib, first, from); err != nil {
		return err
	}
	defer b.Delete()
	for _, this := range node {
		if err = b.Execute(ib, this); err != nil {
			return err
		}
	}
	if err = b.Commit(ib); err != nil {
		return err
	}
	return nil
}

// BuildReadClosers parses a set of one or more already-opened Dockerfiles,
// creates a new Executor, and then runs Prepare/Execute/Commit/Delete over the
// entire set of instructions.
func BuildReadClosers(store storage.Store, options BuildOptions, dockerfile ...io.ReadCloser) error {
	mainFile := dockerfile[0]
	extraFiles := dockerfile[1:]
	for _, dfile := range dockerfile {
		defer dfile.Close()
	}
	builder, parsed, err := imagebuilder.NewBuilderForReader(mainFile, options.Args)
	if err != nil {
		return errors.Wrapf(err, "error creating builder")
	}
	exec, err := NewExecutor(store, options)
	if err != nil {
		return errors.Wrapf(err, "error creating build executor")
	}
	nodes := []*parser.Node{parsed}
	for _, extra := range extraFiles {
		_, parsed, err := imagebuilder.NewBuilderForReader(extra, options.Args)
		if err != nil {
			return errors.Wrapf(err, "error parsing dockerfile")
		}
		nodes = append(nodes, parsed)
	}
	return exec.Build(builder, nodes)
}

// BuildDockerfiles parses a set of one or more Dockerfiles (which may be
// URLs), creates a new Executor, and then runs Prepare/Execute/Commit/Delete
// over the entire set of instructions.
func BuildDockerfiles(store storage.Store, options BuildOptions, dockerfile ...string) error {
	var dockerfiles []io.ReadCloser
	if len(dockerfile) == 0 {
		return errors.Errorf("error building: no dockerfiles specified")
	}
	for _, dfile := range dockerfile {
		var rc io.ReadCloser
		if strings.HasPrefix(dfile, "http://") || strings.HasPrefix(dfile, "https://") {
			logrus.Debugf("reading remote Dockerfile %q", dfile)
			resp, err := http.Get(dfile)
			if err != nil {
				return errors.Wrapf(err, "error getting %q", dfile)
			}
			if resp.ContentLength == 0 {
				resp.Body.Close()
				return errors.Errorf("no contents in %q", dfile)
			}
			rc = resp.Body
		} else {
			if !filepath.IsAbs(dfile) {
				logrus.Debugf("resolving local Dockerfile %q", dfile)
				dfile = filepath.Join(options.ContextDirectory, dfile)
			}
			logrus.Debugf("reading local Dockerfile %q", dfile)
			contents, err := os.Open(dfile)
			if err != nil {
				return errors.Wrapf(err, "error reading %q", dfile)
			}
			dinfo, err := contents.Stat()
			if err != nil {
				contents.Close()
				return errors.Wrapf(err, "error reading info about %q", dfile)
			}
			if dinfo.Size() == 0 {
				contents.Close()
				return errors.Wrapf(err, "no contents in %q", dfile)
			}
			rc = contents
		}
		dockerfiles = append(dockerfiles, rc)
	}
	if err := BuildReadClosers(store, options, dockerfiles...); err != nil {
		return errors.Wrapf(err, "error building")
	}
	return nil
}
