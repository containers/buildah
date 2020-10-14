package dockerclient

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	docker "github.com/fsouza/go-dockerclient"
	"k8s.io/klog"

	"github.com/openshift/imagebuilder"
	"github.com/openshift/imagebuilder/dockerfile/parser"
	"github.com/openshift/imagebuilder/imageprogress"
)

// NewClientFromEnv is exposed to simplify getting a client when vendoring this library.
func NewClientFromEnv() (*docker.Client, error) {
	return docker.NewClientFromEnv()
}

// Mount represents a binding between the current system and the destination client
type Mount struct {
	SourcePath      string
	DestinationPath string
}

// ClientExecutor can run Docker builds from a Docker client.
type ClientExecutor struct {
	// Name is an optional name for this executor.
	Name string
	// Named is a map of other named executors.
	Named map[string]*ClientExecutor

	// TempDir is the temporary directory to use for storing file
	// contents. If unset, the default temporary directory for the
	// system will be used.
	TempDir string
	// Client is a client to a Docker daemon.
	Client *docker.Client
	// Directory is the context directory to build from, will use
	// the current working directory if not set. Ignored if
	// ContextArchive is set.
	Directory string
	// A compressed or uncompressed tar archive that should be used
	// as the build context.
	ContextArchive string
	// Excludes are a list of file patterns that should be excluded
	// from the context. Will be set to the contents of the
	// .dockerignore file if nil.
	Excludes []string
	// Tag is an optional value to tag the resulting built image.
	Tag string
	// Additional tags is an optional array of other tags to apply
	// to the image.
	AdditionalTags []string
	// AllowPull when set will pull images that are not present on
	// the daemon.
	AllowPull bool
	// IgnoreUnrecognizedInstructions, if true, allows instructions
	// that are not yet supported to be ignored (will be printed)
	IgnoreUnrecognizedInstructions bool
	// StrictVolumeOwnership if true will fail the build if a RUN
	// command follows a VOLUME command, since this client cannot
	// guarantee that the restored contents of the VOLUME directory
	// will have the right permissions.
	StrictVolumeOwnership bool
	// TransientMounts are a set of mounts from outside the build
	// to the inside that will not be part of the final image. Any
	// content created inside the mount's destinationPath will be
	// omitted from the final image.
	TransientMounts []Mount

	// The path within the container to perform the transient mount.
	ContainerTransientMount string

	// The streams used for canonical output.
	Out, ErrOut io.Writer

	// Container is optional and can be set to a container to use as
	// the execution environment for a build.
	Container *docker.Container
	// Command, if set, will be used as the entrypoint for the new
	// container. This is ignored if Container is set.
	Command []string
	// Image is optional and may be set to control which image is used
	// as a base for this build. Otherwise the FROM value from the
	// Dockerfile is read (will be pulled if not locally present).
	Image *docker.Image

	// AuthFn will handle authenticating any docker pulls if Image
	// is set to nil.
	AuthFn func(name string) ([]dockertypes.AuthConfig, bool)
	// HostConfig is used to start the container (if necessary).
	HostConfig *docker.HostConfig
	// LogFn is an optional command to log information to the end user
	LogFn func(format string, args ...interface{})

	// Deferred is a list of operations that must be cleaned up at
	// the end of execution. Use Release() to handle these.
	Deferred []func() error

	// Volumes handles saving and restoring volumes after RUN
	// commands are executed.
	Volumes *ContainerVolumeTracker
}

// NotAuthFn can be used for AuthFn when no authentication is required in Docker.
func NoAuthFn(string) ([]dockertypes.AuthConfig, bool) {
	return nil, false
}

// NewClientExecutor creates a client executor.
func NewClientExecutor(client *docker.Client) *ClientExecutor {
	return &ClientExecutor{
		Client: client,
		LogFn:  func(string, ...interface{}) {},

		ContainerTransientMount: "/.imagebuilder-transient-mount",
	}
}

func (e *ClientExecutor) DefaultExcludes() error {
	var err error
	e.Excludes, err = imagebuilder.ParseDockerignore(e.Directory)
	return err
}

// WithName creates a new child executor that will be used whenever a COPY statement
// uses --from=NAME.
func (e *ClientExecutor) WithName(name string) *ClientExecutor {
	if e.Named == nil {
		e.Named = make(map[string]*ClientExecutor)
		e.Deferred = append([]func() error{func() error {
			var errs []error
			for _, named := range e.Named {
				errs = append(errs, named.Release()...)
			}
			if len(errs) > 0 {
				return fmt.Errorf("%v", errs)
			}
			return nil
		}}, e.Deferred...)
	}

	copied := *e
	copied.Name = name
	copied.Container = nil
	copied.Deferred = nil
	copied.Image = nil
	copied.Volumes = nil

	child := &copied
	e.Named[name] = child
	return child
}

