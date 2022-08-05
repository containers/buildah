//go:build freebsd
// +build freebsd

package chroot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/containers/buildah/bind"
	"github.com/containers/buildah/pkg/jail"
	"github.com/containers/storage/pkg/mount"
	"github.com/containers/storage/pkg/unshare"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

var (
	rlimitsMap = map[string]int{
		"RLIMIT_AS":      unix.RLIMIT_AS,
		"RLIMIT_CORE":    unix.RLIMIT_CORE,
		"RLIMIT_CPU":     unix.RLIMIT_CPU,
		"RLIMIT_DATA":    unix.RLIMIT_DATA,
		"RLIMIT_FSIZE":   unix.RLIMIT_FSIZE,
		"RLIMIT_MEMLOCK": unix.RLIMIT_MEMLOCK,
		"RLIMIT_NOFILE":  unix.RLIMIT_NOFILE,
		"RLIMIT_NPROC":   unix.RLIMIT_NPROC,
		"RLIMIT_RSS":     unix.RLIMIT_RSS,
		"RLIMIT_STACK":   unix.RLIMIT_STACK,
	}
	rlimitsReverseMap = map[int]string{}
)

type runUsingChrootSubprocOptions struct {
	Spec       *specs.Spec
	BundlePath string
}

// runUsingChroot, still in the grandparent process, sets up various bind
// mounts and then runs the parent process in its own user namespace with the
// necessary ID mappings.
func runUsingChroot(spec *specs.Spec, bundlePath string, ctty *os.File, stdin io.Reader, stdout, stderr io.Writer, closeOnceRunning []*os.File) (wstatus unix.WaitStatus, err error) {
	var confwg sync.WaitGroup

	// Create a new mount namespace for ourselves and bind mount everything to a new location.
	undoIntermediates, err := bind.SetupIntermediateMountNamespace(spec, bundlePath)
	if err != nil {
		return 1, err
	}
	defer func() {
		if undoErr := undoIntermediates(); undoErr != nil {
			logrus.Debugf("error cleaning up intermediate mount NS: %v", err)
		}
	}()

	// Bind mount in our filesystems.
	undoChroots, err := setupChrootBindMounts(spec, bundlePath)
	if err != nil {
		return 1, err
	}
	defer func() {
		if undoErr := undoChroots(); undoErr != nil {
			logrus.Debugf("error cleaning up intermediate chroot bind mounts: %v", err)
		}
	}()

	// Create a pipe for passing configuration down to the next process.
	preader, pwriter, err := os.Pipe()
	if err != nil {
		return 1, fmt.Errorf("error creating configuration pipe: %w", err)
	}
	config, conferr := json.Marshal(runUsingChrootExecSubprocOptions{
		Spec:       spec,
		BundlePath: bundlePath,
	})
	if conferr != nil {
		fmt.Fprintf(os.Stderr, "error re-encoding configuration for %q", runUsingChrootExecCommand)
		os.Exit(1)
	}

	// Start the parent subprocess.
	cmd := unshare.Command(append([]string{runUsingChrootExecCommand}, spec.Process.Args...)...)
	setPdeathsig(cmd.Cmd)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	cmd.Dir = "/"
	cmd.Env = []string{fmt.Sprintf("LOGLEVEL=%d", logrus.GetLevel())}
	if ctty != nil {
		cmd.Setsid = true
		cmd.Ctty = ctty
	}
	cmd.ExtraFiles = append([]*os.File{preader}, cmd.ExtraFiles...)
	interrupted := make(chan os.Signal, 100)
	cmd.Hook = func(int) error {
		for _, f := range closeOnceRunning {
			f.Close()
		}
		signal.Notify(interrupted, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			for receivedSignal := range interrupted {
				if err := cmd.Process.Signal(receivedSignal); err != nil {
					logrus.Infof("%v while attempting to forward %v to child process", err, receivedSignal)
				}
			}
		}()
		return nil
	}

	logrus.Debugf("Running %#v in %#v", cmd.Cmd, cmd)
	confwg.Add(1)
	go func() {
		_, conferr = io.Copy(pwriter, bytes.NewReader(config))
		pwriter.Close()
		confwg.Done()
	}()
	err = cmd.Run()
	confwg.Wait()
	signal.Stop(interrupted)
	close(interrupted)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if waitStatus, ok := exitError.ProcessState.Sys().(syscall.WaitStatus); ok {
				if waitStatus.Exited() {
					if waitStatus.ExitStatus() != 0 {
						fmt.Fprintf(os.Stderr, "subprocess exited with status %d\n", waitStatus.ExitStatus())
					}
					os.Exit(waitStatus.ExitStatus())
				} else if waitStatus.Signaled() {
					fmt.Fprintf(os.Stderr, "subprocess exited on %s\n", waitStatus.Signal())
					os.Exit(1)
				}
			}
		}
		fmt.Fprintf(os.Stderr, "process exited with error: %v", err)
		os.Exit(1)
	}

	return 0, nil
}

