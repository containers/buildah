package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/buildah/imagebuildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type budResults struct {
	*buildahcli.LayerResults
	*buildahcli.BudResults
	*buildahcli.UserNSResults
	*buildahcli.FromAndBudResults
	*buildahcli.NameSpaceResults
}

func init() {
	budDescription := `
  Builds an OCI image using instructions in one or more Dockerfiles.

  If no arguments are specified, Buildah will use the current working directory
  as the build context and look for a Dockerfile. The build fails if no
  Dockerfile is present.`

	layerFlagsResults := buildahcli.LayerResults{}
	budFlagResults := buildahcli.BudResults{}
	fromAndBudResults := buildahcli.FromAndBudResults{}
	userNSResults := buildahcli.UserNSResults{}
	namespaceResults := buildahcli.NameSpaceResults{}

	budCommand := &cobra.Command{
		Use:     "build-using-dockerfile",
		Aliases: []string{"bud"},
		Short:   "Build an image using instructions in a Dockerfile",
		Long:    budDescription,
		//Flags:                  sortFlags(append(append(buildahcli.BudFlags, buildahcli.LayerFlags...), buildahcli.FromAndBudFlags...)),
		RunE: func(cmd *cobra.Command, args []string) error {
			br := budResults{
				&layerFlagsResults,
				&budFlagResults,
				&userNSResults,
				&fromAndBudResults,
				&namespaceResults,
			}
			return budCmd(cmd, args, br)
		},
		Example: `buildah bud
  buildah bud -f Dockerfile.simple .
  buildah bud --volume /home/test:/myvol:ro,Z -t imageName .
  buildah bud -f Dockerfile.simple -f Dockerfile.notsosimple .`,
	}
	budCommand.SetUsageTemplate(UsageTemplate())

	flags := budCommand.Flags()
	flags.SetInterspersed(false)

	// BUD is a all common flags
	budFlags := buildahcli.GetBudFlags(&budFlagResults)
	layerFlags := buildahcli.GetLayerFlags(&layerFlagsResults)
	fromAndBudFlags := buildahcli.GetFromAndBudFlags(&fromAndBudResults, &userNSResults, &namespaceResults)

	flags.AddFlagSet(&budFlags)
	flags.AddFlagSet(&layerFlags)
	flags.AddFlagSet(&fromAndBudFlags)

	rootCmd.AddCommand(budCommand)
}

func getDockerfiles(files []string) []string {
	var dockerfiles []string
	for _, f := range files {
		if f == "-" {
			dockerfiles = append(dockerfiles, "/dev/stdin")
		} else {
			dockerfiles = append(dockerfiles, f)
		}
	}
	return dockerfiles
}

