package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containers/buildah"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type fromReply struct {
	authfile        string
	certDir         string
	cidfile         string
	creds           string
	format          string
	name            string
	pull            bool
	pullAlways      bool
	quiet           bool
	signaturePolicy string
	tlsVerify       bool
	*buildahcli.FromAndBudResults
	*buildahcli.UserNSResults
	*buildahcli.NameSpaceResults
}

func init() {
	var (
		fromDescription = "\n  Creates a new working container, either from scratch or using a specified\n  image as a starting point."
		opts            fromReply
	)
	fromAndBudResults := buildahcli.FromAndBudResults{}
	userNSResults := buildahcli.UserNSResults{}
	namespaceResults := buildahcli.NameSpaceResults{}
	fromCommand := &cobra.Command{
		Use:   "from",
		Short: "Create a working container based on an image",
		Long:  fromDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Add in the results from the common cli commands
			opts.FromAndBudResults = &fromAndBudResults
			opts.UserNSResults = &userNSResults
			opts.NameSpaceResults = &namespaceResults
			return fromCmd(cmd, args, opts)
		},
		Example: `buildah from --pull imagename
  buildah from docker-daemon:imagename:imagetag
  buildah from --name "myimagename" myregistry/myrepository/imagename:imagetag`,
	}
	fromCommand.SetUsageTemplate(UsageTemplate())

	flags := fromCommand.Flags()
	flags.SetInterspersed(false)
	flags.StringVar(&opts.authfile, "authfile", buildahcli.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.cidfile, "cidfile", "", "write the container ID to the file")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVarP(&opts.format, "format", "f", defaultFormat(), "`format` of the image manifest and metadata")
	flags.StringVar(&opts.name, "name", "", "`name` for the working container")
	flags.BoolVar(&opts.pull, "pull", true, "pull the image if not present")
	flags.BoolVar(&opts.pullAlways, "pull-always", false, "pull the image even if named image is present in store (supersedes pull option)")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pulling images")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	flags.MarkHidden("signature-policy")
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry")

	// Add in the common flags
	fromAndBudFlags := buildahcli.GetFromAndBudFlags(&fromAndBudResults, &userNSResults, &namespaceResults)
	flags.AddFlagSet(&fromAndBudFlags)

	rootCmd.AddCommand(fromCommand)
}

func onBuild(builder *buildah.Builder) error {
	ctr := 0
	for _, onBuildSpec := range builder.OnBuild() {
		ctr = ctr + 1
		commands := strings.Split(onBuildSpec, " ")
		command := strings.ToUpper(commands[0])
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
			if err := builder.Add(dest, command == "ADD", buildah.AddAndCopyOptions{}, args...); err != nil {
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

func fromCmd(c *cobra.Command, args []string, iopts fromReply) error {
	if len(args) == 0 {
		return errors.Errorf("an image name (or \"scratch\") must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}

	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}

	pullPolicy := buildah.PullNever
	if iopts.pull {
		pullPolicy = buildah.PullIfMissing
	}
	if iopts.pullAlways {
		pullPolicy = buildah.PullAlways
	}

	signaturePolicy := iopts.signaturePolicy

	store, err := getStore(c)
	if err != nil {
		return err
	}

	commonOpts, err := parse.CommonBuildOptions(c)
	if err != nil {
		return err
	}

	isolation, err := parse.IsolationOption(c)
	if err != nil {
		return err
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

	format, err := getFormat(iopts.format)
	if err != nil {
		return err
	}

	options := buildah.BuilderOptions{
		FromImage:             args[0],
		Container:             iopts.name,
		PullPolicy:            pullPolicy,
		SignaturePolicyPath:   signaturePolicy,
		SystemContext:         systemContext,
		DefaultMountsFilePath: globalFlagResults.DefaultMountsFile,
		Isolation:             isolation,
		NamespaceOptions:      namespaceOptions,
		ConfigureNetwork:      networkPolicy,
		CNIPluginPath:         iopts.CNIPlugInPath,
		CNIConfigDir:          iopts.CNIConfigDir,
		IDMappingOptions:      idmappingOptions,
		AddCapabilities:       iopts.CapAdd,
		DropCapabilities:      iopts.CapDrop,
		CommonBuildOpts:       commonOpts,
		Format:                format,
		BlobDirectory:         iopts.BlobCache,
	}

	if !iopts.quiet {
		options.ReportWriter = os.Stderr
	}

	builder, err := buildah.NewBuilder(getContext(), store, options)
	if err != nil {
		return err
	}

	if err := onBuild(builder); err != nil {
		return err
	}

	if iopts.cidfile != "" {
		filePath := iopts.cidfile
		if err := ioutil.WriteFile(filePath, []byte(builder.ContainerID), 0644); err != nil {
			return errors.Wrapf(err, "filed to write Container ID File %q", filePath)
		}
	}
	fmt.Printf("%s\n", builder.Container)
	return builder.Save()
}