func createJail(options runUsingChrootExecSubprocOptions) error {
	path := options.Spec.Root.Path
	jconf := jail.NewConfig()
	jconf.Set("name", filepath.Base(path)+"-chroot")
	jconf.Set("host.hostname", options.Spec.Hostname)
	jconf.Set("persist", false)
	jconf.Set("path", path)
	jconf.Set("ip4", jail.INHERIT)
	jconf.Set("ip6", jail.INHERIT)
	jconf.Set("allow.raw_sockets", true)
	jconf.Set("enforce_statfs", 1)
	_, err := jail.CreateAndAttach(jconf)
	if err != nil {
		return fmt.Errorf("error creating jail: %w", err)
	}
	return nil
}

// main() for parent subprocess.  Its main job is to try to make our
// environment look like the one described by the runtime configuration blob,
// and then launch the intended command as a child.
func runUsingChrootExecMain() {
	args := os.Args[1:]
	var options runUsingChrootExecSubprocOptions
	var err error

	runtime.LockOSThread()

	// Set logging.
	if level := os.Getenv("LOGLEVEL"); level != "" {
		if ll, err := strconv.Atoi(level); err == nil {
			logrus.SetLevel(logrus.Level(ll))
		}
		os.Unsetenv("LOGLEVEL")
	}

	// Unpack our configuration.
	confPipe := os.NewFile(3, "confpipe")
	if confPipe == nil {
		fmt.Fprintf(os.Stderr, "error reading options pipe\n")
		os.Exit(1)
	}
	defer confPipe.Close()
	if err := json.NewDecoder(confPipe).Decode(&options); err != nil {
		fmt.Fprintf(os.Stderr, "error decoding options: %v\n", err)
		os.Exit(1)
	}

	// Set the hostname.  We're already in a distinct UTS namespace and are admins in the user
	// namespace which created it, so we shouldn't get a permissions error, but seccomp policy
	// might deny our attempt to call sethostname() anyway, so log a debug message for that.
	if options.Spec == nil || options.Spec.Process == nil {
		fmt.Fprintf(os.Stderr, "invalid options spec passed in\n")
		os.Exit(1)
	}

	/*if options.Spec.Hostname != "" {
		if err := unix.Sethostname([]byte(options.Spec.Hostname)); err != nil {
			logrus.Debugf("failed to set hostname %q for process: %v", options.Spec.Hostname, err)
		}
	}*/

	// Try to create a jail and if that fails, fall back to chroot
	if err := createJail(options); err == nil {
		logrus.Debugf("jailed into %q", options.Spec.Root.Path)
	} else {
		// Try to chroot into the root.  Do this before we potentially block the syscall via the
		// seccomp profile.
		var oldst, newst unix.Stat_t
		if err := unix.Stat(options.Spec.Root.Path, &oldst); err != nil {
			fmt.Fprintf(os.Stderr, "error stat()ing intended root directory %q: %v\n", options.Spec.Root.Path, err)
			os.Exit(1)
		}
		if err := unix.Chdir(options.Spec.Root.Path); err != nil {
			fmt.Fprintf(os.Stderr, "error chdir()ing to intended root directory %q: %v\n", options.Spec.Root.Path, err)
			os.Exit(1)
		}
		if err := unix.Chroot(options.Spec.Root.Path); err != nil {
			fmt.Fprintf(os.Stderr, "error chroot()ing into directory %q: %v\n", options.Spec.Root.Path, err)
			os.Exit(1)
		}
		if err := unix.Stat("/", &newst); err != nil {
			fmt.Fprintf(os.Stderr, "error stat()ing current root directory: %v\n", err)
			os.Exit(1)
		}
		if oldst.Dev != newst.Dev || oldst.Ino != newst.Ino {
			fmt.Fprintf(os.Stderr, "unknown error chroot()ing into directory %q: %v\n", options.Spec.Root.Path, err)
			os.Exit(1)
		}
		logrus.Debugf("chrooted into %q", options.Spec.Root.Path)
	}

	// not doing because it's still shared: creating devices
	// not doing because it's not applicable: setting annotations
	// not doing because it's still shared: setting sysctl settings
	// not doing because cgroupfs is read only: configuring control groups
	// -> this means we can use the freezer to make sure there aren't any lingering processes
	// -> this means we ignore cgroups-based controls
	// not doing because we don't set any in the config: running hooks
	// not doing because we don't set it in the config: setting rootfs read-only
	// not doing because we don't set it in the config: setting rootfs propagation
	logrus.Debugf("setting resource limits")
	if err = setRlimits(options.Spec, false, false); err != nil {
		fmt.Fprintf(os.Stderr, "error setting process resource limits for process: %v\n", err)
		os.Exit(1)
	}

	// Try to change to the directory.
	cwd := options.Spec.Process.Cwd
	if !filepath.IsAbs(cwd) {
		cwd = "/" + cwd
	}
	cwd = filepath.Clean(cwd)
	if err := unix.Chdir("/"); err != nil {
		fmt.Fprintf(os.Stderr, "error chdir()ing into new root directory %q: %v\n", options.Spec.Root.Path, err)
		os.Exit(1)
	}
	if err := unix.Chdir(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "error chdir()ing into directory %q under root %q: %v\n", cwd, options.Spec.Root.Path, err)
		os.Exit(1)
	}
	logrus.Debugf("changed working directory to %q", cwd)

	// Drop privileges.
	user := options.Spec.Process.User
	if len(user.AdditionalGids) > 0 {
		gids := make([]int, len(user.AdditionalGids))
		for i := range user.AdditionalGids {
			gids[i] = int(user.AdditionalGids[i])
		}
		logrus.Debugf("setting supplemental groups")
		if err = syscall.Setgroups(gids); err != nil {
			fmt.Fprintf(os.Stderr, "error setting supplemental groups list: %v", err)
			os.Exit(1)
		}
	} else {
		setgroups, _ := ioutil.ReadFile("/proc/self/setgroups")
		if strings.Trim(string(setgroups), "\n") != "deny" {
			logrus.Debugf("clearing supplemental groups")
			if err = syscall.Setgroups([]int{}); err != nil {
				fmt.Fprintf(os.Stderr, "error clearing supplemental groups list: %v", err)
				os.Exit(1)
			}
		}
	}

	logrus.Debugf("setting gid")
	if err = unix.Setresgid(int(user.GID), int(user.GID), int(user.GID)); err != nil {
		fmt.Fprintf(os.Stderr, "error setting GID: %v", err)
		os.Exit(1)
	}

	logrus.Debugf("setting uid")
	if err = unix.Setresuid(int(user.UID), int(user.UID), int(user.UID)); err != nil {
		fmt.Fprintf(os.Stderr, "error setting UID: %v", err)
		os.Exit(1)
	}

	// Actually run the specified command.
	cmd := exec.Command(args[0], args[1:]...)
	setPdeathsig(cmd)
	cmd.Env = options.Spec.Process.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Dir = cwd
	logrus.Debugf("Running %#v (PATH = %q)", cmd, os.Getenv("PATH"))
	interrupted := make(chan os.Signal, 100)
	if err = cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "process failed to start with error: %v", err)
	}
	go func() {
		for range interrupted {
			if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
				logrus.Infof("%v while attempting to send SIGKILL to child process", err)
			}
		}
	}()
	signal.Notify(interrupted, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	err = cmd.Wait()
	signal.Stop(interrupted)
	close(interrupted)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if waitStatus, ok := exitError.ProcessState.Sys().(syscall.WaitStatus); ok {
				if waitStatus.Exited() {
					if waitStatus.ExitStatus() != 0 {
						fmt.Fprintf(os.Stderr, "subprocess exited with status %d\n", waitStatus.ExitStatus())
					}
					os.Exit(waitStatus.ExitStatus())
				} else if waitStatus.Signaled() {
					fmt.Fprintf(os.Stderr, "subprocess exited on %s\n", waitStatus.Signal())
					os.Exit(1)
				}
			}
		}
		fmt.Fprintf(os.Stderr, "process exited with error: %v", err)
		os.Exit(1)
	}
}

