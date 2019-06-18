package conformance

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Buildah build conformance test", func() {
	var (
		tempdir     string
		err         error
		buildahtest BuildAhTest
	)

	skipKeyWords := []string{"Created",
		"Id",
		"RepoTags",
		"Parent",
		"Data",
		"Layers",
		"Container",
		"DockerVersion",
		"VirtualSize",
		"Size",
		"Image",
		"Hostname",
		"ContainerConfig:Cmd",
	}
	type BuildTest struct {
		Dockerfile      string
		BuildahRegex    string
		DockerRegex     string
		BuildahErrRegex string
		DockerErrRegex  string
		ExtraOptions    []string
		WithoutDocker   bool
		IsFile          bool
	}

	BeforeEach(func() {
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		buildahtest = BuildahCreate(tempdir)
	})

	AfterEach(func() {
		buildahtest.Cleanup()
		cleanup := buildahtest.Docker([]string{"rmi", "-f", "buildahimage"})
		cleanup.WaitWithDefaultTimeout()
		cleanup = buildahtest.Docker([]string{"rmi", "-f", "dockerimage"})
		cleanup.WaitWithDefaultTimeout()
	})

	DescribeTable("conformance with docker",
		func(test BuildTest) {
			dockerfilePath := filepath.Join(buildahtest.TestDataDir, test.Dockerfile)
			dst := buildahtest.TempDir
			buildDir := buildahtest.TempDir
			if test.IsFile {
				dst = filepath.Join(buildahtest.TempDir, "Dockerfile")
			}
			Expect(CopyFiles(dockerfilePath, dst)).Should(BeNil())

			buildahoptions := []string{"bud", "-t", "buildahimage", buildDir}
			dockeroptions := []string{"build", "-t", "docker.io/dockerimage", buildDir}
			if len(test.ExtraOptions) != 0 {
				for i := 0; i < len(test.ExtraOptions); i++ {
					test.ExtraOptions[i] = strings.Replace(test.ExtraOptions[i], "TEMPDIR",
						buildahtest.TempDir, -1)
				}
				buildahoptions = append([]string{"bud", "-t", "buildahimage"},
					append(test.ExtraOptions, buildDir)...)
				dockeroptions = append([]string{"build", "-t", "docker.io/dockerimage"},
					append(test.ExtraOptions, buildDir)...)
			}

			dockerfile, _ := ioutil.ReadFile(filepath.Join(buildDir, "Dockerfile"))
			errMsg := strings.Replace(ERR_MSG, "DOCKERFILECONTENT",
				strings.Replace(string(dockerfile), "\n", "\n\t", -1), -1)
			buildahcmds := strings.Join(append([]string{buildahtest.BuildAhBinary},
				buildahtest.MakeOptions(buildahoptions)...), " ")
			errMsg = strings.Replace(errMsg, "BUILDAHCMD", buildahcmds, -1)

			buildah := buildahtest.BuildAh(buildahoptions)
			buildah.WaitWithDefaultTimeout()
			Expect(buildah.ExitCode()).To(Equal(0),
				strings.Replace(errMsg, "FAILEDREASON", "Buildah bud failed.", -1))
			buildahoutput := buildah.OutputToString()
			buildaherr := buildah.ErrorToString()

			push := SystemExec("podman", []string{"push", "buildahimage", "docker-daemon:buildahimage:latest"})
			push.WaitWithDefaultTimeout()
			Expect(push.ExitCode()).To(Equal(0),
				strings.Replace(errMsg, "FAILEDREASON", "Failed to push image to docker daemon.", -1))

			// Commet ID check
			check := buildahtest.Docker([]string{"images", "-f", "reference=buildahimage", "-q"})
			check.WaitWithDefaultTimeout()
			Expect(buildahoutput).To(ContainSubstring(check.OutputToString()),
				strings.Replace(errMsg, "FAILEDREASON", "Image ID changed after push to docker daemon.", -1))

			// Image info check
			if test.WithoutDocker {
				br := regexp.MustCompile(test.BuildahRegex)
				Expect(br.MatchString(buildahoutput)).To(BeTrue(),
					strings.Replace(errMsg, "FAILEDREASON", "Buildah Output is not as expect.", -1))
				if test.BuildahErrRegex != "" {
					bre := regexp.MustCompile(test.BuildahErrRegex)
					Expect(bre.MatchString(buildaherr)).To(BeTrue(),
						strings.Replace(errMsg, "FAILEDREASON", "Buildah Output is not as expect", -1))
				}
			} else {
				docker := buildahtest.Docker(dockeroptions)
				docker.WaitWithDefaultTimeout()
				Expect(docker.ExitCode()).To(Equal(0),
					strings.Replace(errMsg, "FAILEDREASON", "Docker build failed.", -1))
				dockeroutput := docker.OutputToString()
				dockererr := docker.ErrorToString()

				if test.BuildahRegex != "" {
					br := regexp.MustCompile(test.BuildahRegex)
					dr := regexp.MustCompile(test.DockerRegex)
					buildahstrs := br.FindAllStringSubmatch(buildahoutput, -1)
					dockerstrs := dr.FindAllStringSubmatch(dockeroutput, -1)
					Expect(len(buildahstrs)).To(Equal(len(dockerstrs)),
						strings.Replace(errMsg, "FAILEDREASON", "Buildah output is different from docker.", -1))
					for i := 0; i < len(buildahstrs); i++ {
						Expect(buildahstrs[i][1]).To(Equal(dockerstrs[i][1]),
							strings.Replace(errMsg, "FAILEDREASON", "Buildah output is different from docker", -1))
					}
				}
				if test.BuildahErrRegex != "" {
					bre := regexp.MustCompile(test.BuildahErrRegex)
					dre := regexp.MustCompile(test.DockerErrRegex)
					buildaherrstrs := bre.FindAllStringSubmatch(buildaherr, -1)
					dockererrstrs := dre.FindAllStringSubmatch(dockererr, -1)
					Expect(len(buildaherrstrs)).To(Equal(len(dockererrstrs)),
						strings.Replace(errMsg, "FAILEDREASON", "Buildah output is different from docker.", -1))
					for i := 0; i < len(buildaherrstrs); i++ {
						Expect(buildaherrstrs[i][1]).To(Equal(dockererrstrs[i][1]),
							strings.Replace(errMsg, "FAILEDREASON", "Buildah output is different from docker", -1))
					}
				}

				buildahimage := buildahtest.Docker([]string{"inspect", "buildahimage:latest"})
				buildahimage.WaitWithDefaultTimeout()
				dockerimage := buildahtest.Docker([]string{"inspect", "dockerimage:latest"})
				dockerimage.WaitWithDefaultTimeout()
				miss, left, diff, same := CompareJSON(dockerimage.OutputToJSON()[0], buildahimage.OutputToJSON()[0], skipKeyWords)
				Expect(same).To(BeTrue(), strings.Replace(errMsg, "FAILEDREASON", InspectCompareResult(miss, left, diff), -1))

				//Check fs with container-diff
				fscheck := SystemExec("container-diff", []string{"diff", "daemon://buildahimage", "daemon://dockerimage", "--type=file"})
				fscheck.WaitWithDefaultTimeout()
				fsr := regexp.MustCompile(`/[a-zA-Z0-9/.\-_]+[ ]+[0-9]+`)
				fsrignore := regexp.MustCompile(`/[a-zA-Z0-9/.\-_]+[ ]+0`)
				Expect(len(fsr.FindAllString(fscheck.OutputToString(),
					-1))).To(Equal(len(fsrignore.FindAllString(fscheck.OutputToString(), -1))),
					strings.Replace(errMsg, "FAILEDREASON", "Files inside image is different.", -1))
			}

		},
		Entry("shell test", BuildTest{
			Dockerfile:   "Dockerfile.shell",
			BuildahRegex: "(?s)[0-9a-z]+(.*)--",
			DockerRegex:  "(?s)RUN env.*?Running in [0-9a-z]+(.*?)---",
			IsFile:       true,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("copy file to root", BuildTest{
			Dockerfile:    "Dockerfile.copyfrom_1",
			BuildahRegex:  "[-rw]+.*?/a",
			WithoutDocker: true,
			IsFile:        true,
		}),

		Entry("copy file to same file", BuildTest{
			Dockerfile:    "Dockerfile.copyfrom_2",
			BuildahRegex:  "[-rw]+.*?/a",
			WithoutDocker: true,
			IsFile:        true,
		}),

		Entry("copy file to workdir", BuildTest{
			Dockerfile:    "Dockerfile.copyfrom_3",
			BuildahRegex:  "[-rw]+.*?/b/a",
			WithoutDocker: true,
			IsFile:        true,
		}),

		Entry("copy file to workdir rename", BuildTest{
			Dockerfile:    "Dockerfile.copyfrom_3_1",
			BuildahRegex:  "[-rw]+.*?/b/b",
			WithoutDocker: true,
			IsFile:        true,
		}),

		Entry("copy folder contents to higher level", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_4",
			BuildahRegex:    "(?s)[-rw]+.*?/b/1.*?[-rw]+.*?/b/2.*?/b.*?[-rw]+.*?1.*?[-rw]+.*?2",
			BuildahErrRegex: "/a: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy wildcard folder contents to higher level", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_5",
			BuildahRegex:    "(?s)[-rw]+.*?/b/1.*?[-rw]+.*?/b/2.*?/b.*?[-rw]+.*?1.*?[-rw]+.*?2",
			BuildahErrRegex: "(?s)/a: No such file or directory.*?/b/a: No such file or directory.*?/b/b: No such file or director",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy folder with dot contents to higher level", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_6",
			BuildahRegex:    "(?s)[-rw]+.*?/b/1.*?[-rw]+.*?/b/2.*?/b.*?[-rw]+.*?1.*?[-rw]+.*?2",
			BuildahErrRegex: "(?s)/a: No such file or directory.*?/b/a: No such file or directory.*?/b/b: No such file or director",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy root file to different root name", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_7",
			BuildahRegex:    "[-rw]+.*?/a",
			BuildahErrRegex: "/b: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy nested file to different root name", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_8",
			BuildahRegex:    "[-rw]+.*?/a",
			BuildahErrRegex: "/b: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy file to deeper directory with explicit slash", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_9",
			BuildahRegex:    "[-rw]+.*?/a/b/c/1",
			BuildahErrRegex: "/a/b/1: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy file to deeper directory without explicit slash", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_10",
			BuildahRegex:    "[-rw]+.*?/a/b/c",
			BuildahErrRegex: "/a/b/1: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy directory to deeper directory without explicit slash", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_11",
			BuildahRegex:    "[-rw]+.*?/a/b/c/1",
			BuildahErrRegex: "/a/b/1: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy directory to root without explicit slash", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_12",
			BuildahRegex:    "[-rw]+.*?/a/1",
			BuildahErrRegex: "/a/a: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("copy directory trailing to root without explicit slash", BuildTest{
			Dockerfile:      "Dockerfile.copyfrom_13",
			BuildahRegex:    "[-rw]+.*?/a/1",
			BuildahErrRegex: "/a/a: No such file or directory",
			WithoutDocker:   true,
			IsFile:          true,
		}),

		Entry("multi stage base", BuildTest{
			Dockerfile:    "Dockerfile.reusebase",
			BuildahRegex:  "[0-9a-z]+ /1",
			WithoutDocker: true,
			IsFile:        true,
		}),

		Entry("Directory", BuildTest{
			Dockerfile:   "dir",
			IsFile:       false,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("copy to dir", BuildTest{
			Dockerfile:   "copy",
			IsFile:       false,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("copy dir", BuildTest{
			Dockerfile:   "copydir",
			IsFile:       false,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("copy to renamed file", BuildTest{
			Dockerfile:   "copyrename",
			IsFile:       false,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("directory with slash", BuildTest{
			Dockerfile:   "overlapdirwithslash",
			IsFile:       false,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("directory without slash", BuildTest{
			Dockerfile:   "overlapdirwithoutslash",
			IsFile:       false,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("environment", BuildTest{
			Dockerfile:   "Dockerfile.env",
			IsFile:       true,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("edgecases", BuildTest{
			Dockerfile: "Dockerfile.edgecases",
			IsFile:     true,
		}),

		Entry("exposed default", BuildTest{
			Dockerfile:   "Dockerfile.exposedefault",
			IsFile:       true,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("add", BuildTest{
			Dockerfile:   "Dockerfile.add",
			IsFile:       true,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("run with JSON", BuildTest{
			Dockerfile:   "Dockerfile.run.args",
			BuildahRegex: "(first|third|fifth|inner) (second|fourth|sixth|outer)",
			DockerRegex:  "Running in [0-9a-z]+.*?(first|third|fifth|inner) (second|fourth|sixth|outer)",
			IsFile:       true,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("shell", BuildTest{
			Dockerfile:   "Dockerfile.shell",
			IsFile:       true,
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("wildcard", BuildTest{
			Dockerfile:   "wildcard",
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("volume", BuildTest{
			Dockerfile:   "volume",
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("volumerun", BuildTest{
			Dockerfile:   "volumerun",
			ExtraOptions: []string{"--no-cache"},
		}),

		Entry("mount", BuildTest{
			Dockerfile:    "mount",
			BuildahRegex:  "/tmp/test/file.*?regular file.*?/tmp/test/file2.*?regular file",
			WithoutDocker: true,
			ExtraOptions:  []string{"--no-cache", "-v", "TEMPDIR:/tmp/test"},
		}),

		Entry("Transient mount", BuildTest{
			Dockerfile:    "transientmount",
			BuildahRegex:  "file2.*?FROM busybox ENV name value",
			WithoutDocker: true,
			ExtraOptions:  []string{"--no-cache", "-v", "TEMPDIR:/mountdir", "-v", "TEMPDIR/Dockerfile.env:/mountfile"},
		}),
	)
})