func budCmd(c *cobra.Command, inputArgs []string, iopts budResults) error {
	output := ""
	tags := []string{}
	if c.Flag("tag").Changed {
		tags = iopts.Tag
		if len(tags) > 0 {
			output = tags[0]
			tags = tags[1:]
		}
	}
	pullPolicy := imagebuildah.PullNever
	if iopts.Pull {
		pullPolicy = imagebuildah.PullIfMissing
	}
	if iopts.PullAlways {
		pullPolicy = imagebuildah.PullAlways
	}

	args := make(map[string]string)
	if c.Flag("build-arg").Changed {
		for _, arg := range iopts.BuildArg {
			av := strings.SplitN(arg, "=", 2)
			if len(av) > 1 {
				args[av[0]] = av[1]
			} else {
				delete(args, av[0])
			}
		}
	}

	dockerfiles := getDockerfiles(iopts.File)
	format, err := getFormat(iopts.Format)
	if err != nil {
		return err
	}
	layers := buildahcli.UseLayers()
	if c.Flag("layers").Changed {
		layers = iopts.Layers
	}
	contextDir := ""
	cliArgs := inputArgs

	// Nothing provided, we assume the current working directory as build
	// context
	if len(cliArgs) == 0 {
		contextDir, err = os.Getwd()
		if err != nil {
			return errors.Wrapf(err, "unable to choose current working directory as build context")
		}
	} else {
		// The context directory could be a URL.  Try to handle that.
		tempDir, subDir, err := imagebuildah.TempDirForURL("", "buildah", cliArgs[0])
		if err != nil {
			return errors.Wrapf(err, "error prepping temporary context directory")
		}
		if tempDir != "" {
			// We had to download it to a temporary directory.
			// Delete it later.
			defer func() {
				if err = os.RemoveAll(tempDir); err != nil {
					logrus.Errorf("error removing temporary directory %q: %v", contextDir, err)
				}
			}()
			contextDir = filepath.Join(tempDir, subDir)
		} else {
			// Nope, it was local.  Use it as is.
			absDir, err := filepath.Abs(cliArgs[0])
			if err != nil {
				return errors.Wrapf(err, "error determining path to directory %q", cliArgs[0])
			}
			contextDir = absDir
		}
	}
	cliArgs = Tail(cliArgs)

	if err := buildahcli.VerifyFlagsArgsOrder(cliArgs); err != nil {
		return err
	}

	if len(dockerfiles) == 0 {
		// Try to find the Dockerfile within the contextDir
		dockerFile, err := discoverDockerfile(contextDir)
		if err != nil {
			return err
		}
		dockerfiles = append(dockerfiles, dockerFile)
		contextDir = filepath.Dir(dockerFile)
	}

	var stdin, stdout, stderr, reporter *os.File
	stdin = os.Stdin
	stdout = os.Stdout
	stderr = os.Stderr
	reporter = os.Stderr
	if c.Flag("logfile").Changed {
		f, err := os.OpenFile(iopts.Logfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Errorf("error opening logfile %q: %v", iopts.Logfile, err)
		}
		defer f.Close()
		logrus.SetOutput(f)
		stdout = f
		stderr = f
		reporter = f
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	isolation, err := parse.IsolationOption(c)
	if err != nil {
		return err
	}

	runtimeFlags := []string{}
	for _, arg := range iopts.RuntimeFlags {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	commonOpts, err := parse.CommonBuildOptions(c)
	if err != nil {
		return err
	}

	if c.Flag("layers").Changed && c.Flag("no-cache").Changed {
		return errors.Errorf("can only set one of 'layers' or 'no-cache'")
	}

	if (c.Flag("rm").Changed || c.Flag("force-rm").Changed) && (!c.Flag("layers").Changed && !c.Flag("no-cache").Changed) {
		return errors.Errorf("'rm' and 'force-rm' can only be set with either 'layers' or 'no-cache'")
	}

	if c.Flag("cache-from").Changed {
		logrus.Debugf("build caching not enabled so --cache-from flag has no effect")
	}

	if c.Flag("compress").Changed {
		logrus.Debugf("--compress option specified but is ignored")
	}

	compression := imagebuildah.Gzip
	if iopts.DisableCompression {
		compression = imagebuildah.Uncompressed
	}

	if c.Flag("disable-content-trust").Changed {
		logrus.Debugf("--disable-content-trust option specified but is ignored")
	}

	namespaceOptions, networkPolicy, err := parse.NamespaceOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing namespace-related options")
	}
	usernsOption, idmappingOptions, err := parse.IDMappingOptions(c, isolation)
	if err != nil {
		return errors.Wrapf(err, "error parsing ID mapping options")
	}
	namespaceOptions.AddOrReplace(usernsOption...)

	defaultsMountFile, _ := c.PersistentFlags().GetString("defaults-mount-file")
	transientMounts := []imagebuildah.Mount{}
	for _, volume := range iopts.Volumes {
		mount, err := parse.Volume(volume)
		if err != nil {
			return err
		}

		transientMounts = append(transientMounts, imagebuildah.Mount(mount))
	}

	devices := []configs.Device{}
	for _, device := range iopts.Devices {
		dev, err := parse.DeviceFromPath(device)
		if err != nil {
			return err
		}
		devices = append(devices, dev)
	}

	options := imagebuildah.BuildOptions{
		ContextDirectory:        contextDir,
		PullPolicy:              pullPolicy,
		Compression:             compression,
		Quiet:                   iopts.Quiet,
		SignaturePolicyPath:     iopts.SignaturePolicy,
		Args:                    args,
		Output:                  output,
		AdditionalTags:          tags,
		In:                      stdin,
		Out:                     stdout,
		Err:                     stderr,
		ReportWriter:            reporter,
		Runtime:                 iopts.Runtime,
		RuntimeArgs:             runtimeFlags,
		OutputFormat:            format,
		SystemContext:           systemContext,
		Isolation:               isolation,
		NamespaceOptions:        namespaceOptions,
		ConfigureNetwork:        networkPolicy,
		CNIPluginPath:           iopts.CNIPlugInPath,
		CNIConfigDir:            iopts.CNIConfigDir,
		IDMappingOptions:        idmappingOptions,
		AddCapabilities:         iopts.CapAdd,
		DropCapabilities:        iopts.CapDrop,
		CommonBuildOpts:         commonOpts,
		DefaultMountsFilePath:   defaultsMountFile,
		IIDFile:                 iopts.Iidfile,
		Squash:                  iopts.Squash,
		Labels:                  iopts.Label,
		Annotations:             iopts.Annotation,
		Layers:                  layers,
		NoCache:                 iopts.NoCache,
		RemoveIntermediateCtrs:  iopts.Rm,
		ForceRmIntermediateCtrs: iopts.ForceRm,
		BlobDirectory:           iopts.BlobCache,
		Target:                  iopts.Target,
		TransientMounts:         transientMounts,
		Devices:                 devices,
	}

	if iopts.Quiet {
		options.ReportWriter = ioutil.Discard
	}

	_, _, err = imagebuildah.BuildDockerfiles(getContext(), store, options, dockerfiles...)
	return err
}

// discoverDockerfile tries to find a Dockerfile within the provided `path`.
func discoverDockerfile(path string) (foundDockerfile string, err error) {
	// Test for existence of the file
	target, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "discovering Dockerfile")
	}

	switch mode := target.Mode(); {
	case mode.IsDir():
		// If the path is a real directory, we assume a Dockerfile within it
		dockerfile := filepath.Join(path, "Dockerfile")

		// Test for existence of the new file
		file, err := os.Stat(dockerfile)
		if err != nil {
			return "", errors.Wrap(err, "finding assumed Dockerfile")
		}

		// The file exists, now verify the correct mode
		if mode := file.Mode(); mode.IsRegular() {
			foundDockerfile = dockerfile
		} else {
			return "", errors.Errorf("assumed Dockerfile %q is not a file", dockerfile)
		}

	case mode.IsRegular():
		// If the context dir is a file, we assume this as Dockerfile
		foundDockerfile = path
	}

	return foundDockerfile, nil
}