// parses the resource limits for ourselves and any processes that
// we'll start into a format that's more in line with the kernel APIs
func parseRlimits(spec *specs.Spec) (map[int]unix.Rlimit, error) {
	if spec.Process == nil {
		return nil, nil
	}
	parsed := make(map[int]unix.Rlimit)
	for _, limit := range spec.Process.Rlimits {
		resource, recognized := rlimitsMap[strings.ToUpper(limit.Type)]
		if !recognized {
			return nil, fmt.Errorf("error parsing limit type %q", limit.Type)
		}
		parsed[resource] = unix.Rlimit{Cur: int64(limit.Soft), Max: int64(limit.Hard)}
	}
	return parsed, nil
}

// setRlimits sets any resource limits that we want to apply to processes that
// we'll start.
func setRlimits(spec *specs.Spec, onlyLower, onlyRaise bool) error {
	limits, err := parseRlimits(spec)
	if err != nil {
		return err
	}
	for resource, desired := range limits {
		var current unix.Rlimit
		if err := unix.Getrlimit(resource, &current); err != nil {
			return fmt.Errorf("error reading %q limit: %w", rlimitsReverseMap[resource], err)
		}
		if desired.Max > current.Max && onlyLower {
			// this would raise a hard limit, and we're only here to lower them
			continue
		}
		if desired.Max < current.Max && onlyRaise {
			// this would lower a hard limit, and we're only here to raise them
			continue
		}
		if err := unix.Setrlimit(resource, &desired); err != nil {
			return fmt.Errorf("error setting %q limit to soft=%d,hard=%d (was soft=%d,hard=%d): %w", rlimitsReverseMap[resource], desired.Cur, desired.Max, current.Cur, current.Max, err)
		}
	}
	return nil
}