// Stages executes all of the provided stages, starting from the base image. It returns the executor of the last stage
// or an error if a stage fails.
func (e *ClientExecutor) Stages(b *imagebuilder.Builder, stages imagebuilder.Stages, from string) (*ClientExecutor, error) {
	var stageExecutor *ClientExecutor
	for i, stage := range stages {
		stageExecutor = e.WithName(stage.Name)

		var stageFrom string
		if i == 0 {
			stageFrom = from
		} else {
			from, err := b.From(stage.Node)
			if err != nil {
				return nil, err
			}
			if prereq := e.Named[from]; prereq != nil {
				b, ok := stages.ByName(from)
				if !ok {
					return nil, fmt.Errorf("error: Unable to find stage %s builder", from)
				}
				stageExecutor.Image = &docker.Image{
					Config: b.Builder.Config(),
				}
				stageExecutor.Container = prereq.Container
				klog.V(4).Infof("Using previous stage %s as image: %#v", from, stageExecutor.Image.Config)
			}
			stageFrom = from
		}

		if err := stageExecutor.Prepare(stage.Builder, stage.Node, stageFrom); err != nil {
			return nil, err
		}
		if err := stageExecutor.Execute(stage.Builder, stage.Node); err != nil {
			return nil, err
		}

		// remember the outcome of the stage execution on the container config in case
		// another stage needs to access incremental state
		stageExecutor.Container.Config = stage.Builder.Config()
	}
	return stageExecutor, nil
}

// Build is a helper method to perform a Docker build against the
// provided Docker client. It will load the image if not specified,
// create a container if one does not already exist, and start a
// container if the Dockerfile contains RUN commands. It will cleanup
// any containers it creates directly, and set the e.Image.ID field
// to the generated image.
func (e *ClientExecutor) Build(b *imagebuilder.Builder, node *parser.Node, from string) error {
	defer e.Release()
	if err := e.Prepare(b, node, from); err != nil {
		return err
	}
	if err := e.Execute(b, node); err != nil {
		return err
	}
	return e.Commit(b)
}

func (e *ClientExecutor) Prepare(b *imagebuilder.Builder, node *parser.Node, from string) error {
	var err error

	// identify the base image
	if len(from) == 0 {
		from, err = b.From(node)
		if err != nil {
			return err
		}
	}

	// load the image
	if e.Image == nil {
		if from == imagebuilder.NoBaseImageSpecifier {
			if runtime.GOOS == "windows" {
				return fmt.Errorf("building from scratch images is not supported")
			}
			from, err = e.CreateScratchImage()
			if err != nil {
				return fmt.Errorf("unable to create a scratch image for this build: %v", err)
			}
			e.Deferred = append(e.Deferred, func() error { return e.Client.RemoveImage(from) })
		}
		klog.V(4).Infof("Retrieving image %q", from)
		e.Image, err = e.LoadImage(from)
		if err != nil {
			return err
		}
	}

	// update the builder with any information from the image, including ONBUILD
	// statements
	if err := b.FromImage(e.Image, node); err != nil {
		return err
	}

	b.RunConfig.Image = from
	if len(e.Name) > 0 {
		e.LogFn("FROM %s as %s", from, e.Name)
	} else {
		e.LogFn("FROM %s", from)
	}
	klog.V(4).Infof("step: FROM %s as %s", from, e.Name)

	b.Excludes = e.Excludes

	var sharedMount string

	defaultShell := b.RunConfig.Shell
	if len(defaultShell) == 0 {
		defaultShell = []string{"/bin/sh", "-c"}
	}

	// create a container to execute in, if necessary
	mustStart := b.RequiresStart(node)
	if e.Container == nil {
		opts := docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: from,
			},
			HostConfig: &docker.HostConfig{},
		}
		if e.HostConfig != nil {
			opts.HostConfig = e.HostConfig
		}
		originalBinds := opts.HostConfig.Binds

		if mustStart {
			// Transient mounts only make sense on images that will be running processes
			if len(e.TransientMounts) > 0 {
				volumeName, err := randSeq(imageSafeCharacters, 24)
				if err != nil {
					return err
				}
				v, err := e.Client.CreateVolume(docker.CreateVolumeOptions{Name: volumeName})
				if err != nil {
					return fmt.Errorf("unable to create volume to mount secrets: %v", err)
				}
				e.Deferred = append(e.Deferred, func() error { return e.Client.RemoveVolume(volumeName) })
				sharedMount = v.Mountpoint
				opts.HostConfig.Binds = append(opts.HostConfig.Binds, volumeName+":"+e.ContainerTransientMount)
			}

			// TODO: windows support
			if len(e.Command) > 0 {
				opts.Config.Cmd = e.Command
				opts.Config.Entrypoint = nil
			} else {
				// TODO; replace me with a better default command
				opts.Config.Cmd = []string{fmt.Sprintf("%s\nsleep 86400", "#(imagebuilder)")}
				opts.Config.Entrypoint = append([]string{}, defaultShell...)
			}
		}

		if len(opts.Config.Cmd) == 0 {
			opts.Config.Entrypoint = append(append([]string{}, defaultShell...), "#(imagebuilder)")
		}

		// copy any source content into the temporary mount path
		if mustStart && len(e.TransientMounts) > 0 {
			if len(sharedMount) == 0 {
				return fmt.Errorf("no mount point available for temporary mounts")
			}
			binds, err := e.PopulateTransientMounts(opts, e.TransientMounts, sharedMount)
			if err != nil {
				return err
			}
			opts.HostConfig.Binds = append(originalBinds, binds...)
		}

		klog.V(4).Infof("Creating container with %#v %#v", opts.Config, opts.HostConfig)
		container, err := e.Client.CreateContainer(opts)
		if err != nil {
			return fmt.Errorf("unable to create build container: %v", err)
		}
		e.Container = container
		e.Deferred = append([]func() error{func() error { return e.removeContainer(container.ID) }}, e.Deferred...)
	}

	// TODO: lazy start
	if mustStart && !e.Container.State.Running {
		if err := e.Client.StartContainer(e.Container.ID, nil); err != nil {
			return fmt.Errorf("unable to start build container: %v", err)
		}
		e.Container.State.Running = true
		// TODO: is this racy? may have to loop wait in the actual run step
	}
	return nil
}

