package integration

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Podman load", func() {
	var (
		tempdir     string
		err         error
		buildahtest BuildAhTest
	)

	BeforeEach(func() {
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		buildahtest = BuildahCreate(tempdir)
	})

	AfterEach(func() {
		buildahtest.Cleanup()
	})

	It("buildah inspect json", func() {
		b := buildahtest.BuildAh([]string{"from", "--pull=false", "scratch"})
		b.WaitWithDefaultTimeout()
		Expect(b.ExitCode()).To(Equal(0))
		cid := b.OutputToString()
		result := buildahtest.BuildAh([]string{"inspect", cid})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(result.IsJSONOutputValid()).To(BeTrue())
	})

	It("buildah inspect format", func() {
		b := buildahtest.BuildAh([]string{"from", "--pull=false", "scratch"})
		b.WaitWithDefaultTimeout()
		Expect(b.ExitCode()).To(Equal(0))
		cid := b.OutputToString()
		result := buildahtest.BuildAh([]string{"inspect", "--format", "\"{{.}}\"", cid})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
	})

	It("buildah inspect image", func() {
		b := buildahtest.BuildAh([]string{"from", "--pull=false", "scratch"})
		b.WaitWithDefaultTimeout()
		Expect(b.ExitCode()).To(Equal(0))
		cid := b.OutputToString()
		commit := buildahtest.BuildAh([]string{"commit", cid, "scratchy-image"})
		commit.WaitWithDefaultTimeout()
		Expect(commit.ExitCode()).To(Equal(0))

		result := buildahtest.BuildAh([]string{"inspect", "--type", "image", "scratchy-image"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(result.IsJSONOutputValid()).To(BeTrue())

		result = buildahtest.BuildAh([]string{"inspect", "--type", "image", "scratchy-image:latest"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))
		Expect(result.IsJSONOutputValid()).To(BeTrue())
	})

	It("buildah HTML escaped", func() {
		b := buildahtest.BuildAh([]string{"from", "--pull=false", "scratch"})
		b.WaitWithDefaultTimeout()
		Expect(b.ExitCode()).To(Equal(0))
		cid := b.OutputToString()

		config := buildahtest.BuildAh([]string{"config", "--label", "maintainer=\"Darth Vader <dvader@darkside.io>\"", cid})
		config.WaitWithDefaultTimeout()
		Expect(config.ExitCode()).To(Equal(0))

		commit := buildahtest.BuildAh([]string{"commit", cid, "darkside-image"})
		commit.WaitWithDefaultTimeout()
		Expect(commit.ExitCode()).To(Equal(0))

		result := buildahtest.BuildAh([]string{"inspect", "--type", "image", "darkside-image"})
		result.WaitWithDefaultTimeout()
		Expect(result.ExitCode()).To(Equal(0))

		data := result.InspectImageJSON()
		Expect(data.Docker.Config.Labels["maintainer"]).To(Equal("\"Darth Vader <dvader@darkside.io>\""))

	})
})
