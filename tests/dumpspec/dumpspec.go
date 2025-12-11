package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/reexec"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// use defined names for our various commands. we absolutely don't support
// everything that an actual functional runtime would, and have no intention of
// expanding to do so
type modeType string

const (
	modeCreate  = modeType("create")
	modeStart   = modeType("start")
	modeState   = modeType("state")
	modeKill    = modeType("kill")
	modeDelete  = modeType("delete")
	subprocName = "dumpspec-subproc"
)

// signalsByName is a guess at which signals we'd be asked to send to a child
// process, currently restricted to the subset defined across all of our
// targets
var signalsByName = map[string]syscall.Signal{
	"SIGABRT": syscall.SIGABRT,
	"SIGALRM": syscall.SIGALRM,
	"SIGBUS":  syscall.SIGBUS,
	"SIGFPE":  syscall.SIGFPE,
	"SIGHUP":  syscall.SIGHUP,
	"SIGILL":  syscall.SIGILL,
	"SIGINT":  syscall.SIGINT,
	"SIGKILL": syscall.SIGKILL,
	"SIGPIPE": syscall.SIGPIPE,
	"SIGQUIT": syscall.SIGQUIT,
	"SIGSEGV": syscall.SIGSEGV,
	"SIGTERM": syscall.SIGTERM,
	"SIGTRAP": syscall.SIGTRAP,
}

var (
	globalArgs struct {
		debug         bool
		cgroupManager string
		log           string
		logFormat     string
		logLevel      string
		root          string
		systemdCgroup bool
		rootless      bool
	}
	createArgs struct {
		bundleDir     string
		configFile    string
		consoleSocket string
		pidFile       string
		noPivot       bool
		noNewKeyring  bool
		preserveFds   int
	}
	stateArgs struct {
		all   bool
		pid   int
		regex string
	}
	killArgs struct {
		all    bool
		pid    int
		regex  string
		signal int
	}
	deleteArgs struct {
		force bool
		regex string
	}
)