// Execute performs all of the provided steps against the initialized container. May be
// invoked multiple times for a given container.
func (e *ClientExecutor) Execute(b *imagebuilder.Builder, node *parser.Node) error {
	for i, child := range node.Children {
		step := b.Step()
		if err := step.Resolve(child); err != nil {
			return err
		}
		klog.V(4).Infof("step: %s", step.Original)
		if e.LogFn != nil {
			// original may have unescaped %, so perform fmt escaping
			e.LogFn(strings.Replace(step.Original, "%", "%%", -1))
		}
		noRunsRemaining := !b.RequiresStart(&parser.Node{Children: node.Children[i+1:]})

		if err := b.Run(step, e, noRunsRemaining); err != nil {
			return err
		}
	}

	return nil
}

// Commit saves the completed build as an image with the provided tag. It will
// stop the container, commit the image, and then remove the container.
func (e *ClientExecutor) Commit(b *imagebuilder.Builder) error {
	config := b.Config()

	if e.Container.State.Running {
		klog.V(4).Infof("Stopping container %s ...", e.Container.ID)
		if err := e.Client.StopContainer(e.Container.ID, 0); err != nil {
			return fmt.Errorf("unable to stop build container: %v", err)
		}
		e.Container.State.Running = false
		// Starting the container may perform escaping of args, so to be consistent
		// we also set that here
		config.ArgsEscaped = true
	}

	var repository, tag string
	if len(e.Tag) > 0 {
		repository, tag = docker.ParseRepositoryTag(e.Tag)
		klog.V(4).Infof("Committing built container %s as image %q: %#v", e.Container.ID, e.Tag, config)
		if e.LogFn != nil {
			e.LogFn("Committing changes to %s ...", e.Tag)
		}
	} else {
		klog.V(4).Infof("Committing built container %s: %#v", e.Container.ID, config)
		if e.LogFn != nil {
			e.LogFn("Committing changes ...")
		}
	}

	defer func() {
		for _, err := range e.Release() {
			e.LogFn("Unable to cleanup: %v", err)
		}
	}()

	image, err := e.Client.CommitContainer(docker.CommitContainerOptions{
		Author:     b.Author,
		Container:  e.Container.ID,
		Run:        config,
		Repository: repository,
		Tag:        tag,
	})
	if err != nil {
		return fmt.Errorf("unable to commit build container: %v", err)
	}

	e.Image = image
	klog.V(4).Infof("Committed %s to %s", e.Container.ID, image.ID)

	if len(e.Tag) > 0 {
		for _, s := range e.AdditionalTags {
			repository, tag := docker.ParseRepositoryTag(s)
			err := e.Client.TagImage(image.ID, docker.TagImageOptions{
				Repo: repository,
				Tag:  tag,
			})
			if err != nil {
				e.Deferred = append(e.Deferred, func() error { return e.Client.RemoveImageExtended(image.ID, docker.RemoveImageOptions{Force: true}) })
				return fmt.Errorf("unable to tag %q: %v", s, err)
			}
			e.LogFn("Tagged as %s", s)
		}
	}

	if e.LogFn != nil {
		e.LogFn("Done")
	}
	return nil
}