func makeReadOnly(mntpoint string, flags uintptr) error {
	var fs unix.Statfs_t
	// Make sure it's read-only.
	if err := unix.Statfs(mntpoint, &fs); err != nil {
		return fmt.Errorf("error checking if directory %q was bound read-only: %w", mntpoint, err)
	}
	return nil
}

func isDevNull(dev os.FileInfo) bool {
	if dev.Mode()&os.ModeCharDevice != 0 {
		stat, _ := dev.Sys().(*syscall.Stat_t)
		nullStat := syscall.Stat_t{}
		if err := syscall.Stat(os.DevNull, &nullStat); err != nil {
			logrus.Warnf("unable to stat /dev/null: %v", err)
			return false
		}
		if stat.Rdev == nullStat.Rdev {
			return true
		}
	}
	return false
}

func saveDir(spec *specs.Spec, path string) string {
	id := filepath.Base(spec.Root.Path)
	return filepath.Join(filepath.Dir(path), ".save-"+id)
}

func copyFile(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

type rename struct {
	from, to string
}

// setupChrootBindMounts actually bind mounts things under the rootfs, and returns a
// callback that will clean up its work.
func setupChrootBindMounts(spec *specs.Spec, bundlePath string) (undoBinds func() error, err error) {
	renames := []rename{}
	unmounts := []string{}
	removes := []string{}
	undoBinds = func() error {
		for _, r := range renames {
			if err2 := os.Rename(r.to, r.from); err2 != nil {
				logrus.Warnf("pkg/chroot: error renaming %q to %q: %v", r.to, r.from, err2)
				if err == nil {
					err = err2
				}
			}
		}
		for _, path := range unmounts {
			if err2 := mount.Unmount(path); err2 != nil {
				logrus.Warnf("pkg/chroot: error unmounting %q: %v", spec.Root.Path, err2)
				if err == nil {
					err = err2
				}
			}
		}
		for _, path := range removes {
			if err2 := os.Remove(path); err2 != nil {
				logrus.Warnf("pkg/chroot: error removing %q: %v", path, err2)
				if err == nil {
					err = err2
				}
			}
		}
		return err
	}

	// Now mount all of those things to be under the rootfs's location in this
	// mount namespace.
	for _, m := range spec.Mounts {
		// If the target is there, we can just mount it.
		var srcinfo os.FileInfo
		switch m.Type {
		case "nullfs":
			srcinfo, err = os.Stat(m.Source)
			if err != nil {
				return undoBinds, fmt.Errorf("error examining %q for mounting in mount namespace: %w", m.Source, err)
			}
		}
		target := filepath.Join(spec.Root.Path, m.Destination)
		if _, err := os.Stat(target); err != nil {
			// If the target can't be stat()ted, check the error.
			if !os.IsNotExist(err) {
				return undoBinds, fmt.Errorf("error examining %q for mounting in mount namespace: %w", target, err)
			}
			// The target isn't there yet, so create it, and make a
			// note to remove it later.
			// XXX: This was copied from the linux version which supports bind mounting files.
			// Leaving it here since I plan to add this to FreeBSD's nullfs.
			if m.Type != "nullfs" || srcinfo.IsDir() {
				if err = os.MkdirAll(target, 0111); err != nil {
					return undoBinds, fmt.Errorf("error creating mountpoint %q in mount namespace: %w", target, err)
				}
				removes = append(removes, target)
			} else {
				if err = os.MkdirAll(filepath.Dir(target), 0111); err != nil {
					return undoBinds, fmt.Errorf("error ensuring parent of mountpoint %q (%q) is present in mount namespace: %w", target, filepath.Dir(target), err)
				}
				// Don't do this until we can support file mounts in nullfs
				/*var file *os.File
				if file, err = os.OpenFile(target, os.O_WRONLY|os.O_CREATE, 0); err != nil {
					return undoBinds, errors.Wrapf(err, "error creating mountpoint %q in mount namespace", target)
				}
				file.Close()
				removes = append(removes, target)*/
			}
		}
		logrus.Debugf("mount: %v", m)
		switch m.Type {
		case "nullfs":
			// Do the bind mount.
			if !srcinfo.IsDir() {
				logrus.Debugf("emulating file mount %q on %q", m.Source, target)
				_, err := os.Stat(target)
				if err == nil {
					save := saveDir(spec, target)
					if _, err := os.Stat(save); err != nil {
						if os.IsNotExist(err) {
							err = os.MkdirAll(save, 0111)
						}
						if err != nil {
							return undoBinds, fmt.Errorf("error creating file mount save directory %q: %w", save, err)
						}
						removes = append(removes, save)
					}
					savePath := filepath.Join(save, filepath.Base(target))
					if _, err := os.Stat(target); err == nil {
						logrus.Debugf("moving %q to %q", target, savePath)
						if err := os.Rename(target, savePath); err != nil {
							return undoBinds, fmt.Errorf("error moving %q to %q: %w", target, savePath, err)
						}
						renames = append(renames, rename{
							from: target,
							to:   savePath,
						})
					}
				} else {
					removes = append(removes, target)
				}
				if err := copyFile(m.Source, target); err != nil {
					return undoBinds, fmt.Errorf("error copying %q to %q: %w", m.Source, target, err)
				}
			} else {
				logrus.Debugf("bind mounting %q on %q", m.Destination, filepath.Join(spec.Root.Path, m.Destination))
				if err := mount.Mount(m.Source, target, "nullfs", strings.Join(m.Options, ",")); err != nil {
					return undoBinds, fmt.Errorf("error bind mounting %q from host to %q in mount namespace (%q): %w", m.Source, m.Destination, target, err)
				}
				logrus.Debugf("bind mounted %q to %q", m.Source, target)
				unmounts = append(unmounts, target)
			}
		case "devfs", "fdescfs", "tmpfs":
			// Mount /dev, /dev/fd.
			if err := mount.Mount(m.Source, target, m.Type, strings.Join(m.Options, ",")); err != nil {
				return undoBinds, fmt.Errorf("error mounting %q to %q in mount namespace (%q, %q): %w", m.Type, m.Destination, target, strings.Join(m.Options, ","), err)
			}
			logrus.Debugf("mounted a %q to %q", m.Type, target)
			unmounts = append(unmounts, target)
		}
	}
	return undoBinds, nil
}

// setPdeathsig sets a parent-death signal for the process
func setPdeathsig(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL
}
