package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah/imagebuildah"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	budFlags = []cli.Flag{
		cli.StringSliceFlag{
			Name:  "build-arg",
			Usage: "`argument=value` to supply to the builder",
		},
		cli.StringSliceFlag{
			Name:  "file, f",
			Usage: "`pathname or URL` of a Dockerfile",
		},
		cli.StringFlag{
			Name:  "format",
			Usage: "`format` of the built image's manifest and metadata",
		},
		cli.BoolTFlag{
			Name:  "pull",
			Usage: "pull the image if not present",
		},
		cli.BoolFlag{
			Name:  "pull-always",
			Usage: "pull the image, even if a version is present",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "refrain from announcing build instructions and image read/write progress",
		},
		cli.StringFlag{
			Name:  "runtime",
			Usage: "`path` to an alternate runtime",
			Value: imagebuildah.DefaultRuntime,
		},
		cli.StringSliceFlag{
			Name:  "runtime-flag",
			Usage: "add global flags for the container runtime",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.StringSliceFlag{
			Name:  "tag, t",
			Usage: "`tag` to apply to the built image",
		},
	}

	budDescription = "Builds an OCI image using instructions in one or more Dockerfiles."
	budCommand     = cli.Command{
		Name:        "build-using-dockerfile",
		Aliases:     []string{"bud"},
		Usage:       "Build an image using instructions in a Dockerfile",
		Description: budDescription,
		Flags:       budFlags,
		Action:      budCmd,
		ArgsUsage:   "CONTEXT-DIRECTORY | URL",
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
	format := "oci"
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
		// No context directory or URL was specified.  Try to use the
		// home of the first locally-available Dockerfile.
		for i := range dockerfiles {
			if strings.HasPrefix(dockerfiles[i], "http://") ||
				strings.HasPrefix(dockerfiles[i], "https://") ||
				strings.HasPrefix(dockerfiles[i], "git://") ||
				strings.HasPrefix(dockerfiles[i], "github.com/") {
				continue
			}
			absFile, err := filepath.Abs(dockerfiles[i])
			if err != nil {
				return errors.Wrapf(err, "error determining path to file %q", dockerfiles[i])
			}
			contextDir = filepath.Dir(absFile)
			dockerfiles[i], err = filepath.Rel(contextDir, absFile)
			if err != nil {
				return errors.Wrapf(err, "error determining path to file %q", dockerfiles[i])
			}
			break
		}
	}
	if contextDir == "" {
		return errors.Errorf("no context directory specified, and no dockerfile specified")
	}
	if len(dockerfiles) == 0 {
		dockerfiles = append(dockerfiles, filepath.Join(contextDir, "Dockerfile"))
	}
	if err := validateFlags(c, budFlags); err != nil {
		return err
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	options := imagebuildah.BuildOptions{
		ContextDirectory:    contextDir,
		PullPolicy:          pullPolicy,
		Compression:         imagebuildah.Gzip,
		Quiet:               c.Bool("quiet"),
		SignaturePolicyPath: c.String("signature-policy"),
		Args:                args,
		Output:              output,
		AdditionalTags:      tags,
		Runtime:             c.String("runtime"),
		RuntimeArgs:         c.StringSlice("runtime-flag"),
		OutputFormat:        format,
	}
	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}

	return imagebuildah.BuildDockerfiles(store, options, dockerfiles...)
}
