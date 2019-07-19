package conformance

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	defaultWaitTimeout = 240
	GLOBALOPTIONS      = []string{"--root",
		"--runroot",
		"--registries-conf",
		"--registries-conf-dir",
		"--storage-driver",
		"--storage-opt",
		"--userns-uid-map",
		"--userns-gid-map",
	}
	BUILDAH_SUBCMD = []string{"add",
		"bud",
		"commit",
		"config",
		"containers",
		"copy",
		"from",
		"images",
		"inspect",
		"mount",
		"push",
		"rm",
		"rmi",
		"run",
		"tag",
		"umount",
		"unshare",
	}
	ERR_MSG = `The test Dockerfile:
	DOCKERFILECONTENT
The Buildah bud command:
	BUILDAHCMD
Test failed reason:
	FAILEDREASON`
)

// BuildahTestSession wraps the gexec.session so we can extend it
type BuildahTestSession struct {
	*gexec.Session
}

// BuildAhTest struct for command line options
type BuildAhTest struct {
	BuildAhBinary     string
	DockerBinary      string
	TempDir           string
	TestDataDir       string
	GlobalOptions     map[string]string
	BuildahCmdOptions map[string][]string
}

// TestBuildAh ginkgo master function
// func TestBuildAh(t *testing.T) {
func TestConformance(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buildah Conformance Test Suite")
}

var _ = BeforeSuite(func() {
	// Check docker installed and started
	session := SystemExec("docker", []string{"pull", "busybox"})
	session.WaitWithDefaultTimeout()
	if session.ExitCode() != 0 {
		fmt.Printf("This test suite request docker running. Please check your env.")
		os.Exit(1)
	}

	session = SystemExec("podman", []string{"images"})
	session.WaitWithDefaultTimeout()
	if session.ExitCode() != 0 {
		fmt.Printf("This test suite request podman installed. Please check your env.")
		os.Exit(1)
	}

	session = SystemExec("container-diff", []string{"analyze", "daemon://busybox", "--type=history"})
	session.WaitWithDefaultTimeout()
	if session.ExitCode() != 0 {
		fmt.Printf("This test suite request container-diff installed. Please check your env.")
		os.Exit(1)
	}
})

// CreateTempDirin
func CreateTempDirInTempDir() (string, error) {
	return ioutil.TempDir("", "buildah_test")
}

// Check if the key is seted in Env
func envSeted(key string) bool {
	_, set := os.LookupEnv(key)
	return set
}

// BuildahCreate a BuildAhTest instance for the tests
func BuildahCreate(tempDir string) BuildAhTest {
	var option string
	var envKey string
	buildahCmdOptions := make(map[string][]string)
	globalOptions := make(map[string]string)
	cwd, _ := os.Getwd()

	for _, n := range GLOBALOPTIONS {
		option = strings.Replace(strings.Title(strings.Trim(n, " ")), " ", "", -1)
		envKey = strings.Replace(strings.ToUpper(strings.Trim(n, "-")), "-", "_", -1)
		if envSeted(envKey) {
			globalOptions[option] = os.Getenv(envKey)
		}
	}

	for _, n := range BUILDAH_SUBCMD {
		envKey = strings.Replace("BUILDAH_SUBCMD_OPTIONS", "SUBCMD", strings.ToUpper(n), -1)
		if envSeted(envKey) {
			buildahCmdOptions[n] = strings.Split(os.Getenv(envKey), " ")
		}
	}

	buildAhBinary := filepath.Join(cwd, "../../buildah")
	if envSeted("BUILDAH_BINARY") {
		buildAhBinary = os.Getenv("BUILDAH_BINARY")
	}
	testDataDir := filepath.Join(cwd, "testdata")
	if os.Getenv("TEST_DATA_DIR") != "" {
		testDataDir = os.Getenv("TEST_DATA_DIR")
	}
	dockerBinary := "docker"
	if os.Getenv("DOCKER_BINARY") != "" {
		dockerBinary = os.Getenv("DOCKER_BINARY")
	}

	return BuildAhTest{
		BuildAhBinary:     buildAhBinary,
		DockerBinary:      dockerBinary,
		TempDir:           tempDir,
		TestDataDir:       testDataDir,
		GlobalOptions:     globalOptions,
		BuildahCmdOptions: buildahCmdOptions,
	}
}

