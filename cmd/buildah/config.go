package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/docker"
	buildahcli "github.com/containers/buildah/pkg/cli"
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type configResults struct {
	addHistory             bool
	annotation             []string
	arch                   string
	author                 string
	cmd                    string
	comment                string
	createdBy              string
	domainName             string
	entrypoint             string
	env                    []string
	healthcheck            string
	healthcheckInterval    string
	healthcheckRetries     int
	healthcheckStartPeriod string
	healthcheckTimeout     string
	historyComment         string
	hostname               string
	label                  []string
	onbuild                []string
	os                     string
	ports                  []string
	shell                  string
	stopSignal             string
	user                   string
	volume                 []string
	workingDir             string
}

func init() {
	var (
		configDescription = "\n  Modifies the configuration values which will be saved to the image."
		opts              configResults
	)
	configCommand := &cobra.Command{
		Use:   "config",
		Short: "Update image configuration settings",
		Long:  configDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return configCmd(cmd, args, opts)
		},
		Example: `buildah config --author='Jane Austen' --workingdir='/etc/mycontainers' containerID
  buildah config --entrypoint '[ "/entrypoint.sh", "dev" ]' containerID
  buildah config --env foo=bar --env PATH=$PATH containerID`,
	}
	configCommand.SetUsageTemplate(UsageTemplate())

	flags := configCommand.Flags()
	flags.SetInterspersed(false)
	flags.BoolVar(&opts.addHistory, "add-history", false, "add an entry for this operation to the image's history.  Use BUILDAH_HISTORY environment variable to override. (default false)")
	flags.StringArrayVarP(&opts.annotation, "annotation", "a", []string{}, "add `annotation` e.g. annotation=value, for the target image (default [])")
	flags.StringVar(&opts.arch, "arch", "", "set `architecture` of the target image")
	flags.StringVar(&opts.author, "author", "", "set image author contact `information`")
	flags.StringVar(&opts.cmd, "cmd", "", "set the default `command` to run for containers based on the image")
	flags.StringVar(&opts.comment, "comment", "", "set a `comment` in the target image")
	flags.StringVar(&opts.createdBy, "created-by", "", "set `description` of how the image was created")
	flags.StringVar(&opts.domainName, "domainname", "", "set a domain `name` for containers based on image")
	flags.StringVar(&opts.entrypoint, "entrypoint", "", "set `entry point` for containers based on image")
	flags.StringArrayVarP(&opts.env, "env", "e", []string{}, "add `environment variable` to be set when running containers based on image (default [])")
	flags.StringVar(&opts.healthcheck, "healthcheck", "", "set a `healthcheck` command for the target image")
	flags.StringVar(&opts.healthcheckInterval, "healthcheck-interval", "", "set the `interval` between runs of the `healthcheck` command for the target image")
	flags.IntVar(&opts.healthcheckRetries, "healthcheck-retries", 0, "set the `number` of times the `healthcheck` command has to fail")
	flags.StringVar(&opts.healthcheckStartPeriod, "healthcheck-start-period", "", "set the amount of `time` to wait after starting a container before a failed `healthcheck` command will count as a failure")
	flags.StringVar(&opts.healthcheckTimeout, "healthcheck-timeout", "", "set the maximum amount of `time` to wait for a `healthcheck` command for the target image")
	flags.StringVar(&opts.historyComment, "history-comment", "", "set a `comment` for the history of the target image")
	flags.StringVar(&opts.hostname, "hostname", "", "set a host`name` for containers based on image")
	flags.StringArrayVarP(&opts.label, "label", "l", []string{}, "add image configuration `label` e.g. label=value")
	flags.StringSliceVar(&opts.onbuild, "onbuild", []string{}, "add onbuild command to be run on images based on this image. Only supported on 'docker' formatted images")
	flags.StringVar(&opts.os, "os", "", "set `operating system` of the target image")
	flags.StringSliceVarP(&opts.ports, "port", "p", []string{}, "add `port` to expose when running containers based on image (default [])")
	flags.StringVar(&opts.shell, "shell", "", "add `shell` to run in containers")
	flags.StringVar(&opts.stopSignal, "stop-signal", "", "set `stop signal` for containers based on image")
	flags.StringVarP(&opts.user, "user", "u", "", "set default `user` to run inside containers based on image")
	flags.StringSliceVarP(&opts.volume, "volume", "v", []string{}, "add default `volume` path to be created for containers based on image (default [])")
	flags.StringVar(&opts.workingDir, "workingdir", "", "set working `directory` for containers based on image")

	rootCmd.AddCommand(configCommand)

}

