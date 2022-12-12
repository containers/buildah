package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containers/buildah"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/directory"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/storage"
	"github.com/containers/image/v5/types"
	sstorage "github.com/containers/storage"
	"github.com/containers/storage/pkg/reexec"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	artifactDir = "/tmp/.artifacts"
)

var (
	integrationRoot    string
	cacheImages        = []string{"alpine", "busybox", "quay.io/libpod/fedora-minimal:34"}
	restoreImages      = []string{"alpine", "busybox"}
	defaultWaitTimeout = 90
)

// BuildAhSession wraps the gexec.session so we can extend it
type BuildAhSession struct {
	*gexec.Session
}

// BuildAhTest struct for command line options
type BuildAhTest struct {
	BuildAhBinary  string
	RunRoot        string
	StorageOptions string
	ArtifactPath   string
	TempDir        string
	SignaturePath  string
	Root           string
	RegistriesConf string
}

// TestBuildAh ginkgo master function
func TestBuildAh(t *testing.T) {
	if reexec.Init() {
		os.Exit(1)
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buildah Suite")
}

var _ = BeforeSuite(func() {
	//Cache images
	cwd, _ := os.Getwd()
	integrationRoot = filepath.Join(cwd, "../../")
	buildah := BuildahCreate("/tmp")
	buildah.ArtifactPath = artifactDir
	if _, err := os.Stat(artifactDir); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(artifactDir, 0777); err != nil {
			fmt.Printf("%q\n", err)
			os.Exit(1)
		}
	}
	for _, image := range cacheImages {
		fmt.Printf("Caching %s...\n", image)
		if err := buildah.CreateArtifact(image); err != nil {
			fmt.Printf("%q\n", err)
			os.Exit(1)
		}
	}

})

// CreateTempDirin
func CreateTempDirInTempDir() (string, error) {
	return os.MkdirTemp("", "buildah_test")
}

// BuildahCreate a BuildAhTest instance for the tests
func BuildahCreate(tempDir string) BuildAhTest {
	cwd, _ := os.Getwd()

	buildAhBinary := filepath.Join(cwd, "../../bin/buildah")
	if os.Getenv("BUILDAH_BINARY") != "" {
		buildAhBinary = os.Getenv("BUILDAH_BINARY")
	}
	storageOpts := "--storage-driver vfs"
	if os.Getenv("STORAGE_DRIVER") != "" {
		storageOpts = fmt.Sprintf("--storage-driver %s", os.Getenv("STORAGE_DRIVER"))
	}

	return BuildAhTest{
		BuildAhBinary:  buildAhBinary,
		RunRoot:        filepath.Join(tempDir, "runroot"),
		Root:           filepath.Join(tempDir, "root"),
		StorageOptions: storageOpts,
		ArtifactPath:   artifactDir,
		TempDir:        tempDir,
		SignaturePath:  "../../tests/policy.json",
		RegistriesConf: "../../tests/registries.conf",
	}
}

//MakeOptions assembles all the buildah main options
func (p *BuildAhTest) MakeOptions() []string {
	return strings.Split(fmt.Sprintf("--root %s --runroot %s --registries-conf %s",
		p.Root, p.RunRoot, p.RegistriesConf), " ")
}

