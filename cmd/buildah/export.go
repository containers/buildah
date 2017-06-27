package main

import (
	"fmt"
	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/containers/storage/pkg/archive"
	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	"github.com/urfave/cli"
)

var (
	exportFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "output, o",
			Usage: "write to a file, instead of STDOUT",
		},
	}
	exportCommand = cli.Command{
		Name:  "export",
		Usage: "Export container's filesystem contents as a tar archive",
		Description: `This command exports the full or shortened container ID or container name to
   STDOUT and should be redirected to a tar file.
       `,
		Flags:     exportFlags,
		Action:    exportCmd,
		ArgsUsage: "CONTAINER",
	}
)

func exportCmd(c *cli.Context) error {
	var builder *buildah.Builder

	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("container name must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}

	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err = openBuilder(store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	mountPoint, err := builder.Mount("")
	if err != nil {
		return errors.Wrapf(err, "error mounting %q container %q", name, builder.Container)
	}
	defer func() {
		if err := builder.Unmount(); err != nil {
			fmt.Printf("Failed to umount %q: %v\n", builder.Container, err)
		}
	}()

	input, err := archive.Tar(mountPoint, 0)
	if err != nil {
		return errors.Wrapf(err, "error reading directory %q", name)
	}

	outFile := os.Stdout
	if c.IsSet("output") {
		outfile := c.String("output")
		outFile, err = os.Create(outfile)
		if err != nil {
			return errors.Wrapf(err, "error creating file %q", outfile)
		}
		defer outFile.Close()
	}
	if logrus.IsTerminal(outFile) {
		return errors.Errorf("Refusing to save to a terminal. Use the -o flag or redirect.")
	}

	_, err = io.Copy(outFile, input)
	return err
}