func (e *ClientExecutor) PopulateTransientMounts(opts docker.CreateContainerOptions, transientMounts []Mount, sharedMount string) ([]string, error) {
	container, err := e.Client.CreateContainer(opts)
	if err != nil {
		return nil, fmt.Errorf("unable to create transient container: %v", err)
	}
	defer e.removeContainer(container.ID)

	var copies []imagebuilder.Copy
	for i, mount := range transientMounts {
		copies = append(copies, imagebuilder.Copy{
			FromFS: true,
			Src:    []string{mount.SourcePath},
			Dest:   filepath.Join(e.ContainerTransientMount, strconv.Itoa(i)),
		})
	}
	if err := e.CopyContainer(container, nil, copies...); err != nil {
		return nil, fmt.Errorf("unable to copy transient context into container: %v", err)
	}

	// mount individual items temporarily
	var binds []string
	for i, mount := range e.TransientMounts {
		binds = append(binds, fmt.Sprintf("%s:%s:%s", filepath.Join(sharedMount, strconv.Itoa(i)), mount.DestinationPath, "ro"))
	}
	return binds, nil
}

// Release deletes any items started by this executor.
func (e *ClientExecutor) Release() []error {
	errs := e.Volumes.Release()
	for _, fn := range e.Deferred {
		if err := fn(); err != nil {
			errs = append(errs, err)
		}
	}
	e.Deferred = nil
	return errs
}

// removeContainer removes the provided container ID
func (e *ClientExecutor) removeContainer(id string) error {
	e.Client.StopContainer(id, 0)
	err := e.Client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	})
	if _, ok := err.(*docker.NoSuchContainer); err != nil && !ok {
		return fmt.Errorf("unable to cleanup container: %v", err)
	}
	return nil
}

// CreateScratchImage creates a new, zero byte layer that is identical to "scratch"
// except that the resulting image will have two layers.
func (e *ClientExecutor) CreateScratchImage() (string, error) {
	random, err := randSeq(imageSafeCharacters, 24)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("scratch%s", random)

	buf := &bytes.Buffer{}
	w := tar.NewWriter(buf)
	w.Close()

	return name, e.Client.ImportImage(docker.ImportImageOptions{
		Repository:  name,
		Source:      "-",
		InputStream: buf,
	})
}

// imageSafeCharacters are characters allowed to be part of a Docker image name.
const imageSafeCharacters = "abcdefghijklmnopqrstuvwxyz0123456789"

// randSeq returns a sequence of random characters drawn from source. It returns
// an error if cryptographic randomness is not available or source is more than 255
// characters.
func randSeq(source string, n int) (string, error) {
	if len(source) > 255 {
		return "", fmt.Errorf("source must be less than 256 bytes long")
	}
	random := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, random); err != nil {
		return "", err
	}
	for i := range random {
		random[i] = source[random[i]%byte(len(source))]
	}
	return string(random), nil
}

// LoadImage checks the client for an image matching from. If not found,
// attempts to pull the image and then tries to inspect again.
func (e *ClientExecutor) LoadImage(from string) (*docker.Image, error) {
	image, err := e.Client.InspectImage(from)
	if err == nil {
		return image, nil
	}
	if err != docker.ErrNoSuchImage {
		return nil, err
	}

	if !e.AllowPull {
		klog.V(4).Infof("image %s did not exist", from)
		return nil, docker.ErrNoSuchImage
	}

	repository, tag := docker.ParseRepositoryTag(from)
	if len(tag) == 0 {
		tag = "latest"
	}

	klog.V(4).Infof("attempting to pull %s with auth from repository %s:%s", from, repository, tag)

	// TODO: we may want to abstract looping over multiple credentials
	auth, _ := e.AuthFn(repository)
	if len(auth) == 0 {
		auth = append(auth, dockertypes.AuthConfig{})
	}

	if e.LogFn != nil {
		e.LogFn("Image %s was not found, pulling ...", from)
	}

	var lastErr error
	outputProgress := func(s string) {
		e.LogFn("%s", s)
	}
	for _, config := range auth {
		// TODO: handle IDs?
		var pullErr error
		func() { // A scope for defer
			pullWriter := imageprogress.NewPullWriter(outputProgress)
			defer func() {
				err := pullWriter.Close()
				if pullErr == nil {
					pullErr = err
				}
			}()

			pullImageOptions := docker.PullImageOptions{
				Repository:    repository,
				Tag:           tag,
				OutputStream:  pullWriter,
				RawJSONStream: true,
			}
			if klog.V(5) {
				pullImageOptions.OutputStream = os.Stderr
				pullImageOptions.RawJSONStream = false
			}
			authConfig := docker.AuthConfiguration{Username: config.Username, ServerAddress: config.ServerAddress, Password: config.Password}
			pullErr = e.Client.PullImage(pullImageOptions, authConfig)
		}()
		if pullErr == nil {
			break
		}
		lastErr = pullErr
		continue
	}
	if lastErr != nil {
		return nil, fmt.Errorf("unable to pull image (from: %s, tag: %s): %v", repository, tag, lastErr)
	}

	return e.Client.InspectImage(from)
}