// BuildAh is the exec call to buildah on the filesystem
func (p *BuildAhTest) BuildAh(args []string) *BuildAhSession {
	buildAhOptions := p.MakeOptions()
	buildAhOptions = append(buildAhOptions, strings.Split(p.StorageOptions, " ")...)
	buildAhOptions = append(buildAhOptions, args...)
	fmt.Printf("Running: %s %s\n", p.BuildAhBinary, strings.Join(buildAhOptions, " "))
	command := exec.Command(p.BuildAhBinary, buildAhOptions...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	if err != nil {
		Fail(fmt.Sprintf("unable to run buildah command: %s", strings.Join(buildAhOptions, " ")))
	}
	return &BuildAhSession{session}
}

// Cleanup cleans up the temporary store
func (p *BuildAhTest) Cleanup() {
	// Nuke tempdir
	if err := os.RemoveAll(p.TempDir); err != nil {
		fmt.Printf("%q\n", err)
	}
}

// GrepString takes session output and behaves like grep. it returns a bool
// if successful and an array of strings on positive matches
func (s *BuildAhSession) GrepString(term string) (bool, []string) {
	var (
		greps   []string
		matches bool
	)

	for _, line := range strings.Split(s.OutputToString(), "\n") {
		if strings.Contains(line, term) {
			matches = true
			greps = append(greps, line)
		}
	}
	return matches, greps
}

// OutputToString formats session output to string
func (s *BuildAhSession) OutputToString() string {
	fields := bytes.Fields(s.Out.Contents())
	return string(bytes.Join(fields, []byte{' '}))
}

// OutputToStringArray returns the output as a []string
// where each array item is a line split by newline
func (s *BuildAhSession) OutputToStringArray() []string {
	return strings.Split(string(s.Out.Contents()), "\n")
}

// IsJSONOutputValid attempts to unmarshall the session buffer
// and if successful, returns true, else false
func (s *BuildAhSession) IsJSONOutputValid() bool {
	var i interface{}
	if err := json.Unmarshal(s.Out.Contents(), &i); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

func (s *BuildAhSession) WaitWithDefaultTimeout() {
	s.Wait(defaultWaitTimeout)
}

// SystemExec is used to exec a system command to check its exit code or output
func (p *BuildAhTest) SystemExec(command string, args []string) *BuildAhSession {
	c := exec.Command(command, args...)
	session, err := gexec.Start(c, GinkgoWriter, GinkgoWriter)
	if err != nil {
		Fail(fmt.Sprintf("unable to run command: %s %s", command, strings.Join(args, " ")))
	}
	return &BuildAhSession{session}
}

// CreateArtifact creates a cached image in the artifact dir
func (p *BuildAhTest) CreateArtifact(image string) error {
	systemContext := types.SystemContext{
		SignaturePolicyPath: p.SignaturePath,
	}
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return fmt.Errorf("loading signature policy: %w", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("loading signature policy: %w", err)
	}
	defer func() {
		_ = policyContext.Destroy()
	}()
	options := &copy.Options{}

	importRef, err := docker.ParseReference("//" + image)
	if err != nil {
		return fmt.Errorf("parsing image name %v: %w", image, err)
	}

	imageDir := strings.Replace(image, "/", "_", -1)
	exportDir := filepath.Join(p.ArtifactPath, imageDir)
	exportRef, err := directory.NewReference(exportDir)
	if err != nil {
		return fmt.Errorf("creating image reference for %v: %w", exportDir, err)
	}

	_, err = copy.Image(context.Background(), policyContext, exportRef, importRef, options)
	return err
}

// RestoreArtifact puts the cached image into our test store
func (p *BuildAhTest) RestoreArtifact(image string) error {
	storeOptions, _ := sstorage.DefaultStoreOptions(false, 0)
	storeOptions.GraphDriverName = os.Getenv("STORAGE_DRIVER")
	if storeOptions.GraphDriverName == "" {
		storeOptions.GraphDriverName = "vfs"
	}
	storeOptions.GraphRoot = p.Root
	storeOptions.RunRoot = p.RunRoot
	store, err := sstorage.GetStore(storeOptions)

	options := &copy.Options{}
	if err != nil {
		return fmt.Errorf("opening storage: %w", err)
	}
	defer func() {
		_, _ = store.Shutdown(false)
	}()

	storage.Transport.SetStore(store)
	ref, err := storage.Transport.ParseStoreReference(store, image)
	if err != nil {
		return fmt.Errorf("parsing image name: %w", err)
	}

	imageDir := strings.Replace(image, "/", "_", -1)
	importDir := filepath.Join(p.ArtifactPath, imageDir)
	importRef, err := directory.NewReference(importDir)
	if err != nil {
		return fmt.Errorf("creating image reference for %v: %w", image, err)
	}
	systemContext := types.SystemContext{
		SignaturePolicyPath: p.SignaturePath,
	}
	policy, err := signature.DefaultPolicy(&systemContext)
	if err != nil {
		return fmt.Errorf("loading signature policy: %w", err)
	}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return fmt.Errorf("loading signature policy: %w", err)
	}
	defer func() {
		_ = policyContext.Destroy()
	}()
	_, err = copy.Image(context.Background(), policyContext, ref, importRef, options)
	if err != nil {
		return fmt.Errorf("importing %s: %w", importDir, err)
	}
	return nil
}

// RestoreAllArtifacts unpacks all cached images
func (p *BuildAhTest) RestoreAllArtifacts() error {
	for _, image := range restoreImages {
		if err := p.RestoreArtifact(image); err != nil {
			return err
		}
	}
	return nil
}

//LineInOutputStartsWith returns true if a line in a
// session output starts with the supplied string
func (s *BuildAhSession) LineInOutputStartsWith(term string) bool {
	for _, i := range s.OutputToStringArray() {
		if strings.HasPrefix(i, term) {
			return true
		}
	}
	return false
}

//LineInOutputContains returns true if a line in a
// session output starts with the supplied string
func (s *BuildAhSession) LineInOutputContains(term string) bool {
	for _, i := range s.OutputToStringArray() {
		if strings.Contains(i, term) {
			return true
		}
	}
	return false
}

// InspectContainerToJSON takes the session output of an inspect
// container and returns json
func (s *BuildAhSession) InspectImageJSON() buildah.BuilderInfo {
	var i buildah.BuilderInfo
	err := json.Unmarshal(s.Out.Contents(), &i)
	Expect(err).To(BeNil())
	return i
}
