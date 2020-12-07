package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/buildah/imagebuildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/buildah/util"
	"github.com/containers/common/pkg/auth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type budOptions struct {
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
		Use:     "bud",
		Aliases: []string{"build-using-dockerfile"},
		Short:   "Build an image using instructions in a Dockerfile",
		Long:    budDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			br := budOptions{
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
	budFlags.StringVar(&budFlagResults.Runtime, "runtime", util.Runtime(), "`path` to an alternate runtime. Use BUILDAH_RUNTIME environment variable to override.")

	layerFlags := buildahcli.GetLayerFlags(&layerFlagsResults)
	fromAndBudFlags, err := buildahcli.GetFromAndBudFlags(&fromAndBudResults, &userNSResults, &namespaceResults)
	if err != nil {
		logrus.Errorf("failed to setup From and Bud flags: %v", err)
		os.Exit(1)
	}

	flags.AddFlagSet(&budFlags)
	flags.AddFlagSet(&layerFlags)
	flags.AddFlagSet(&fromAndBudFlags)
	flags.SetNormalizeFunc(buildahcli.AliasFlags)

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

func budCmd(c *cobra.Command, inputArgs []string, iopts budOptions) error {
	output := ""
	tags := []string{}
	if c.Flag("tag").Changed {
		tags = iopts.Tag
		if len(tags) > 0 {
			output = tags[0]
			tags = tags[1:]
		}
	}
	if err := auth.CheckAuthFile(iopts.BudResults.Authfile); err != nil {
		return err
	}

	pullPolicy := imagebuildah.PullIfMissing
	if iopts.Pull {
		pullPolicy = imagebuildah.PullIfNewer
	}
	if iopts.PullAlways {
		pullPolicy = imagebuildah.PullAlways
	}
	if iopts.PullNever {
		pullPolicy = imagebuildah.PullNever
	}
	logrus.Debugf("Pull Policy for pull [%v]", pullPolicy)

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
					logrus.Errorf("error removing temporary directory: %v", err)
				}
			}()
			contextDir = filepath.Join(tempDir, subDir)
		} else {
			// Nope, it was local.  Use it as is.
			absDir, err := filepath.Abs(cliArgs[0])
			if err != nil {
				return errors.Wrapf(err, "error determining path to directory")
			}
			contextDir = absDir
		}
	}
	cliArgs = Tail(cliArgs)

	if err := buildahcli.VerifyFlagsArgsOrder(cliArgs); err != nil {
		return err
	}

	if len(dockerfiles) == 0 {
		// Try to find the Containerfile/Dockerfile within the contextDir
		dockerFile, err := discoverContainerfile(contextDir)
		if err != nil {
			return err
		}
		dockerfiles = append(dockerfiles, dockerFile)
		contextDir = filepath.Dir(dockerFile)
	}

	contextDir, err = filepath.EvalSymlinks(contextDir)
	if err != nil {
		return errors.Wrapf(err, "error evaluating symlinks in build context path")
	}

	var stdout, stderr, reporter *os.File
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

	isolation, err := parse.IsolationOption(iopts.Isolation)
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

	pullFlagsCount := 0
	if c.Flag("pull").Changed {
		pullFlagsCount++
	}
	if c.Flag("pull-always").Changed {
		pullFlagsCount++
	}
	if c.Flag("pull-never").Changed {
		pullFlagsCount++
	}

	if pullFlagsCount > 1 {
		return errors.Errorf("can only set one of 'pull' or 'pull-always' or 'pull-never'")
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
		return err
	}
	usernsOption, idmappingOptions, err := parse.IDMappingOptions(c, isolation)
	if err != nil {
		return errors.Wrapf(err, "error parsing ID mapping options")
	}
	namespaceOptions.AddOrReplace(usernsOption...)

	defaultsMountFile, _ := c.PersistentFlags().GetString("defaults-mount-file")

	imageOS, arch, err := parse.PlatformFromOptions(c)
	if err != nil {
		return err
	}

	decConfig, err := getDecryptConfig(iopts.DecryptionKeys)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain decrypt config")
	}

	options := imagebuildah.BuildOptions{
		AddCapabilities:         iopts.CapAdd,
		AdditionalTags:          tags,
		Annotations:             iopts.Annotation,
		Architecture:            arch,
		Args:                    args,
		BlobDirectory:           iopts.BlobCache,
		CNIConfigDir:            iopts.CNIConfigDir,
		CNIPluginPath:           iopts.CNIPlugInPath,
		CommonBuildOpts:         commonOpts,
		Compression:             compression,
		ConfigureNetwork:        networkPolicy,
		ContextDirectory:        contextDir,
		DefaultMountsFilePath:   defaultsMountFile,
		Devices:                 iopts.Devices,
		DropCapabilities:        iopts.CapDrop,
		Err:                     stderr,
		ForceRmIntermediateCtrs: iopts.ForceRm,
		IDMappingOptions:        idmappingOptions,
		IIDFile:                 iopts.Iidfile,
		Isolation:               isolation,
		Labels:                  iopts.Label,
		Layers:                  layers,
		MaxPullPushRetries:      maxPullPushRetries,
		NamespaceOptions:        namespaceOptions,
		NoCache:                 iopts.NoCache,
		OS:                      imageOS,
		Out:                     stdout,
		Output:                  output,
		OutputFormat:            format,
		PullPolicy:              pullPolicy,
		PullPushRetryDelay:      pullPushRetryDelay,
		Quiet:                   iopts.Quiet,
		RemoveIntermediateCtrs:  iopts.Rm,
		ReportWriter:            reporter,
		Runtime:                 iopts.Runtime,
		RuntimeArgs:             runtimeFlags,
		SignBy:                  iopts.SignBy,
		SignaturePolicyPath:     iopts.SignaturePolicy,
		Squash:                  iopts.Squash,
		SystemContext:           systemContext,
		Target:                  iopts.Target,
		TransientMounts:         iopts.Volumes,
		OciDecryptConfig:        decConfig,
		Jobs:                    &iopts.Jobs,
		LogRusage:               iopts.LogRusage,
	}
	if iopts.IgnoreFile != "" {
		excludes, err := parseDockerignore(iopts.IgnoreFile)
		if err != nil {
			return err
		}
		options.Excludes = excludes
	}
	if c.Flag("timestamp").Changed {
		timestamp := time.Unix(iopts.Timestamp, 0).UTC()
		options.Timestamp = &timestamp
	}

	if iopts.Quiet {
		options.ReportWriter = ioutil.Discard
	}

	_, _, err = imagebuildah.BuildDockerfiles(getContext(), store, options, dockerfiles...)
	return err
}

