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
		Name:           "build-using-dockerfile",
		Aliases:        []string{"bud"},
		Usage:          "Build an image using instructions in a Dockerfile",
		Description:    budDescription,
		Flags:          append(buildahcli.BudFlags, buildahcli.FromAndBudFlags...),
		Action:         budCmd,
		ArgsUsage:      "CONTEXT-DIRECTORY | URL",
		SkipArgReorder: true,
	}
)

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

	dockerfiles := c.StringSlice("file")
	format := defaultFormat()
	if c.IsSet("format") {
		format = strings.ToLower(c.String("format"))
	}
	if strings.HasPrefix(format, "oci") {
		format = imagebuildah.OCIv1ImageFormat
	} else if strings.HasPrefix(format, "docker") {
		format = imagebuildah.Dockerv2ImageFormat
	} else {
		return errors.Errorf("unrecognized image type %q", format)
	}
	contextDir := ""
	cliArgs := c.Args()
	if len(cliArgs) > 0 {
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
	} else {
		return errors.Errorf("no context directory or URL specified")
	}
	if len(dockerfiles) == 0 {
		dockerfiles = append(dockerfiles, filepath.Join(contextDir, "Dockerfile"))
	}
	if err := parse.ValidateFlags(c, buildahcli.BudFlags); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, buildahcli.FromAndBudFlags); err != nil {
		return err
	}
	var stdout, stderr, reporter *os.File
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

	runtimeFlags := []string{}
	for _, arg := range c.StringSlice("runtime-flag") {
		runtimeFlags = append(runtimeFlags, "--"+arg)
	}

	commonOpts, err := parse.ParseCommonBuildOptions(c)
	if err != nil {
		return err
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

	if c.IsSet("force-rm") {
		logrus.Debugf("build caching not enabled so --force-rm flag has no effect")
	}

	if c.IsSet("no-cache") {
		logrus.Debugf("build caching not enabled so --no-cache flag has no effect")
	}

	if c.IsSet("rm") {
		logrus.Debugf("build caching not enabled so --rm flag has no effect")
	}

	namespaceOptions, networkPolicy, err := parseNamespaceOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing namespace-related options")
	}
	usernsOption, idmappingOptions, err := parseIDMappingOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error parsing ID mapping options")
	}
	namespaceOptions.AddOrReplace(usernsOption...)

	options := imagebuildah.BuildOptions{
		ContextDirectory:      contextDir,
		PullPolicy:            pullPolicy,
		Compression:           imagebuildah.Gzip,
		Quiet:                 c.Bool("quiet"),
		SignaturePolicyPath:   c.String("signature-policy"),
		Args:                  args,
		Output:                output,
		AdditionalTags:        tags,
		Out:                   stdout,
		Err:                   stderr,
		ReportWriter:          reporter,
		Runtime:               c.String("runtime"),
		RuntimeArgs:           runtimeFlags,
		OutputFormat:          format,
		SystemContext:         systemContext,
		NamespaceOptions:      namespaceOptions,
		ConfigureNetwork:      networkPolicy,
		CNIPluginPath:         c.String("cni-plugin-path"),
		CNIConfigDir:          c.String("cni-config-dir"),
		IDMappingOptions:      idmappingOptions,
		CommonBuildOpts:       commonOpts,
		DefaultMountsFilePath: c.GlobalString("default-mounts-file"),
		IIDFile:               c.String("iidfile"),
		Squash:                c.Bool("squash"),
		Labels:                c.StringSlice("label"),
		Annotations:           c.StringSlice("annotation"),
	}

	if c.Bool("quiet") {
		options.ReportWriter = ioutil.Discard
	}

	return imagebuildah.BuildDockerfiles(getContext(), store, options, dockerfiles...)
}
