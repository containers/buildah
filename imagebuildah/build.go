package imagebuildah

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	is "github.com/containers/image/storage"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	"github.com/containers/image/types"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/stringid"
	"github.com/containers/storage/storage"
	"github.com/docker/docker/builder/dockerfile/parser"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/openshift/imagebuilder"
	"github.com/projectatomic/buildah"
)

const (
	PullIfMissing  = buildah.PullIfMissing
	PullAlways     = buildah.PullAlways
	PullNever      = buildah.PullNever
	DefaultRuntime = buildah.DefaultRuntime

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
	// reference to a source image.
	Registry string
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
}

// Executor is a buildah-based implementation of the imagebuilder.Executor
// interface.
type Executor struct {
	store                          storage.Store
	contextDir                     string
	builder                        *buildah.Builder
	pullPolicy                     int
	registry                       string
	ignoreUnrecognizedInstructions bool
	quiet                          bool
	runtime                        string
	runtimeArgs                    []string
	transientMounts                []Mount
	compression                    archive.Compression
	output                         string
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
		// we're already preserving, so there's nothing new to be done.
		return nil
	}
	// Figure out where the cache for this volume would be stored.
	b.preserved++
	cacheDir, err := b.store.GetContainerDirectory(b.builder.ContainerID)
	if err != nil {
		return fmt.Errorf("unable to locate temporary directory for container")
	}
	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("volume%d.tar", b.preserved))
	// Save info about the top level of the location that we'll be archiving.
	archivedPath := filepath.Join(b.mountPoint, path)
	st, err := os.Stat(archivedPath)
	if err != nil {
		logrus.Debugf("error reading info about %q: %v", archivedPath, err)
		return err
	}
	b.volumeCacheInfo[path] = st
	if !b.volumes.Add(path) {
		// This path is not a subdirectory of a volume path that we're
		// already preserving, so adding it to the list should work.
		return fmt.Errorf("error adding %q to the volume cache", path)
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
			return fmt.Errorf("error removing %q: %v", b.volumeCache[cachedPath], err)
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
			return fmt.Errorf("error removing volume cache %q: %v", b.volumeCache[cachedPath], err)
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
			return fmt.Errorf("error checking for cache of %q in %q: %v", archivedPath, cacheFile, err)
		}
		logrus.Debugf("caching contents of volume %q in %q", archivedPath, cacheFile)
		cache, err := os.Create(cacheFile)
		if err != nil {
			return fmt.Errorf("error creating archive at %q: %v", cacheFile, err)
		}
		defer cache.Close()
		rc, err := archive.Tar(archivedPath, archive.Uncompressed)
		if err != nil {
			return fmt.Errorf("error archiving %q: %v", archivedPath, err)
		}
		defer rc.Close()
		_, err = io.Copy(cache, rc)
		if err != nil {
			return fmt.Errorf("error archiving %q to %q: %v", archivedPath, cacheFile, err)
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
			return fmt.Errorf("error opening archive at %q: %v", cacheFile, err)
		}
		defer cache.Close()
		if err := os.RemoveAll(archivedPath); err != nil {
			return fmt.Errorf("error clearing volume path %q: %v", archivedPath, err)
		}
		if err := os.MkdirAll(archivedPath, 0700); err != nil {
			return fmt.Errorf("error recreating volume path %q: %v", archivedPath, err)
		}
		err = archive.Untar(cache, archivedPath, nil)
		if err != nil {
			return fmt.Errorf("error extracting archive at %q: %v", archivedPath, err)
		}
		if st, ok := b.volumeCacheInfo[cachedPath]; ok {
			if err := os.Chmod(archivedPath, st.Mode()); err != nil {
				return fmt.Errorf("error restoring permissions on %q: %v", archivedPath, err)
			}
			if err := os.Chown(archivedPath, 0, 0); err != nil {
				return fmt.Errorf("error setting ownership on %q: %v", archivedPath, err)
			}
			if err := os.Chtimes(archivedPath, st.ModTime(), st.ModTime()); err != nil {
				return fmt.Errorf("error restoring datestamps on %q: %v", archivedPath, err)
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
		return fmt.Errorf("no build container available")
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
	return fmt.Errorf("Unrecognized instruction: %#v", step)
}

// NewExecutor creates a new instance of the imagebuilder.Executor interface.
func NewExecutor(store storage.Store, options BuildOptions) (*Executor, error) {
	exec := Executor{
		store:                          store,
		contextDir:                     options.ContextDirectory,
		pullPolicy:                     options.PullPolicy,
		registry:                       options.Registry,
		ignoreUnrecognizedInstructions: options.IgnoreUnrecognizedInstructions,
		quiet:               options.Quiet,
		runtime:             options.Runtime,
		runtimeArgs:         options.RuntimeArgs,
		transientMounts:     options.TransientMounts,
		compression:         options.Compression,
		output:              options.Output,
		additionalTags:      options.AdditionalTags,
		signaturePolicyPath: options.SignaturePolicyPath,
		systemContext:       makeSystemContext(options.SignaturePolicyPath),
		volumeCache:         make(map[string]string),
		volumeCacheInfo:     make(map[string]os.FileInfo),
		log:                 options.Log,
		out:                 options.Out,
		err:                 options.Err,
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
			return fmt.Errorf("error determining starting point for build: %v", err)
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
		SignaturePolicyPath: b.signaturePolicyPath,
	}
	builder, err := buildah.NewBuilder(b.store, builderOptions)
	if err != nil {
		return fmt.Errorf("error creating build container: %v", err)
	}
	dImage := docker.Image{
		Config: &docker.Config{
			Env:   builder.UpdatedEnv(),
			Image: from,
		},
	}
	err = ib.FromImage(&dImage, node)
	if err != nil {
		if err2 := builder.Delete(); err2 != nil {
			logrus.Debugf("error deleting container which we failed to update: %v", err2)
		}
		return fmt.Errorf("error updating build context: %v", err)
	}
	mountPoint, err := builder.Mount("")
	if err != nil {
		if err2 := builder.Delete(); err2 != nil {
			logrus.Debugf("error deleting container which we failed to mount: %v", err2)
		}
		return fmt.Errorf("error mounting new container: %v", err)
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
			return fmt.Errorf("error resolving step %+v: %v", *node, err)
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
			return fmt.Errorf("error building at step %+v: %v", *step, err)
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
	if err != nil {
		return err
	}
	options := buildah.CommitOptions{
		Compression:         b.compression,
		SignaturePolicyPath: b.signaturePolicyPath,
		AdditionalTags:      b.additionalTags,
	}
	return b.builder.Commit(imageRef, options)
}

// Build takes care of the details of running Prepare/Execute/Commit/Delete
// over each of the one or more parsed Dockerfiles.
func (b *Executor) Build(ib *imagebuilder.Builder, node []*parser.Node) (err error) {
	if len(node) == 0 {
		return fmt.Errorf("error building: no build instructions")
	}
	first := node[0]
	from, err := ib.From(first)
	if err != nil {
		logrus.Debugf("Build(first.Children=%#v)", first.Children)
		return fmt.Errorf("error determining starting point for build: %v", err)
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
		return fmt.Errorf("error creating builder: %v", err)
	}
	exec, err := NewExecutor(store, options)
	if err != nil {
		return fmt.Errorf("error creating build executor: %v", err)
	}
	nodes := []*parser.Node{parsed}
	for _, extra := range extraFiles {
		_, parsed, err := imagebuilder.NewBuilderForReader(extra, options.Args)
		if err != nil {
			return fmt.Errorf("error parsing dockerfile: %v", err)
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
		return fmt.Errorf("error building: no dockerfiles specified")
	}
	for _, dfile := range dockerfile {
		var rc io.ReadCloser
		if strings.HasPrefix(dfile, "http://") || strings.HasPrefix(dfile, "https://") {
			logrus.Debugf("reading remote Dockerfile %q", dfile)
			resp, err := http.Get(dfile)
			if err != nil {
				return fmt.Errorf("error getting %q: %v", dfile, err)
			}
			if resp.ContentLength == 0 {
				resp.Body.Close()
				return fmt.Errorf("no contents in %q", dfile)
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
				return fmt.Errorf("error reading %q: %v", dfile, err)
			}
			dinfo, err := contents.Stat()
			if err != nil {
				contents.Close()
				return fmt.Errorf("error reading info about %q: %v", dfile, err)
			}
			if dinfo.Size() == 0 {
				contents.Close()
				return fmt.Errorf("no contents in %q: %v", dfile, err)
			}
			rc = contents
		}
		dockerfiles = append(dockerfiles, rc)
	}
	if err := BuildReadClosers(store, options, dockerfiles...); err != nil {
		return fmt.Errorf("error building: %v", err)
	}
	return nil
}