// discoverContainerfile tries to find a Containerfile or a Dockerfile within the provided `path`.
func discoverContainerfile(path string) (foundCtrFile string, err error) {
	// Test for existence of the file
	target, err := os.Stat(path)
	if err != nil {
		return "", errors.Wrap(err, "discovering Containerfile")
	}

	switch mode := target.Mode(); {
	case mode.IsDir():
		// If the path is a real directory, we assume a Containerfile or a Dockerfile within it
		ctrfile := filepath.Join(path, "Containerfile")

		// Test for existence of the Containerfile file
		file, err := os.Stat(ctrfile)
		if err != nil {
			// See if we have a Dockerfile within it
			ctrfile = filepath.Join(path, "Dockerfile")

			// Test for existence of the Dockerfile file
			file, err = os.Stat(ctrfile)
			if err != nil {
				return "", errors.Wrap(err, "cannot find Containerfile or Dockerfile in context directory")
			}
		}

		// The file exists, now verify the correct mode
		if mode := file.Mode(); mode.IsRegular() {
			foundCtrFile = ctrfile
		} else {
			return "", errors.Errorf("assumed Containerfile %q is not a file", ctrfile)
		}

	case mode.IsRegular():
		// If the context dir is a file, we assume this as Containerfile
		foundCtrFile = path
	}

	return foundCtrFile, nil
}