func (e *ClientExecutor) Preserve(path string) error {
	if e.Volumes == nil {
		e.Volumes = NewContainerVolumeTracker()
	}

	if err := e.EnsureContainerPath(path); err != nil {
		return err
	}

	e.Volumes.Add(path)
	return nil
}

func (e *ClientExecutor) EnsureContainerPath(path string) error {
	createPath := func(dest string) error {
		var writerErr error
		if !strings.HasSuffix(dest, "/") {
			dest = dest + "/"
		}
		reader, writer := io.Pipe()
		opts := docker.UploadToContainerOptions{
			InputStream: reader,
			Path:        "/",
			Context:     context.TODO(),
		}
		go func() {
			defer writer.Close()
			tarball := tar.NewWriter(writer)
			defer tarball.Close()
			writerErr = tarball.WriteHeader(&tar.Header{
				Name:     dest,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			})
		}()
		klog.V(4).Infof("Uploading empty archive to %q", dest)
		err := e.Client.UploadToContainer(e.Container.ID, opts)
		if err != nil {
			return fmt.Errorf("unable to ensure existence of preserved path %s: %v", dest, err)
		}
		if writerErr != nil {
			return fmt.Errorf("error generating tarball to ensure existence of preserved path %s: %v", dest, writerErr)
		}
		return nil
	}
	readPath := func(dest string) error {
		if !strings.HasSuffix(dest, "/") {
			dest = dest + "/"
		}
		err := e.Client.DownloadFromContainer(e.Container.ID, docker.DownloadFromContainerOptions{
			Path:         dest,
			OutputStream: ioutil.Discard,
		})
		return err
	}
	if err := readPath(path); err != nil {
		if err = createPath(path); err != nil {
			return fmt.Errorf("error creating container directory %s: %v", path, err)
		}
	}
	return nil
}

func (e *ClientExecutor) UnrecognizedInstruction(step *imagebuilder.Step) error {
	if e.IgnoreUnrecognizedInstructions {
		e.LogFn("warning: Unknown instruction: %s", strings.ToUpper(step.Command))
		return nil
	}
	return fmt.Errorf("Unknown instruction: %s", strings.ToUpper(step.Command))
}

// Run executes a single Run command against the current container using exec().
// Since exec does not allow ENV or WORKINGDIR to be set, we force the execution of
// the user command into a shell and perform those operations before. Since RUN
// requires /bin/sh, we can use both 'cd' and 'export'.
func (e *ClientExecutor) Run(run imagebuilder.Run, config docker.Config) error {
	args := make([]string, len(run.Args))
	copy(args, run.Args)

	defaultShell := config.Shell
	if len(defaultShell) == 0 {
		if runtime.GOOS == "windows" {
			defaultShell = []string{"cmd", "/S", "/C"}
		} else {
			defaultShell = []string{"/bin/sh", "-c"}
		}
	}
	if runtime.GOOS == "windows" {
		if len(config.WorkingDir) > 0 {
			args[0] = fmt.Sprintf("cd %s && %s", imagebuilder.BashQuote(config.WorkingDir), args[0])
		}
		// TODO: implement windows ENV
		args = append(defaultShell, args...)
	} else {
		if run.Shell {
			if len(config.WorkingDir) > 0 {
				args[0] = fmt.Sprintf("cd %s && %s", imagebuilder.BashQuote(config.WorkingDir), args[0])
			}
			if len(config.Env) > 0 {
				args[0] = imagebuilder.ExportEnv(config.Env) + args[0]
			}
			args = append(defaultShell, args...)
		} else {
			switch {
			case len(config.WorkingDir) == 0 && len(config.Env) == 0:
				// no change necessary
			case len(args) > 0:
				setup := "exec \"$@\""
				if len(config.WorkingDir) > 0 {
					setup = fmt.Sprintf("cd %s && %s", imagebuilder.BashQuote(config.WorkingDir), setup)
				}
				if len(config.Env) > 0 {
					setup = imagebuilder.ExportEnv(config.Env) + setup
				}
				newArgs := make([]string, 0, len(args)+4)
				newArgs = append(newArgs, defaultShell...)
				newArgs = append(newArgs, setup, "")
				newArgs = append(newArgs, args...)
				args = newArgs
			}
		}
	}

	if e.StrictVolumeOwnership && !e.Volumes.Empty() {
		return fmt.Errorf("a RUN command was executed after a VOLUME command, which may result in ownership information being lost")
	}
	if err := e.Volumes.Save(e.Container.ID, e.TempDir, e.Client); err != nil {
		return err
	}

	config.Cmd = args
	klog.V(4).Infof("Running %#v inside of %s as user %s", config.Cmd, e.Container.ID, config.User)
	exec, err := e.Client.CreateExec(docker.CreateExecOptions{
		Cmd:          config.Cmd,
		Container:    e.Container.ID,
		AttachStdout: true,
		AttachStderr: true,
		User:         config.User,
	})
	if err != nil {
		return err
	}
	if err := e.Client.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: e.Out,
		ErrorStream:  e.ErrOut,
	}); err != nil {
		return err
	}
	status, err := e.Client.InspectExec(exec.ID)
	if err != nil {
		return err
	}
	if status.ExitCode != 0 {
		klog.V(4).Infof("Failed command (code %d): %v", status.ExitCode, args)
		return fmt.Errorf("running '%s' failed with exit code %d", strings.Join(run.Args, " "), status.ExitCode)
	}

	if err := e.Volumes.Restore(e.Container.ID, e.Client); err != nil {
		return err
	}

	return nil
}

