package main

import (
	"fmt"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type addCopyResults struct {
	addHistory bool
	chown      string
	quiet      bool
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
		Example: `  buildah add containerID '/myapp/app.conf'
  buildah add containerID '/myapp/app.conf' '/myapp/app.conf'`,
		Args: cobra.MinimumNArgs(1),
	}

	copyCommand := &cobra.Command{
		Use:   "copy",
		Short: "Copy content into the container",
		Long:  copyDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return copyCmd(cmd, args, copyOpts)
		},
		Example: `  buildah copy containerID '/myapp/app.conf'
  buildah copy containerID '/myapp/app.conf' '/myapp/app.conf'`,
		Args: cobra.MinimumNArgs(1),
	}

	addFlags := addCommand.Flags()
	addFlags.SetInterspersed(false)
	addFlags.BoolVar(&addOpts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	addFlags.StringVar(&addOpts.chown, "chown", "", "set the user and group ownership of the destination content")
	addFlags.BoolVarP(&addOpts.quiet, "quiet", "q", false, "don't output a digest of the newly-added/copied content")

	// TODO We could avoid some duplication here if need-be; given it is small, leaving as is
	copyFlags := copyCommand.Flags()
	copyFlags.SetInterspersed(false)
	copyFlags.BoolVar(&copyOpts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	copyFlags.StringVar(&copyOpts.chown, "chown", "", "set the user and group ownership of the destination content")
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

	digester := digest.Canonical.Digester()
	options := buildah.AddAndCopyOptions{
		Chown:  iopts.chown,
		Hasher: digester.Hash(),
	}

	if err := builder.Add(dest, extractLocalArchives, options, args...); err != nil {
		return errors.Wrapf(err, "error adding content to container %q", builder.Container)
	}

	if !iopts.quiet {
		fmt.Printf("%s\n", digester.Digest().Hex())
	}
	conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) %s file:%s", verb, digester.Digest().Hex())
	return builder.Save()
}

func addCmd(c *cobra.Command, args []string, iopts addCopyResults) error {
	return addAndCopyCmd(c, args, "ADD", true, iopts)
}

func copyCmd(c *cobra.Command, args []string, iopts addCopyResults) error {
	return addAndCopyCmd(c, args, "COPY", false, iopts)
}
