package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/containers/buildah/pkg/parse"
	"github.com/containers/common/pkg/auth"
	"github.com/containers/common/pkg/config"
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
	pullNever       bool
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
	flags.StringVar(&opts.authfile, "authfile", auth.GetDefaultAuthFile(), "path of the authentication file. Use REGISTRY_AUTH_FILE environment variable to override")
	flags.StringVar(&opts.certDir, "cert-dir", "", "use certificates at the specified path to access the registry")
	flags.StringVar(&opts.cidfile, "cidfile", "", "write the container ID to the file")
	flags.StringVar(&opts.creds, "creds", "", "use `[username[:password]]` for accessing the registry")
	flags.StringVarP(&opts.format, "format", "f", defaultFormat(), "`format` of the image manifest and metadata")
	flags.StringVar(&opts.name, "name", "", "`name` for the working container")
	flags.BoolVar(&opts.pull, "pull", true, "pull the image from the registry if newer or not present in store, if false, only pull the image if not present")
	flags.BoolVar(&opts.pullAlways, "pull-always", false, "pull the image even if the named image is present in store")
	flags.BoolVar(&opts.pullNever, "pull-never", false, "do not pull the image, use the image present in store if available")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "don't output progress information when pulling images")
	flags.StringVar(&opts.signaturePolicy, "signature-policy", "", "`pathname` of signature policy file (not usually used)")
	if err := flags.MarkHidden("signature-policy"); err != nil {
		panic(fmt.Sprintf("error marking signature-policy as hidden: %v", err))
	}
	flags.BoolVar(&opts.tlsVerify, "tls-verify", true, "require HTTPS and verify certificates when accessing the registry. TLS verification cannot be used when talking to an insecure registry.")

	// Add in the common flags
	fromAndBudFlags, err := buildahcli.GetFromAndBudFlags(&fromAndBudResults, &userNSResults, &namespaceResults)
	if err != nil {
		logrus.Errorf("failed to setup From and Bud flags: %v", err)
		os.Exit(1)
	}
	flags.AddFlagSet(&fromAndBudFlags)
	flags.SetNormalizeFunc(buildahcli.AliasFlags)

	rootCmd.AddCommand(fromCommand)
}

func onBuild(builder *buildah.Builder, quiet bool) error {
	ctr := 0
	for _, onBuildSpec := range builder.OnBuild() {
		ctr = ctr + 1
		commands := strings.Split(onBuildSpec, " ")
		command := strings.ToUpper(commands[0])
		args := commands[1:]
		if !quiet {
			fmt.Fprintf(os.Stderr, "STEP %d: %s\n", ctr, onBuildSpec)
		}
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
			var stdout io.Writer
			if quiet {
				stdout = ioutil.Discard
			}
			if err := builder.Run(args, buildah.RunOptions{Stdout: stdout}); err != nil {
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
	defaultContainerConfig, err := config.Default()
	if err != nil {
		return errors.Wrapf(err, "failed to get container config")
	}

	if len(args) == 0 {
		return errors.Errorf("an image name (or \"scratch\") must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}

	if err := auth.CheckAuthFile(iopts.authfile); err != nil {
		return err
	}
	systemContext, err := parse.SystemContextFromOptions(c)
	if err != nil {
		return errors.Wrapf(err, "error building system context")
	}
	platforms, err := parse.PlatformsFromOptions(c)
	if err != nil {
		return err
	}
	if len(platforms) > 1 {
		logrus.Warnf("ignoring platforms other than %+v: %+v", platforms[0], platforms[1:])
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

	pullPolicy := define.PullIfMissing
	if iopts.pull {
		pullPolicy = define.PullIfNewer
	}
	if iopts.pullAlways {
		pullPolicy = define.PullAlways
	}
	if iopts.pullNever {
		pullPolicy = define.PullNever
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

	isolation, err := parse.IsolationOption(iopts.Isolation)
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
	devices := define.ContainerDevices{}
	for _, device := range append(defaultContainerConfig.Containers.Devices, iopts.Devices...) {
		dev, err := parse.DeviceFromPath(device)
		if err != nil {
			return err
		}
		devices = append(devices, dev...)
	}

	capabilities, err := defaultContainerConfig.Capabilities("", iopts.CapAdd, iopts.CapDrop)
	if err != nil {
		return err
	}

	commonOpts.Ulimit = append(defaultContainerConfig.Containers.DefaultUlimits, commonOpts.Ulimit...)

	decConfig, err := getDecryptConfig(iopts.DecryptionKeys)
	if err != nil {
		return errors.Wrapf(err, "unable to obtain decrypt config")
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
		Capabilities:          capabilities,
		CommonBuildOpts:       commonOpts,
		Format:                format,
		BlobDirectory:         iopts.BlobCache,
		Devices:               devices,
		DefaultEnv:            defaultContainerConfig.GetDefaultEnv(),
		MaxPullRetries:        maxPullPushRetries,
		PullRetryDelay:        pullPushRetryDelay,
		OciDecryptConfig:      decConfig,
	}

	if !iopts.quiet {
		options.ReportWriter = os.Stderr
	}

	builder, err := buildah.NewBuilder(getContext(), store, options)
	if err != nil {
		return err
	}

	if err := onBuild(builder, iopts.quiet); err != nil {
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
