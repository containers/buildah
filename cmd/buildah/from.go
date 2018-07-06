package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/projectatomic/buildah"
	buildahcli "github.com/projectatomic/buildah/pkg/cli"
	"github.com/projectatomic/buildah/pkg/parse"
	util "github.com/projectatomic/buildah/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	fromFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "authfile",
			Usage: "path of the authentication file. Default is ${XDG_RUNTIME_DIR}/containers/auth.json",
		},
		cli.StringFlag{
			Name:  "cert-dir",
			Value: "",
			Usage: "use certificates at the specified path to access the registry",
		},
		cli.StringFlag{
			Name:  "cidfile",
			Usage: "write the container ID to the file",
		},
		cli.StringFlag{
			Name:  "creds",
			Value: "",
			Usage: "use `[username[:password]]` for accessing the registry",
		},
		cli.StringFlag{
			Name:  "name",
			Usage: "`name` for the working container",
		},
		cli.BoolTFlag{
			Name:  "pull",
			Usage: "pull the image if not present",
		},
		cli.BoolFlag{
			Name:  "pull-always",
			Usage: "pull the image even if named image is present in store (supersedes pull option)",
		},
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "don't output progress information when pulling images",
		},
		cli.StringFlag{
			Name:  "signature-policy",
			Usage: "`pathname` of signature policy file (not usually used)",
		},
		cli.BoolTFlag{
			Name:  "tls-verify",
			Usage: "require HTTPS and verify certificates when accessing the registry",
		},
	}
	fromDescription = "Creates a new working container, either from scratch or using a specified\n   image as a starting point"

	fromCommand = cli.Command{
		Name:           "from",
		Usage:          "Create a working container based on an image",
		Description:    fromDescription,
		Flags:          append(fromFlags, buildahcli.FromAndBudFlags...),
		Action:         fromCmd,
		ArgsUsage:      "IMAGE",
		SkipArgReorder: true,
	}
)

func onBuild(builder *buildah.Builder) error {
	ctr := 0
	for _, onBuildSpec := range builder.OnBuild() {
		ctr = ctr + 1
		commands := strings.Split(onBuildSpec, " ")
		command := commands[0]
		args := commands[1:]
		fmt.Fprintf(os.Stderr, "STEP %d: %s\n", ctr, onBuildSpec)
		switch command {
		case "ADD":
		case "COPY":
			dest := ""
			size := len(args)
			if size > 1 {
				dest = args[size-1]
				args = args[:size-1]
			}
			if err := builder.Add(dest, false, buildah.AddAndCopyOptions{}, args...); err != nil {
				return err
			}
		case "ANNOTATION":
			annotation := strings.SplitN(args[0], "=", 2)
			if len(annotation) > 1 {
				builder.SetAnnotation(annotation[0], annotation[1])
			} else {
				builder.UnsetAnnotation(annotation[0])
			}
		case "CMD":
			builder.SetCmd(args)
		case "ENV":
			env := strings.SplitN(args[0], "=", 2)
			if len(env) > 1 {
				builder.SetEnv(env[0], env[1])
			} else {
				builder.UnsetEnv(env[0])
			}
		case "ENTRYPOINT":
			builder.SetEntrypoint(args)
		case "EXPOSE":
			builder.SetPort(strings.Join(args, " "))
		case "HOSTNAME":
			builder.SetHostname(strings.Join(args, " "))
		case "LABEL":
			label := strings.SplitN(args[0], "=", 2)
			if len(label) > 1 {
				builder.SetLabel(label[0], label[1])
			} else {
				builder.UnsetLabel(label[0])
			}
		case "MAINTAINER":
			builder.SetMaintainer(strings.Join(args, " "))
		case "ONBUILD":
			builder.SetOnBuild(strings.Join(args, " "))
		case "RUN":
			if err := builder.Run(args, buildah.RunOptions{}); err != nil {
				return err
			}
		case "SHELL":
			builder.SetShell(args)
		case "STOPSIGNAL":
			builder.SetStopSignal(strings.Join(args, " "))
		case "USER":
			builder.SetUser(strings.Join(args, " "))
		case "VOLUME":
			builder.AddVolume(strings.Join(args, " "))
		case "WORKINGDIR":
			builder.SetWorkDir(strings.Join(args, " "))
		default:
			logrus.Errorf("unknown OnBuild command %q; ignored", onBuildSpec)
		}
	}
	builder.ClearOnBuild()
	return nil
}

func fromCmd(c *cli.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return errors.Errorf("an image name (or \"scratch\") must be specified")
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	if err := parse.ValidateFlags(c, fromFlags); err != nil {
		return err
	}
	if err := parse.ValidateFlags(c, buildahcli.FromAndBudFlags); err != nil {
		return err
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	pullPolicy := buildah.PullNever
	if c.BoolT("pull") {
		pullPolicy = buildah.PullIfMissing
	}
	if c.Bool("pull-always") {
		pullPolicy = buildah.PullAlways
	}

	signaturePolicy := c.String("signature-policy")

	store, err := getStore(c)
	if err != nil {
		return err
	}

	commonOpts, err := parse.CommonBuildOptions(c)
	if err != nil {
		return err
	}

	transport := buildah.DefaultTransport
	arr := strings.SplitN(args[0], ":", 2)
	if len(arr) == 2 {
		if _, ok := util.Transports[arr[0]]; ok {
			transport = arr[0]
		}
	}

	isolation, err := parse.IsolationOption(c)
	if err != nil {
		return err
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

	options := buildah.BuilderOptions{
		FromImage:             args[0],
		Transport:             transport,
		Container:             c.String("name"),
		PullPolicy:            pullPolicy,
		SignaturePolicyPath:   signaturePolicy,
		SystemContext:         systemContext,
		DefaultMountsFilePath: c.GlobalString("default-mounts-file"),
		Isolation:             isolation,
		NamespaceOptions:      namespaceOptions,
		ConfigureNetwork:      networkPolicy,
		CNIPluginPath:         c.String("cni-plugin-path"),
		CNIConfigDir:          c.String("cni-config-dir"),
		IDMappingOptions:      idmappingOptions,
		AddCapabilities:       c.StringSlice("cap-add"),
		DropCapabilities:      c.StringSlice("cap-drop"),
		CommonBuildOpts:       commonOpts,
	}

	if !c.Bool("quiet") {
		options.ReportWriter = os.Stderr
	}

	builder, err := buildah.NewBuilder(getContext(), store, options)
	if err != nil {
		return err
	}

	if err := onBuild(builder); err != nil {
		return err
	}

	if c.String("cidfile") != "" {
		filePath := c.String("cidfile")
		if err := ioutil.WriteFile(filePath, []byte(builder.ContainerID), 0644); err != nil {
			return errors.Wrapf(err, "filed to write Container ID File %q", filePath)
		}
	}
	fmt.Printf("%s\n", builder.Container)
	return builder.Save()
}