func updateEntrypoint(builder *buildah.Builder, iopts configResults) {
	entrypoint := iopts.entrypoint
	if len(strings.TrimSpace(entrypoint)) == 0 {
		builder.SetEntrypoint(nil)
		return
	}
	var entrypointJSON []string
	err := json.Unmarshal([]byte(entrypoint), &entrypointJSON)

	if err == nil {
		builder.SetEntrypoint(entrypointJSON)
		if len(builder.Cmd()) > 0 {
			logrus.Warnf("cmd %q exists and will be passed to entrypoint as a parameter", strings.Join(builder.Cmd(), " "))
		}
		return
	}

	// it wasn't a valid json array, fall back to string
	entrypointSpec := make([]string, 3)
	entrypointSpec[0] = "/bin/sh"
	entrypointSpec[1] = "-c"
	entrypointSpec[2] = entrypoint
	if len(builder.Cmd()) > 0 {
		logrus.Warnf("cmd %q exists but will be ignored because of entrypoint settings", strings.Join(builder.Cmd(), " "))
	}

	builder.SetEntrypoint(entrypointSpec)
}

func conditionallyAddHistory(builder *buildah.Builder, c *cobra.Command, createdByFmt string, args ...interface{}) {
	history := buildahcli.DefaultHistory()
	if c.Flag("add-history").Changed {
		history, _ = c.Flags().GetBool("add-history")
	}
	if history {
		now := time.Now().UTC()
		created := &now
		createdBy := fmt.Sprintf(createdByFmt, args...)
		builder.AddPrependedEmptyLayer(created, createdBy, "", "")
	}
}

func updateConfig(builder *buildah.Builder, c *cobra.Command, iopts configResults) {
	if c.Flag("author").Changed {
		builder.SetMaintainer(iopts.author)
	}
	if c.Flag("created-by").Changed {
		builder.SetCreatedBy(iopts.createdBy)
	}
	if c.Flag("arch").Changed {
		builder.SetArchitecture(iopts.arch)
	}
	if c.Flag("os").Changed {
		builder.SetOS(iopts.os)
	}
	if c.Flag("user").Changed {
		builder.SetUser(iopts.user)
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) USER %s", iopts.user)
	}
	if c.Flag("shell").Changed {
		shell := iopts.shell
		shellSpec, err := shellwords.Parse(shell)
		if err != nil {
			logrus.Errorf("error parsing --shell %q: %v", shell, err)
		} else {
			builder.SetShell(shellSpec)
		}
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) SHELL %s", shell)
	}
	if c.Flag("stop-signal").Changed {
		builder.SetStopSignal(iopts.stopSignal)
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) STOPSIGNAL %s", iopts.stopSignal)
	}
	if c.Flag("port").Changed {
		for _, portSpec := range iopts.ports {
			builder.SetPort(portSpec)
		}
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) EXPOSE %s", strings.Join(iopts.ports, " "))
	}
	if c.Flag("env").Changed {
		for _, envSpec := range iopts.env {
			env := strings.SplitN(envSpec, "=", 2)
			if len(env) > 1 {
				var unexpanded []string
				getenv := func(name string) string {
					for _, envvar := range builder.Env() {
						val := strings.SplitN(envvar, "=", 2)
						if len(val) == 2 && val[0] == name {
							return val[1]
						}
					}
					logrus.Errorf("error expanding variable %q: no value set in configuration", name)
					unexpanded = append(unexpanded, name)
					return name
				}
				env[1] = os.Expand(env[1], getenv)
				builder.SetEnv(env[0], env[1])
			} else {
				builder.UnsetEnv(env[0])
			}
		}
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) ENV %s", strings.Join(iopts.env, " "))
	}
	if c.Flag("entrypoint").Changed {
		updateEntrypoint(builder, iopts)
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) ENTRYPOINT %s", iopts.entrypoint)
	}
	// cmd should always run after entrypoint; setting entrypoint clears cmd
	if c.Flag("cmd").Changed {
		cmd := iopts.cmd
		cmdSpec, err := shellwords.Parse(cmd)
		if err != nil {
			logrus.Errorf("error parsing --cmd %q: %v", cmd, err)
		} else {
			builder.SetCmd(cmdSpec)
		}
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop)  CMD %s", cmd)
	}
	if c.Flag("volume").Changed {
		if volSpec := iopts.volume; len(volSpec) > 0 {
			for _, spec := range volSpec {
				builder.AddVolume(spec)
				conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) VOLUME %s", spec)
			}
		}
	}
	updateHealthcheck(builder, c, iopts)
	if c.Flag("label").Changed {
		for _, labelSpec := range iopts.label {
			label := strings.SplitN(labelSpec, "=", 2)
			if len(label) > 1 {
				builder.SetLabel(label[0], label[1])
			} else {
				builder.UnsetLabel(label[0])
			}
		}
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) LABEL %s", strings.Join(iopts.label, " "))
	}
	if c.Flag("workingdir").Changed {
		builder.SetWorkDir(iopts.workingDir)
		conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) WORKDIR %s", iopts.workingDir)
	}
	if c.Flag("comment").Changed {
		builder.SetComment(iopts.comment)
	}
	if c.Flag("history-comment").Changed {
		builder.SetHistoryComment(iopts.historyComment)
	}
	if c.Flag("domainname").Changed {
		builder.SetDomainname(iopts.domainName)
	}
	if c.Flag("hostname").Changed {
		name := iopts.hostname
		if name != "" && builder.Format == buildah.OCIv1ImageManifest {
			logrus.Errorf("HOSTNAME is not supported for OCI V1 image format, hostname %s will be ignored. Must use `docker` format", name)
		}
		builder.SetHostname(name)
	}
	if c.Flag("onbuild").Changed {
		fmt.Println("--------------->")
		for _, onbuild := range iopts.onbuild {
			builder.SetOnBuild(onbuild)
			conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) ONBUILD %s", onbuild)
		}
	}
	if c.Flag("annotation").Changed {
		for _, annotationSpec := range iopts.annotation {
			annotation := strings.SplitN(annotationSpec, "=", 2)
			if len(annotation) > 1 {
				builder.SetAnnotation(annotation[0], annotation[1])
			} else {
				builder.UnsetAnnotation(annotation[0])
			}
		}
	}
}

