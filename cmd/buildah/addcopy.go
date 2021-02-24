package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type addCopyResults struct {
	addHistory bool
	chmod      string
	chown      string
	quiet      bool
	ignoreFile string
	contextdir string
}

func init() {
	var (
		addDescription  = "\n  Adds the contents of a file, URL, or directory to a container's working\n  directory.  If a local file appears to be an archive, its contents are\n  extracted and added instead of the archive file itself."
		copyDescription = "\n  Copies the contents of a file, URL, or directory into a container's working\n  directory."
		addOpts         addCopyResults
		copyOpts        addCopyResults
	)
	addCommand := &cobra.Command{
		Use:   "add",
		Short: "Add content to the container",
		Long:  addDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return addCmd(cmd, args, addOpts)
		},
		Example: `buildah add containerID '/myapp/app.conf'
  buildah add containerID '/myapp/app.conf' '/myapp/app.conf'`,
		Args: cobra.MinimumNArgs(1),
	}
	addCommand.SetUsageTemplate(UsageTemplate())

	copyCommand := &cobra.Command{
		Use:   "copy",
		Short: "Copy content into the container",
		Long:  copyDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return copyCmd(cmd, args, copyOpts)
		},
		Example: `buildah copy containerID '/myapp/app.conf'
  buildah copy containerID '/myapp/app.conf' '/myapp/app.conf'`,
		Args: cobra.MinimumNArgs(1),
	}
	copyCommand.SetUsageTemplate(UsageTemplate())

	addFlags := addCommand.Flags()
	addFlags.SetInterspersed(false)
	addFlags.BoolVar(&addOpts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	addFlags.StringVar(&addOpts.chown, "chown", "", "set the user and group ownership of the destination content")
	addFlags.StringVar(&addOpts.chmod, "chmod", "", "set the access permissions of the destination content")
	addFlags.StringVar(&addOpts.contextdir, "contextdir", "", "context directory path")
	addFlags.StringVar(&addOpts.ignoreFile, "ignorefile", "", "path to .dockerignore file")
	addFlags.BoolVarP(&addOpts.quiet, "quiet", "q", false, "don't output a digest of the newly-added/copied content")

	// TODO We could avoid some duplication here if need-be; given it is small, leaving as is
	copyFlags := copyCommand.Flags()
	copyFlags.SetInterspersed(false)
	copyFlags.BoolVar(&copyOpts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	copyFlags.StringVar(&copyOpts.chown, "chown", "", "set the user and group ownership of the destination content")
	copyFlags.StringVar(&copyOpts.chmod, "chmod", "", "set the access permissions of the destination content")
	copyFlags.StringVar(&copyOpts.ignoreFile, "ignorefile", "", "path to .dockerignore file")
	copyFlags.StringVar(&copyOpts.contextdir, "contextdir", "", "context directory path")
	copyFlags.BoolVarP(&copyOpts.quiet, "quiet", "q", false, "don't output a digest of the newly-added/copied content")

	rootCmd.AddCommand(addCommand)
	rootCmd.AddCommand(copyCommand)
}

func addAndCopyCmd(c *cobra.Command, args []string, verb string, extractLocalArchives bool, iopts addCopyResults) error {
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	name := args[0]
	args = Tail(args)
	if len(args) == 0 {
		return errors.Errorf("src must be specified")
	}

	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}

	// If list is greater than one, the last item is the destination
	dest := ""
	size := len(args)
	if size > 1 {
		dest = args[size-1]
		args = args[:size-1]
	}

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	builder.ContentDigester.Restart()

	options := buildah.AddAndCopyOptions{
		Chmod:      iopts.chmod,
		Chown:      iopts.chown,
		ContextDir: iopts.contextdir,
	}
	if iopts.ignoreFile != "" {
		if iopts.contextdir == "" {
			return errors.Errorf("--ignore options requires that you specify a context dir using --contextdir")
		}

		excludes, err := parseDockerignore(iopts.ignoreFile)
		if err != nil {
			return err
		}
		options.Excludes = excludes
	}

	err = builder.Add(dest, extractLocalArchives, options, args...)
	if err != nil {
		return errors.Wrapf(err, "error adding content to container %q", builder.Container)
	}

	contentType, digest := builder.ContentDigester.Digest()
	if !iopts.quiet {
		fmt.Printf("%s\n", digest.Hex())
	}
	if contentType != "" {
		contentType = contentType + ":"
	}
	conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) %s %s%s", verb, contentType, digest.Hex())
	return builder.Save()
}

func addCmd(c *cobra.Command, args []string, iopts addCopyResults) error {
	return addAndCopyCmd(c, args, "ADD", true, iopts)
}

func copyCmd(c *cobra.Command, args []string, iopts addCopyResults) error {
	return addAndCopyCmd(c, args, "COPY", false, iopts)
}

func parseDockerignore(ignoreFile string) ([]string, error) {
	var excludes []string
	ignore, err := ioutil.ReadFile(ignoreFile)
	if err != nil {
		return excludes, err
	}
	for _, e := range strings.Split(string(ignore), "\n") {
		if len(e) == 0 || e[0] == '#' {
			continue
		}
		excludes = append(excludes, e)
	}
	return excludes, nil
}