func main() {
	if reexec.Init() {
		return
	}

	if len(os.Args) < 2 {
		return
	}

	var container, containerID, containerDir string

	mainCommand := cobra.Command{
		Use:   "dumpspec",
		Short: "fake OCI runtime",
		PersistentPreRunE: func(_ *cobra.Command, args []string) error {
			tmpdir, ok := os.LookupEnv("XDG_RUNTIME_DIR")
			if !ok {
				tmpdir = filepath.Join(os.TempDir(), strconv.Itoa(os.Getuid()))
			}
			if globalArgs.root != "" {
				tmpdir = globalArgs.root
			}
			tmpdir = filepath.Join(tmpdir, "dumpspec")
			if err := os.MkdirAll(tmpdir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
				return fmt.Errorf("ensuring that %q exists: %w", tmpdir, err)
			}
			if len(args) > 0 {
				// this is the first arg for all of the commands that we care about
				container = args[0]
			}
			containerID = mapToContainerID(container)
			containerDir = filepath.Join(tmpdir, containerID)
			return nil
		},
	}
	mainFlags := mainCommand.PersistentFlags()
	mainFlags.BoolVar(&globalArgs.debug, "debug", false, "log for debugging")
	mainFlags.BoolVar(&globalArgs.systemdCgroup, "systemd-cgroup", false, "use systemd for handling cgroups")
	mainFlags.BoolVar(&globalArgs.rootless, "rootless", false, "ignore some settings to that conflict with rootless operation")
	mainFlags.StringVar(&globalArgs.cgroupManager, "cgroup-manager", "cgroupfs", "method for managing cgroups")
	mainFlags.StringVar(&globalArgs.log, "log", "", "logging destination")
	mainFlags.StringVar(&globalArgs.logFormat, "log-format", "", "logging format specifier")
	mainFlags.StringVar(&globalArgs.logLevel, "log-level", "", "logging level")
	rootUsage := "root `directory` of runtime data"
	rootDefault := ""
	if xdgRuntimeDir, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok {
		rootUsage += " (default $XDG_RUNTIME_DIR)"
		rootDefault = xdgRuntimeDir
	}
	mainFlags.StringVar(&globalArgs.root, "root", rootDefault, rootUsage)

	createCommand := &cobra.Command{
		Use:   string(modeCreate),
		Args:  cobra.ExactArgs(1),
		Short: "create a ready-to-start container process",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := os.MkdirAll(containerDir, 0o700); err != nil {
				return fmt.Errorf("creating container directory: %w", err)
			}
			configFile := createArgs.configFile
			if configFile == "" {
				configFile = filepath.Join(createArgs.bundleDir, "config.json")
			}
			config, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("reading runtime configuration: %w", err)
			}
			var spec rspec.Spec
			if err := json.Unmarshal(config, &spec); err != nil {
				return fmt.Errorf("parsing runtime configuration: %w", err)
			}
			if err := os.WriteFile(filepath.Join(containerDir, "config.json"), config, 0o600); err != nil {
				return fmt.Errorf("saving copy of runtime configuration: %w", err)
			}
			state := rspec.State{
				Version: rspec.Version,
				ID:      container,
				Bundle:  createArgs.bundleDir,
			}
			stateBytes, err := json.Marshal(state)
			if err != nil {
				return fmt.Errorf("encoding initial runtime state: %w", err)
			}
			if err := os.WriteFile(filepath.Join(containerDir, "state"), stateBytes, 0o600); err != nil {
				return fmt.Errorf("writing initial runtime state: %w", err)
			}
			pr, pw, err := os.Pipe()
			if err != nil {
				return fmt.Errorf("internal error: %w", err)
			}
			defer pr.Close()
			cmd := getStarter(containerDir, createArgs.consoleSocket, createArgs.pidFile, spec, pw)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("internal error: %w", err)
			}
			pw.Close()
			ready, err := io.ReadAll(pr)
			if err != nil {
				return fmt.Errorf("waiting for child to start: %w", err)
			}
			if strings.TrimSpace(string(ready)) != "OK" {
				return fmt.Errorf("unexpected child status %q", string(ready))
			}
			return nil
		},
	}
	createFlags := createCommand.Flags()
	createFlags.StringVarP(&createArgs.bundleDir, "bundle", "b", "", "`directory` containing config.json")
	createFlags.StringVarP(&createArgs.configFile, "config", "f", "", "`path` to config.json")
	createFlags.StringVar(&createArgs.consoleSocket, "console-socket", "", "socket `path` for passing PTY")
	createFlags.StringVar(&createArgs.pidFile, "pid-file", "", "`path` in which to store child PID")
	createFlags.BoolVar(&createArgs.noPivot, "no-pivot", false, "use chroot() instead of pivot_root()")
	createFlags.BoolVar(&createArgs.noNewKeyring, "no-new-keyring", false, "don't create a new keyring")
	mainCommand.AddCommand(createCommand)

	startCommand := &cobra.Command{
		Use:   string(modeStart),
		Args:  cobra.ExactArgs(1),
		Short: "start a previously-created container process",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := ioutils.AtomicWriteFile(filepath.Join(containerDir, "start"), []byte("start"), 0o600); err != nil {
				return fmt.Errorf("writing start file: %w", err)
			}
			return nil
		},
	}
	mainCommand.AddCommand(startCommand)

	stateCommand := &cobra.Command{
		Use:   string(modeState),
		Args:  cobra.ExactArgs(1),
		Short: "poll the state of a container process",
		RunE: func(_ *cobra.Command, _ []string) error {
			stateFile, err := os.Open(filepath.Join(containerDir, "state"))
			if err != nil {
				return err
			}
			defer stateFile.Close()
			if _, err := io.Copy(os.Stdout, stateFile); err != nil {
				return fmt.Errorf("copying state file: %w", err)
			}
			return nil
		},
	}
	stateFlags := stateCommand.Flags()
	stateFlags.BoolVarP(&stateArgs.all, "all", "a", false, "start all containers")
	stateFlags.IntVar(&stateArgs.pid, "pid", 0, "start container by `pid`")
	stateFlags.StringVarP(&stateArgs.regex, "regex", "r", "", "start containers with IDs matching a `regex`")
	mainCommand.AddCommand(stateCommand)

	killCommand := &cobra.Command{
		Use:   string(modeKill),
		Args:  cobra.RangeArgs(1, 2),
		Short: "signal/kill a container process",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 1 {
				signalString := args[1]
				signalNumber, err := strconv.Atoi(signalString)
				if err != nil {
					n, ok := signalsByName[signalString]
					if !ok {
						n, ok = signalsByName["SIG"+signalString]
						if !ok {
							return fmt.Errorf("%v: unrecognized signal %q", os.Args, signalString)
						}
					}
					signalNumber = int(n)
				}
				killArgs.signal = signalNumber
			}
			if err := ioutils.AtomicWriteFile(filepath.Join(containerDir, "kill"), []byte(strconv.Itoa(killArgs.signal)), 0o600); err != nil {
				return fmt.Errorf("writing exit status file: %w", err)
			}
			return nil
		},
	}
	killFlags := killCommand.Flags()
	killFlags.BoolVarP(&killArgs.all, "all", "a", false, "signal/kill all containers")
	killFlags.IntVar(&killArgs.pid, "pid", 0, "signal/kill container by `pid`")
	killFlags.StringVarP(&killArgs.regex, "regex", "r", "", "signal/kill containers with IDs matching a `regex`")
	mainCommand.AddCommand(killCommand)

	deleteCommand := &cobra.Command{
		Use:   string(modeDelete),
		Args:  cobra.ExactArgs(1),
		Short: "delete a container process",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := os.RemoveAll(containerDir); err != nil {
				return fmt.Errorf("removing container directory: %w", err)
			}
			return nil
		},
	}
	deleteFlags := deleteCommand.Flags()
	deleteFlags.StringVarP(&deleteArgs.regex, "regex", "r", "", "delete containers with IDs matching a `regex`")
	deleteFlags.BoolVarP(&deleteArgs.force, "force", "f", false, "forcibly stop containers which are not stopped")
	mainCommand.AddCommand(deleteCommand)

	err := mainCommand.Execute()
	if err != nil {
		logrus.Fatal(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func mapToContainerID(container string) string {
	var encoder strings.Builder
	for _, c := range container {
		if unicode.IsLetter(c) || unicode.IsNumber(c) {
			if _, err := encoder.WriteRune(c); err != nil {
				logrus.Fatalf("%v: encoding container ID: %q: %v", os.Args, c, err)
			}
		} else {
			if _, err := encoder.WriteString(strconv.Itoa(int(c))); err != nil {
				logrus.Fatalf("%v: encoding container ID: %q: %v", os.Args, c, err)
			}
		}
	}
	return encoder.String()
}

func waitForFile(dirname, basename string) string {
	waitedFile := filepath.Join(dirname, basename)
	for {
		if _, err := os.Stat(dirname); err != nil {
			logrus.Fatalf("%v: %v", os.Args, err)
		}
		st, err := os.Stat(waitedFile)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			logrus.Fatalf("%v: %v", os.Args, err)
		}
		if err != nil || st.Size() == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		contents, err := os.ReadFile(waitedFile)
		if err != nil {
			logrus.Fatalf("%v: %v", os.Args, err)
		}
		text := strings.TrimSpace(string(contents))
		return text
	}
}

func init() {
	reexec.Register(subprocName, subproc)
}

func subproc() {
	mainCommand := cobra.Command{
		Use:   "dumpspec",
		Short: "fake OCI runtime",
		Long:  "dumpspec containerDir consoleSocket pidFile [spec ...]",
		Args:  cobra.ExactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			dir := args[0]
			consoleSocket := args[1]
			pidFile := args[2]

			config, err := os.ReadFile(filepath.Join(dir, "config.json"))
			if err != nil {
				return fmt.Errorf("reading runtime configuration: %w", err)
			}
			var spec rspec.Spec
			if err := json.Unmarshal(config, &spec); err != nil {
				return fmt.Errorf("parsing runtime configuration: %w", err)
			}

			stateBytes, err := os.ReadFile(filepath.Join(dir, "state"))
			if err != nil {
				return fmt.Errorf("reading initial state : %w", err)
			}
			var state rspec.State
			if err := json.Unmarshal(stateBytes, &state); err != nil {
				return fmt.Errorf("parsing initial state: %w", err)
			}

			saveState := func() error {
				stateBytes, err := json.Marshal(state)
				if err != nil {
					return fmt.Errorf("encoding updated state: %w", err)
				}
				err = ioutils.AtomicWriteFile(filepath.Join(dir, "state"), stateBytes, 0o600)
				if err != nil {
					return fmt.Errorf("writing updated state: %w", err)
				}
				return nil
			}

			output := io.Writer(os.Stdout)

			if pidFile != "" {
				if err := ioutils.AtomicWriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
					return fmt.Errorf("writing pid file %q: %w", pidFile, err)
				}
			}

			state.Pid = os.Getpid()
			state.Status = rspec.StateCreated
			if err := saveState(); err != nil {
				return err
			}

			if consoleSocket != "" {
				if output, err = sendConsoleDescriptor(consoleSocket); err != nil {
					return fmt.Errorf("sending terminal control fd to parent process: %w", err)
				}
			}

			ok := os.NewFile(3, "startup status pipe")
			fmt.Fprintf(ok, "OK")
			ok.Close()

			start := waitForFile(dir, "start")
			if start != "start" {
				return fmt.Errorf("unexpected start indicator %q", start)
			}

			state.Status = rspec.StateRunning
			if err := saveState(); err != nil {
				return err
			}

			if spec.Process == nil || len(spec.Process.Args) == 0 {
				if _, err := io.Copy(output, bytes.NewReader(config)); err != nil {
					return fmt.Errorf("writing configuration: %w", err)
				}
			} else {
				for _, query := range spec.Process.Args {
					var data any
					if err := json.Unmarshal(config, &data); err != nil {
						return fmt.Errorf("parsing runtime configuration: %w", err)
					}
					path := strings.Split(query, ".")
					for i, component := range path {
						if component == "" {
							continue
						}
						pathSoFar := strings.Join(path[:i], ".")
						if data == nil {
							return fmt.Errorf("unable to descend into %q after %q", component, pathSoFar)
						}
						if m, ok := data.(map[string]any); ok {
							data = m[component]
						} else if s, ok := data.([]any); ok {
							i, err := strconv.Atoi(component)
							if err != nil {
								return fmt.Errorf("%q is not numeric while indexing slice at %q", component, pathSoFar)
							}
							data = s[i]
						} else {
							return fmt.Errorf("unable to descend into %q after %q", component, pathSoFar)
						}
					}
					final, err := json.Marshal(data)
					if err != nil {
						return fmt.Errorf("encoding query result: %w", err)
					}
					if len(final) == 0 || final[len(final)-1] != '\n' {
						final = append(final, byte('\n'))
					}
					if _, err := io.Copy(output, bytes.NewReader(final)); err != nil {
						return fmt.Errorf("writing configuration subset %q: %w", query, err)
					}
				}
			}

			state.Status = rspec.StateStopped
			if err := saveState(); err != nil {
				return err
			}
			return nil
		},
	}
	err := mainCommand.Execute()
	if err != nil {
		logrus.Fatal(err)
		os.Exit(1)
	}
	os.Exit(0)
}