func updateHealthcheck(builder *buildah.Builder, c *cobra.Command, iopts configResults) {
	if c.Flag("healthcheck").Changed || c.Flag("healthcheck-interval").Changed || c.Flag("healthcheck-retries").Changed || c.Flag("healthcheck-start-period").Changed || c.Flag("healthcheck-timeout").Changed {
		healthcheck := builder.Healthcheck()
		args := ""
		if healthcheck == nil {
			healthcheck = &docker.HealthConfig{
				Test:        []string{"NONE"},
				Interval:    30 * time.Second,
				StartPeriod: 0,
				Timeout:     30 * time.Second,
				Retries:     3,
			}
		}
		if c.Flag("healthcheck").Changed {
			test, err := shellwords.Parse(iopts.healthcheck)
			if err != nil {
				logrus.Errorf("error parsing --healthcheck %q: %v", iopts.healthcheck, err)
			}
			healthcheck.Test = test
		}
		if c.Flag("healthcheck-interval").Changed {
			duration, err := time.ParseDuration(iopts.healthcheckInterval)
			if err != nil {
				logrus.Errorf("error parsing --healthcheck-interval %q: %v", iopts.healthcheckInterval, err)
			}
			healthcheck.Interval = duration
			args = args + "--interval=" + iopts.healthcheckInterval + " "
		}
		if c.Flag("healthcheck-retries").Changed {
			healthcheck.Retries = iopts.healthcheckRetries
			args = args + "--retries=" + strconv.Itoa(iopts.healthcheckRetries) + " "
			//args = fmt.Sprintf("%s --retries=%d ", args, iopts.healthcheckRetries)

		}
		if c.Flag("healthcheck-start-period").Changed {
			duration, err := time.ParseDuration(iopts.healthcheckStartPeriod)
			if err != nil {
				logrus.Errorf("error parsing --healthcheck-start-period %q: %v", iopts.healthcheckStartPeriod, err)
			}
			healthcheck.StartPeriod = duration
			args = args + "--start-period=" + iopts.healthcheckStartPeriod + " "
		}
		if c.Flag("healthcheck-timeout").Changed {
			duration, err := time.ParseDuration(iopts.healthcheckTimeout)
			if err != nil {
				logrus.Errorf("error parsing --healthcheck-timeout %q: %v", iopts.healthcheckTimeout, err)
			}
			healthcheck.Timeout = duration
			args = args + "--timeout=" + iopts.healthcheckTimeout + " "
		}
		if len(healthcheck.Test) == 0 {
			builder.SetHealthcheck(nil)
			conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) HEALTHCHECK NONE")
		} else {
			builder.SetHealthcheck(healthcheck)
			conditionallyAddHistory(builder, c, "/bin/sh -c #(nop) HEALTHCHECK %s%s", args, iopts.healthcheck)
		}
	}
}

func configCmd(c *cobra.Command, args []string, iopts configResults) error {
	if len(args) == 0 {
		return errors.Errorf("container ID must be specified")
	}
	if err := buildahcli.VerifyFlagsArgsOrder(args); err != nil {
		return err
	}
	if len(args) > 1 {
		return errors.Errorf("too many arguments specified")
	}
	name := args[0]

	store, err := getStore(c)
	if err != nil {
		return err
	}

	builder, err := openBuilder(getContext(), store, name)
	if err != nil {
		return errors.Wrapf(err, "error reading build container %q", name)
	}

	updateConfig(builder, c, iopts)
	return builder.Save()
}
