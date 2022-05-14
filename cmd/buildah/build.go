package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containers/buildah/define"
	"github.com/containers/buildah/imagebuildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	buildahutil "github.com/containers/buildah/pkg/util"
	"github.com/containers/buildah/util"
	"github.com/containers/common/pkg/auth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type buildOptions struct {
	*buildahcli.LayerResults
	*buildahcli.BudResults
	*buildahcli.UserNSResults
	*buildahcli.FromAndBudResults
	*buildahcli.NameSpaceResults
}

func init() {
	buildDescription := `
  Builds an OCI image using instructions in one or more Containerfiles.

  If no arguments are specified, Buildah will use the current working directory
  as the build context and look for a Containerfile. The build fails if no
  Containerfile nor Dockerfile is present.`

	layerFlagsResults := buildahcli.LayerResults{}
	buildFlagResults := buildahcli.BudResults{}
	fromAndBudResults := buildahcli.FromAndBudResults{}
	userNSResults := buildahcli.UserNSResults{}
	namespaceResults := buildahcli.NameSpaceResults{}

	buildCommand := &cobra.Command{
		Use:     "build [CONTEXT]",
		Aliases: []string{"build-using-dockerfile", "bud"},
		Short:   "Build an image using instructions in a Containerfile",
		Long:    buildDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			br := buildOptions{
				&layerFlagsResults,
				&buildFlagResults,
				&userNSResults,
				&fromAndBudResults,
				&namespaceResults,
			}
			return buildCmd(cmd, args, br)
		},
		Args: cobra.MaximumNArgs(1),
		Example: `buildah build
  buildah bud -f Containerfile.simple .
  buildah bud --volume /home/test:/myvol:ro,Z -t imageName .
  buildah bud -f Containerfile.simple -f Containerfile.notsosimple .`,
	}
	buildCommand.SetUsageTemplate(UsageTemplate())

	flags := buildCommand.Flags()
	flags.SetInterspersed(false)

	// build is a all common flags
	buildFlags := buildahcli.GetBudFlags(&buildFlagResults)
	buildFlags.StringVar(&buildFlagResults.Runtime, "runtime", util.Runtime(), "`path` to an alternate runtime. Use BUILDAH_RUNTIME environment variable to override.")

	layerFlags := buildahcli.GetLayerFlags(&layerFlagsResults)
	fromAndBudFlags, err := buildahcli.GetFromAndBudFlags(&fromAndBudResults, &userNSResults, &namespaceResults)
	if err != nil {
		logrus.Errorf("failed to setup From and Build flags: %v", err)
		os.Exit(1)
	}

	flags.AddFlagSet(&buildFlags)
	flags.AddFlagSet(&layerFlags)
	flags.AddFlagSet(&fromAndBudFlags)
	flags.SetNormalizeFunc(buildahcli.AliasFlags)

	rootCmd.AddCommand(buildCommand)
}

func getContainerfiles(files []string) []string {
	var containerfiles []string
	for _, f := range files {
		if f == "-" {
			containerfiles = append(containerfiles, "/dev/stdin")
		} else {
			containerfiles = append(containerfiles, f)
		}
	}
	return containerfiles
}