// Copy implements the executor copy function.
func (e *ClientExecutor) Copy(excludes []string, copies ...imagebuilder.Copy) error {
	// copying content into a volume invalidates the archived state of any given directory
	for _, copy := range copies {
		e.Volumes.Invalidate(copy.Dest)
	}

	return e.CopyContainer(e.Container, excludes, copies...)
}

// CopyContainer copies the provided content into a destination container.
func (e *ClientExecutor) CopyContainer(container *docker.Container, excludes []string, copies ...imagebuilder.Copy) error {
	for _, c := range copies {
		// TODO: reuse source
		for _, src := range c.Src {
			klog.V(4).Infof("Archiving %s download=%t fromFS=%t from=%s", src, c.Download, c.FromFS, c.From)
			var r io.Reader
			var closer io.Closer
			var err error
			if len(c.From) > 0 {
				r, closer, err = e.archiveFromContainer(c.From, src, c.Dest)
			} else {
				r, closer, err = e.Archive(c.FromFS, src, c.Dest, c.Download, excludes)
			}
			if err != nil {
				return err
			}

			klog.V(5).Infof("Uploading to %s at %s", container.ID, c.Dest)
			if klog.V(6) {
				logArchiveOutput(r, "Archive file for %s")
			}
			err = e.Client.UploadToContainer(container.ID, docker.UploadToContainerOptions{
				InputStream: r,
				Path:        "/",
			})
			if err := closer.Close(); err != nil {
				klog.Errorf("Error while closing stream container copy stream %s: %v", container.ID, err)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type closers []func() error

func (c closers) Close() error {
	var lastErr error
	for _, fn := range c {
		if err := fn(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (e *ClientExecutor) archiveFromContainer(from string, src, dst string) (io.Reader, io.Closer, error) {
	var containerID string
	var containerConfig *docker.Config
	if other, ok := e.Named[from]; ok {
		if other.Container == nil {
			return nil, nil, fmt.Errorf("the stage %q has not been built yet", from)
		}
		klog.V(5).Infof("Using container %s as input for archive request", other.Container.ID)
		containerID = other.Container.ID
		containerConfig = other.Container.Config
	} else {
		klog.V(5).Infof("Creating a container temporarily for image input from %q in %s", from, src)
		_, err := e.LoadImage(from)
		if err != nil {
			return nil, nil, err
		}
		c, err := e.Client.CreateContainer(docker.CreateContainerOptions{
			Config: &docker.Config{
				Image: from,
			},
		})
		if err != nil {
			return nil, nil, err
		}
		containerID = c.ID
		containerConfig = c.Config
		e.Deferred = append([]func() error{func() error { return e.removeContainer(containerID) }}, e.Deferred...)
	}

	if !strings.HasPrefix(src, "/") {
		src = path.Join(containerConfig.WorkingDir, src)
	}

	check := newDirectoryCheck(e.Client, e.Container.ID)
	pr, pw := io.Pipe()
	ar, archiveRoot, err := archiveFromContainer(pr, src, dst, nil, check)
	if err != nil {
		pr.Close()
		pw.Close()
		return nil, nil, err
	}
	closer := newCloser(func() error {
		err2 := pr.Close()
		err3 := ar.Close()
		if err3 != nil {
			return err3
		}
		return err2
	})
	go func() {
		klog.V(6).Infof("Download from container %s at path %s", containerID, archiveRoot)
		err := e.Client.DownloadFromContainer(containerID, docker.DownloadFromContainerOptions{
			OutputStream: pw,
			Path:         archiveRoot,
		})
		pw.CloseWithError(err)
	}()
	return &readCloser{Reader: ar, Closer: closer}, pr, nil
}

func (e *ClientExecutor) Archive(fromFS bool, src, dst string, allowDownload bool, excludes []string) (io.Reader, io.Closer, error) {
	var check DirectoryCheck
	if e.Container != nil {
		check = newDirectoryCheck(e.Client, e.Container.ID)
	}
	if isURL(src) {
		if !allowDownload {
			return nil, nil, fmt.Errorf("source can't be a URL")
		}
		klog.V(5).Infof("Archiving %s -> %s from URL", src, dst)
		return archiveFromURL(src, dst, e.TempDir, check)
	}
	// the input is from the filesystem, use the source as the input
	if fromFS {
		klog.V(5).Infof("Archiving %s %s -> %s from a filesystem location", src, ".", dst)
		return archiveFromDisk(src, ".", dst, allowDownload, excludes, check)
	}
	// if the context is in archive form, read from it without decompressing
	if len(e.ContextArchive) > 0 {
		klog.V(5).Infof("Archiving %s %s -> %s from context archive", e.ContextArchive, src, dst)
		return archiveFromFile(e.ContextArchive, src, dst, excludes, check)
	}
	// if the source is an archive, extract it under the destination directory
	srcFullPath := filepath.Join(e.Directory, src)
	if allowDownload && isArchivePath(srcFullPath) {
		f, err := os.Open(srcFullPath)
		if err != nil {
			return nil, nil, fmt.Errorf("error opening %q: %v", srcFullPath, err)
		}
		klog.V(5).Infof("Extracting %s -> %s from local archive", srcFullPath, dst)
		transform := func(h *tar.Header, r io.Reader) (data []byte, update bool, skip bool, err error) {
			h.Name = filepath.Join(dst, h.Name)
			if h.Typeflag == tar.TypeLink {
				h.Linkname = filepath.Join(dst, h.Linkname)
			}
			return nil, false, false, nil
		}
		filtered, err := transformArchive(f, true, transform)
		if err != nil {
			f.Close()
			return nil, nil, fmt.Errorf("error transforming archive %q: %v", srcFullPath, err)
		}
		return filtered, f, nil
	}
	// if the context is a directory, we only allow relative includes
	klog.V(5).Infof("Archiving %q %q -> %q from disk", e.Directory, src, dst)
	return archiveFromDisk(e.Directory, src, dst, allowDownload, excludes, check)
}

// ContainerVolumeTracker manages tracking archives of specific paths inside a container.
type ContainerVolumeTracker struct {
	paths map[string]string
	errs  []error
}

func NewContainerVolumeTracker() *ContainerVolumeTracker {
	return &ContainerVolumeTracker{
		paths: make(map[string]string),
	}
}

// Empty returns true if the tracker is not watching any paths
func (t *ContainerVolumeTracker) Empty() bool {
	return t == nil || len(t.paths) == 0
}

// Add tracks path unless it already is being tracked.
func (t *ContainerVolumeTracker) Add(path string) {
	if _, ok := t.paths[path]; !ok {
		t.paths[path] = ""
	}
}

// Release removes any stored snapshots
func (t *ContainerVolumeTracker) Release() []error {
	if t == nil {
		return nil
	}
	for path := range t.paths {
		t.ReleasePath(path)
	}
	return t.errs
}

func (t *ContainerVolumeTracker) ReleasePath(path string) {
	if t == nil {
		return
	}
	if archivePath, ok := t.paths[path]; ok && len(archivePath) > 0 {
		err := os.Remove(archivePath)
		if err != nil && !os.IsNotExist(err) {
			t.errs = append(t.errs, err)
		}
		klog.V(5).Infof("Releasing path %s (%v)", path, err)
		t.paths[path] = ""
	}
}

func (t *ContainerVolumeTracker) Invalidate(path string) {
	if t == nil {
		return
	}
	set := imagebuilder.VolumeSet{}
	set.Add(path)
	for path := range t.paths {
		if set.Covers(path) {
			t.ReleasePath(path)
		}
	}
}

// Save ensures that all paths tracked underneath this container are archived or
// returns an error.
func (t *ContainerVolumeTracker) Save(containerID, tempDir string, client *docker.Client) error {
	if t == nil {
		return nil
	}
	set := imagebuilder.VolumeSet{}
	for dest := range t.paths {
		set.Add(dest)
	}
	// remove archive paths that are covered by other paths
	for dest := range t.paths {
		if !set.Has(dest) {
			t.ReleasePath(dest)
			delete(t.paths, dest)
		}
	}
	for dest, archivePath := range t.paths {
		if len(archivePath) > 0 {
			continue
		}
		archivePath, err := snapshotPath(dest, containerID, tempDir, client)
		if err != nil {
			return err
		}
		t.paths[dest] = archivePath
	}
	return nil
}

// filterTarPipe transforms a tar file as it is streamed, calling fn on each header in the file.
// If fn returns false, the file is skipped. If an error occurs it is returned.
func filterTarPipe(w *tar.Writer, r *tar.Reader, fn func(*tar.Header) bool) error {
	for {
		h, err := r.Next()
		if err != nil {
			return err
		}
		if fn(h) {
			if err := w.WriteHeader(h); err != nil {
				return err
			}
			if _, err := io.Copy(w, r); err != nil {
				return err
			}
		} else {
			if _, err := io.Copy(ioutil.Discard, r); err != nil {
				return err
			}
		}
	}
}

// snapshotPath preserves the contents of path in container containerID as a temporary
// archive, returning either an error or the path of the archived file.
func snapshotPath(path, containerID, tempDir string, client *docker.Client) (string, error) {
	f, err := ioutil.TempFile(tempDir, "archived-path")
	if err != nil {
		return "", err
	}
	klog.V(4).Infof("Snapshot %s for later use under %s", path, f.Name())

	r, w := io.Pipe()
	tr := tar.NewReader(r)
	tw := tar.NewWriter(f)
	go func() {
		err := filterTarPipe(tw, tr, func(h *tar.Header) bool {
			if i := strings.Index(h.Name, "/"); i != -1 {
				h.Name = h.Name[i+1:]
			}
			return len(h.Name) > 0
		})
		if err == nil || err == io.EOF {
			tw.Flush()
			w.Close()
			klog.V(5).Infof("Snapshot rewritten from %s", path)
			return
		}
		klog.V(5).Infof("Snapshot of %s failed: %v", path, err)
		w.CloseWithError(err)
	}()

	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	err = client.DownloadFromContainer(containerID, docker.DownloadFromContainerOptions{
		Path:         path,
		OutputStream: w,
	})
	f.Close()
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// Restore ensures the paths managed by t exactly match the container. This requires running
// exec as a user that can delete contents from the container. It will return an error if
// any client operation fails.
func (t *ContainerVolumeTracker) Restore(containerID string, client *docker.Client) error {
	if t == nil {
		return nil
	}
	for dest, archivePath := range t.paths {
		if len(archivePath) == 0 {
			return fmt.Errorf("path %s does not have an archive and cannot be restored", dest)
		}
		klog.V(4).Infof("Restoring contents of %s from %s", dest, archivePath)
		if !strings.HasSuffix(dest, "/") {
			dest = dest + "/"
		}
		exec, err := client.CreateExec(docker.CreateExecOptions{
			Container: containerID,
			Cmd:       []string{"/bin/sh", "-c", "rm -rf $@", "", dest + "*"},
			User:      "0",
		})
		if err != nil {
			return fmt.Errorf("unable to setup clearing preserved path %s: %v", dest, err)
		}
		if err := client.StartExec(exec.ID, docker.StartExecOptions{}); err != nil {
			return fmt.Errorf("unable to clear preserved path %s: %v", dest, err)
		}
		var status *docker.ExecInspect
		for status == nil {
			status, err = client.InspectExec(exec.ID)
			if err != nil {
				break
			}
			if !status.Running {
				break
			}
			status = nil
		}
		if err != nil {
			return fmt.Errorf("clearing preserved path %s did not succeed: %v", dest, err)
		}
		if status.ExitCode != 0 {
			return fmt.Errorf("clearing preserved path %s failed with exit code %d", dest, status.ExitCode)
		}
		err = func() error {
			f, err := os.Open(archivePath)
			if err != nil {
				return fmt.Errorf("unable to open archive %s for preserved path %s: %v", archivePath, dest, err)
			}
			defer f.Close()
			if err := client.UploadToContainer(containerID, docker.UploadToContainerOptions{
				InputStream: f,
				Path:        dest,
			}); err != nil {
				return fmt.Errorf("unable to upload preserved contents from %s to %s: %v", archivePath, dest, err)
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
