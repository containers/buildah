package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/imagebuildah"
	buildahcli "github.com/projectatomic/buildah/pkg/cli"
	"github.com/projectatomic/buildah/pkg/parse"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	budDescription = "Builds an OCI image using instructions in one or more Dockerfiles."
	budCommand     = cli.Command{
		Name:                   "build-using-dockerfile",
		Aliases:                []string{"bud"},
		Usage:                  "Build an image using instructions in a Dockerfile",
		Description:            budDescription,
		Flags:                  append(append(buildahcli.BudFlags, buildahcli.LayerFlags...), buildahcli.FromAndBudFlags...),
		Action:                 budCmd,
		ArgsUsage:              "CONTEXT-DIRECTORY | URL",
		SkipArgReorder:         true,
		UseShortOptionHandling: true,
	}
)

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

func budCmd(c *cli.Context) error {
	output := ""
	tags := []string{}
	if c.IsSet("tag") || c.IsSet("t") {
		tags = c.StringSlice("tag")
		if len(tags) > 0 {
			output = tags[0]
			tags = tags[1:]
		}
	}
	pullPolicy := imagebuildah.PullNever
	if c.BoolT("pull") {
		pullPolicy = imagebuildah.PullIfMissing
	}
	if c.Bool("pull-always") {
		pullPolicy = imagebuildah.PullAlways
	}

	args := make(map[string]string)
	if c.IsSet("build-arg") {
		for _, arg := range c.StringSlice("build-arg") {
			av := strings.SplitN(arg, "=", 2)
			if len(av) > 1 {
				args[av[0]] = av[1]
			} else {
				delete(args, av[0])
			}
		}
	}

	dockerfiles := getDockerfiles(c.StringSlice("file"))
	format, err := getFormat(c)
	if err != nil {
		return err
	}
	layers := buildahcli.UseLayers()
	if c.IsSet("layers") {
		layers = c.Bool("layers")
	}
	contextDir := ""
	cliArgs := c.Args()
	if len(cliArgs) == 0 {
		return errors.Errorf("no context directory or URL specified")
	}
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
	cliArgs = cliArgs.Tail()

	if err := buildahcli.VerifyFlagsArgsOrder(cliArgs); err != nil {
		return err
	}
	if len(dockerfiles) == 0 {
		dockerfiles = append(dockerfiles, filepath.Join(contextDir, "Dockerfile"))
	}
	if err := parse.ValidateFlags(c, buildahcli.BudFlags); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, buildahcli.LayerFlags); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, buildahcli.FromAndBudFlags); err != nil {
		return err
	}
	var stdin, stdout, stderr, reporter *os.File
	stdin = os.Stdin
	stdout = os.Stdout
	stderr = os.Stderr
	reporter = os.Stderr
	if c.IsSet("logfile") {
		f, err := os.OpenFile(c.String("logfile"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return errors.Errorf("error opening logfile %q: %v", c.String("logfile"), err)
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
	for _, arg := range c.StringSlice("runtime-flag") {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	commonOpts, err := parse.CommonBuildOptions(c)
	if err != nil {
		return err
	}

	if c.IsSet("layers") && c.IsSet("no-cache") {
		return errors.Errorf("can only set one of 'layers' or 'no-cache'")
	}

	if (c.IsSet("rm") || c.IsSet("force-rm")) && (!c.IsSet("layers") && !c.IsSet("no-cache")) {
		return errors.Errorf("'rm' and 'force-rm' can only be set with either 'layers' or 'no-cache'")
	}

	if c.IsSet("cache-from") {
		logrus.Debugf("build caching not enabled so --cache-from flag has no effect")
	}

	if c.IsSet("compress") {
		logrus.Debugf("--compress option specified but is ignored")
	}

	if c.IsSet("disable-content-trust") {
		logrus.Debugf("--disable-content-trust option specified but is ignored")
	}

	namespaceOptions, networkPolicy, err := parse.NamespaceOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing namespace-related options")
	}
	usernsOption, idmappingOptions, err := parse.IDMappingOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing ID mapping options")
	}
	namespaceOptions.AddOrReplace(usernsOption...)

	options := imagebuildah.BuildOptions{
		ContextDirectory:        contextDir,
		PullPolicy:              pullPolicy,
		Compression:             imagebuildah.Gzip,
		Quiet:                   c.Bool("quiet"),
		SignaturePolicyPath:     c.String("signature-policy"),
		Args:                    args,
		Output:                  output,
		AdditionalTags:          tags,
		In:                      stdin,
		Out:                     stdout,
		Err:                     stderr,
		ReportWriter:            reporter,
		Runtime:                 c.String("runtime"),
		RuntimeArgs:             runtimeFlags,
		OutputFormat:            format,
		SystemContext:           systemContext,
		Isolation:               isolation,
		NamespaceOptions:        namespaceOptions,
		ConfigureNetwork:        networkPolicy,
		CNIPluginPath:           c.String("cni-plugin-path"),
		CNIConfigDir:            c.String("cni-config-dir"),
		IDMappingOptions:        idmappingOptions,
		AddCapabilities:         c.StringSlice("cap-add"),
		DropCapabilities:        c.StringSlice("cap-drop"),
		CommonBuildOpts:         commonOpts,
		DefaultMountsFilePath:   c.GlobalString("default-mounts-file"),
		IIDFile:                 c.String("iidfile"),
		Squash:                  c.Bool("squash"),
		Labels:                  c.StringSlice("label"),
		Annotations:             c.StringSlice("annotation"),
		Layers:                  layers,
		NoCache:                 c.Bool("no-cache"),
		RemoveIntermediateCtrs:  c.BoolT("rm"),
		ForceRmIntermediateCtrs: c.Bool("force-rm"),
	}

	if c.Bool("quiet") {
		options.ReportWriter = ioutil.Discard
	}

	return imagebuildah.BuildDockerfiles(getContext(), store, options, dockerfiles...)
}