func buildCmd(c *cobra.Command, inputArgs []string, iopts buildOptions) error {
	output := ""
	cleanTmpFile := false
	tags := []string{}
	if c.Flag("tag").Changed {
		tags = iopts.Tag
		if len(tags) > 0 {
			output = tags[0]
			tags = tags[1:]
		}
		if c.Flag("manifest").Changed {
			for _, tag := range tags {
				if tag == iopts.Manifest {
					return errors.New("the same name must not be specified for both '--tag' and '--manifest'")
				}
			}
		}
	}
	if err := auth.CheckAuthFile(iopts.BudResults.Authfile); err != nil {
		return err
	}
	iopts.BudResults.Authfile, cleanTmpFile = buildahutil.MirrorToTempFileIfPathIsDescriptor(iopts.BudResults.Authfile)
	if cleanTmpFile {
		defer os.Remove(iopts.BudResults.Authfile)
	}

	// Allow for --pull, --pull=true, --pull=false, --pull=never, --pull=always
	// --pull-always and --pull-never.  The --pull-never and --pull-always options
	// will not be documented.
	pullPolicy := define.PullIfMissing
	if strings.EqualFold(strings.TrimSpace(iopts.Pull), "true") {
		pullPolicy = define.PullIfNewer
	}
	if iopts.PullAlways || strings.EqualFold(strings.TrimSpace(iopts.Pull), "always") {
		pullPolicy = define.PullAlways
	}
	if iopts.PullNever || strings.EqualFold(strings.TrimSpace(iopts.Pull), "never") {
		pullPolicy = define.PullNever
	}
	logrus.Debugf("Pull Policy for pull [%v]", pullPolicy)

	args := make(map[string]string)
	if c.Flag("build-arg").Changed {
		for _, arg := range iopts.BuildArg {
			av := strings.SplitN(arg, "=", 2)
			if len(av) > 1 {
				args[av[0]] = av[1]
			} else {
				// check if the env is set in the local environment and use that value if it is
				if val, present := os.LookupEnv(av[0]); present {
					args[av[0]] = val
				} else {
					delete(args, av[0])
				}
			}
		}
	}

	containerfiles := getContainerfiles(iopts.File)
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
		tempDir, subDir, err := define.TempDirForURL("", "buildah", cliArgs[0])
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

	if len(containerfiles) == 0 {
		// Try to find the Containerfile/Dockerfile within the contextDir
		containerfile, err := buildahutil.DiscoverContainerfile(contextDir)
		if err != nil {
			return err
		}
		containerfiles = append(containerfiles, containerfile)
		contextDir = filepath.Dir(containerfile)
	}

	contextDir, err = filepath.EvalSymlinks(contextDir)
	if err != nil {
		return errors.Wrapf(err, "error evaluating symlinks in build context path")
	}

	var stdin io.Reader
	if iopts.Stdin {
		stdin = os.Stdin
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

	if (c.Flag("rm").Changed || c.Flag("force-rm").Changed) && (!c.Flag("layers").Changed && !c.Flag("no-cache").Changed) {
		return errors.Errorf("'rm' and 'force-rm' can only be set with either 'layers' or 'no-cache'")
	}

	if c.Flag("cache-from").Changed {
		logrus.Debugf("build --cache-from not enabled, has no effect")
	}

	if c.Flag("compress").Changed {
		logrus.Debugf("--compress option specified but is ignored")
	}

	compression := define.Gzip
	if iopts.DisableCompression {
		compression = define.Uncompressed
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

	platforms, err := parse.PlatformsFromOptions(c)
	if err != nil {
		return err
	}

	decConfig, err := getDecryptConfig(iopts.DecryptionKeys)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain decrypt config")
	}

	var excludes []string
	if iopts.IgnoreFile != "" {
		if excludes, _, err = parse.ContainerIgnoreFile(contextDir, iopts.IgnoreFile); err != nil {
			return err
		}
	}
	var timestamp *time.Time
	if c.Flag("timestamp").Changed {
		t := time.Unix(iopts.Timestamp, 0).UTC()
		timestamp = &t
	}
	if c.Flag("output").Changed {
		buildOption, err := parse.GetBuildOutput(iopts.BuildOutput)
		if err != nil {
			return err
		}
		if buildOption.IsStdout {
			iopts.Quiet = true
		}
	}
	options := define.BuildOptions{
		AddCapabilities:         iopts.CapAdd,
		AdditionalTags:          tags,
		AllPlatforms:            iopts.AllPlatforms,
		Annotations:             iopts.Annotation,
		Architecture:            systemContext.ArchitectureChoice,
		Args:                    args,
		BlobDirectory:           iopts.BlobCache,
		CNIConfigDir:            iopts.CNIConfigDir,
		CNIPluginPath:           iopts.CNIPlugInPath,
		CommonBuildOpts:         commonOpts,
		Compression:             compression,
		ConfigureNetwork:        networkPolicy,
		ContextDirectory:        contextDir,
		CPPFlags:                iopts.CPPFlags,
		DefaultMountsFilePath:   globalFlagResults.DefaultMountsFile,
		Devices:                 iopts.Devices,
		DropCapabilities:        iopts.CapDrop,
		Err:                     stderr,
		ForceRmIntermediateCtrs: iopts.ForceRm,
		From:                    iopts.From,
		IDMappingOptions:        idmappingOptions,
		IIDFile:                 iopts.Iidfile,
		In:                      stdin,
		Isolation:               isolation,
		IgnoreFile:              iopts.IgnoreFile,
		Labels:                  iopts.Label,
		Layers:                  layers,
		LogRusage:               iopts.LogRusage,
		Manifest:                iopts.Manifest,
		MaxPullPushRetries:      maxPullPushRetries,
		NamespaceOptions:        namespaceOptions,
		NoCache:                 iopts.NoCache,
		OS:                      systemContext.OSChoice,
		Out:                     stdout,
		Output:                  output,
		BuildOutput:             iopts.BuildOutput,
		OutputFormat:            format,
		PullPolicy:              pullPolicy,
		PullPushRetryDelay:      pullPushRetryDelay,
		Quiet:                   iopts.Quiet,
		RemoveIntermediateCtrs:  iopts.Rm,
		ReportWriter:            reporter,
		Runtime:                 iopts.Runtime,
		RuntimeArgs:             runtimeFlags,
		RusageLogFile:           iopts.RusageLogFile,
		SignBy:                  iopts.SignBy,
		SignaturePolicyPath:     iopts.SignaturePolicy,
		Squash:                  iopts.Squash,
		SystemContext:           systemContext,
		Target:                  iopts.Target,
		TransientMounts:         iopts.Volumes,
		OciDecryptConfig:        decConfig,
		Jobs:                    &iopts.Jobs,
		Excludes:                excludes,
		Timestamp:               timestamp,
		Platforms:               platforms,
		UnsetEnvs:               iopts.UnsetEnvs,
		Envs:                    iopts.Envs,
		OSFeatures:              iopts.OSFeatures,
		OSVersion:               iopts.OSVersion,
	}
	if iopts.Quiet {
		options.ReportWriter = ioutil.Discard
	}

	id, ref, err := imagebuildah.BuildDockerfiles(getContext(), store, options, containerfiles...)
	if err == nil && options.Manifest != "" {
		logrus.Debugf("manifest list id = %q, ref = %q", id, ref.String())
	}
	return err
}