//MakeOptions assembles all the buildah options
func (p *BuildAhTest) MakeOptions(args []string) []string {
	var addOptions, subArgs []string
	var option string
	for _, n := range GLOBALOPTIONS {
		option = strings.Replace(strings.Title(strings.Trim(n, " ")), " ", "", -1)
		if p.GlobalOptions[option] != "" {
			addOptions = append(addOptions, n, p.GlobalOptions[option])
		}
	}

	subCmd := args[0]
	addOptions = append(addOptions, subCmd)
	if subCmd == "build-using-dockerfile" {
		subCmd = "bud"
	}
	if subCmd == "unmount" {
		subCmd = "umount"
	}
	if subCmd == "delete" {
		subCmd = "rm"
	}

	if _, ok := p.BuildahCmdOptions[subCmd]; ok {
		m := make(map[string]bool)
		subArgs = p.BuildahCmdOptions[subCmd]
		for i := 0; i < len(subArgs); i++ {
			m[subArgs[i]] = true
		}
		for i := 1; i < len(args); i++ {
			if _, ok := m[args[i]]; !ok {
				subArgs = append(subArgs, args[i])
			}
		}
	} else {
		subArgs = args[1:]
	}

	addOptions = append(addOptions, subArgs...)

	return addOptions
}

// BuildAh is the exec call to buildah on the filesystem
func (p *BuildAhTest) BuildAh(args []string) *BuildahTestSession {
	if len(args) == 0 {
		Fail(fmt.Sprintf("Need give a subcommand or -v to buildah command line"))
	}

	buildAhOptions := p.MakeOptions(args)

	fmt.Printf("Running: %s %s\n", p.BuildAhBinary, strings.Join(buildAhOptions, " "))
	command := exec.Command(p.BuildAhBinary, buildAhOptions...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	if err != nil {
		Fail(fmt.Sprintf("unable to run buildah command: %s", strings.Join(buildAhOptions, " ")))
	}
	return &BuildahTestSession{session}
}

// BuildAh is the exec call to buildah on the filesystem
func (p *BuildAhTest) Docker(args []string) *BuildahTestSession {
	dockerOptions := args
	fmt.Printf("Running: %s %s\n", p.DockerBinary, strings.Join(dockerOptions, " "))
	command := exec.Command(p.DockerBinary, dockerOptions...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	if err != nil {
		Fail(fmt.Sprintf("unable to run docker command: %s", strings.Join(dockerOptions, " ")))
	}
	return &BuildahTestSession{session}
}

// Cleanup cleans up the temporary store
func (p *BuildAhTest) Cleanup() {
	// Nuke tempdir
	if err := os.RemoveAll(p.TempDir); err != nil {
		fmt.Printf("%q\n", err)
	}
	cleanup := p.BuildAh([]string{"rmi", "-a", "-f"})
	cleanup.WaitWithDefaultTimeout()
}

// GrepString takes session output and behaves like grep. it returns a bool
// if successful and an array of strings on positive matches
func (s *BuildahTestSession) GrepString(term string) (bool, []string) {
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
func (s *BuildahTestSession) OutputToString() string {
	fields := strings.Fields(fmt.Sprintf("%s", s.Out.Contents()))
	return strings.Join(fields, " ")
}

// ErrorToString formats session output to string
func (s *BuildahTestSession) ErrorToString() string {
	fields := strings.Fields(fmt.Sprintf("%s", s.Err.Contents()))
	return strings.Join(fields, " ")
}

// OutputToStringArray returns the output as a []string
// where each array item is a line split by newline
func (s *BuildahTestSession) OutputToStringArray() []string {
	output := fmt.Sprintf("%s", s.Out.Contents())
	return strings.Split(output, "\n")
}

// IsJSONOutputValid attempts to unmarshall the session buffer
// and if successful, returns true, else false
func (s *BuildahTestSession) IsJSONOutputValid() bool {
	var i interface{}
	if err := json.Unmarshal(s.Out.Contents(), &i); err != nil {
		fmt.Println(err)
		return false
	}
	return true
}

// OutputToJSON returns the output as a map
func (s *BuildahTestSession) OutputToJSON() []map[string]interface{} {
	var i []map[string]interface{}
	if err := json.Unmarshal(s.Out.Contents(), &i); err != nil {
		fmt.Println(err)
		return nil
	}
	return i
}

// contains is used to check if item is exist in []string or not
func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

// AddPrefix add given prefix to each string in []string.
// The prefix and the string is joined with ":"
func addPrefix(a []string, prefix string) []string {
	var b []string
	for i := 0; i < len(a); i++ {
		b = append(b, strings.Join([]string{prefix, a[i]}, ":"))
	}
	return b
}

// CompareJSON compare two JSON output from command line.
// It returns the keys are different as debug information and a bool for compare results.
func CompareJSON(a, b map[string]interface{}, skip []string) ([]string, []string, []string, bool) {
	var missKeys []string
	var leftKeys []string
	var diffKeys []string
	isSame := true

	for k, v := range a {
		vb, ok := b[k]
		if ok {
			delete(b, k)
		}
		if contains(skip, k) {
			continue
		}
		if !ok {
			missKeys = append(missKeys, k)
			isSame = false
			continue
		}
		if reflect.TypeOf(v) != reflect.TypeOf(vb) {
			if reflect.TypeOf(v) == nil && reflect.ValueOf(vb).Len() == 0 {
				continue
			}
			if reflect.TypeOf(vb) == nil && reflect.ValueOf(v).Len() == 0 {
				continue
			}
			diffKeys = append(diffKeys, diffDebug(k, v, vb))
			isSame = false
			continue
		}
		switch v.(type) {
		case map[string]interface{}:
			nextSkip := skip
			for _, s := range skip {
				if strings.Contains(s, ":") {
					tmp := strings.Split(s, ":")
					if tmp[0] == k {
						nextSkip = append(nextSkip, strings.Join(tmp[1:], ":"))
					}
				}
			}
			submiss, subleft, subdiff, ok := CompareJSON(v.(map[string]interface{}),
				vb.(map[string]interface{}), nextSkip)
			missKeys = append(missKeys, addPrefix(submiss, k)...)
			leftKeys = append(leftKeys, addPrefix(subleft, k)...)
			diffKeys = append(diffKeys, addPrefix(subdiff, k)...)
			if !ok {
				isSame = false
			}
		case []interface{}:
			tmpa := v.([]interface{})
			tmpb := vb.([]interface{})
			if len(tmpa) != len(tmpb) {
				diffKeys = append(diffKeys, diffDebug(k, v, vb))
				isSame = false
				break
			}
			m := make(map[interface{}]bool)
			for i := 0; i < len(tmpb); i++ {
				m[tmpb[i]] = true
			}

			for i := 0; i < len(tmpa); i++ {
				if _, ok := m[tmpa[i]]; !ok {
					diffKeys = append(diffKeys, diffDebug(k, v, vb))
					isSame = false
					break
				}
			}
		default:
			if !reflect.DeepEqual(v, vb) {
				diffKeys = append(diffKeys, diffDebug(k, v, vb))
				isSame = false
			}
		}
	}

	if len(b) > 0 {
		for k := range b {
			leftKeys = append(leftKeys, k)
		}
	}

	return missKeys, leftKeys, diffKeys, isSame
}

// diffDebug add field and values as debug information
func diffDebug(k string, a, b interface{}) string {
	return fmt.Sprintf("%v\t\t%v\t\t%v\n", k, a, b)
}

// InspectCompareResult give the compare results from inspect.
func InspectCompareResult(miss, left, diff []string) string {
	msg := `Inspect Error Messages:
	Item missing in buildah output: MISSKEYS
	Item only exist in buildah output: LEFTKEYS
	Item have different value:
	Field name			Value a		Value b
	DIFFKEYS`

	msg = strings.Replace(msg, "MISSKEYS", strings.Join(miss, " "), -1)
	msg = strings.Replace(msg, "LEFTKEYS", strings.Join(left, " "), -1)
	msg = strings.Replace(msg, "DIFFKEYS", strings.Join(diff, "\t"), -1)

	return msg
}

// WaitWithDefaultTimeout wait for command exit with defaultWaitTimeout.
func (s *BuildahTestSession) WaitWithDefaultTimeout() {
	s.Wait(defaultWaitTimeout)
}

// LineInOutputStartsWith returns true if a line in a
// session output starts with the supplied string
func (s *BuildahTestSession) LineInOuputStartsWith(term string) bool {
	for _, i := range s.OutputToStringArray() {
		if strings.HasPrefix(i, term) {
			return true
		}
	}
	return false
}

// LineInOutputContains returns true if a line in a
// session output starts with the supplied string
func (s *BuildahTestSession) LineInOuputContains(term string) bool {
	for _, i := range s.OutputToStringArray() {
		if strings.Contains(i, term) {
			return true
		}
	}
	return false
}

// SystemExec is used to exec a system command to check its exit code or output
func SystemExec(command string, args []string) *BuildahTestSession {
	fmt.Printf("Running: %s %s\n", command, strings.Join(args, " "))
	c := exec.Command(command, args...)
	session, err := gexec.Start(c, GinkgoWriter, GinkgoWriter)
	if err != nil {
		Fail(fmt.Sprintf("unable to run command: %s %s", command, strings.Join(args, " ")))
	}
	return &BuildahTestSession{session}
}

// CopyFile copy file from src to dst. If the file already exist then replace it.
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

// CopyFiles copy file or dir from src to dst.
// If the src is a file, please make sure the dst is a file too.
func CopyFiles(src, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if !srcinfo.IsDir() {
		err = CopyFile(src, dst)
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := filepath.Join(src, fd.Name())
		dstfp := filepath.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = CopyFiles(srcfp, dstfp); err != nil {
				return err
			}
		} else {
			if err = CopyFile(srcfp, dstfp); err != nil {
				return err
			}
		}
	}
	return nil
}
