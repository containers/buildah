//go:build linux || freebsd
// +build linux freebsd

package chroot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/containers/buildah/util"
	"github.com/containers/storage/pkg/ioutils"
	"github.com/containers/storage/pkg/reexec"
	"github.com/containers/storage/pkg/unshare"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	// runUsingChrootCommand is a command we use as a key for reexec
	runUsingChrootCommand = "buildah-chroot-runtime"
	// runUsingChrootExec is a command we use as a key for reexec
	runUsingChrootExecCommand = "buildah-chroot-exec"
)

func init() {
	reexec.Register(runUsingChrootCommand, runUsingChrootMain)
	reexec.Register(runUsingChrootExecCommand, runUsingChrootExecMain)
	for limitName, limitNumber := range rlimitsMap {
		rlimitsReverseMap[limitNumber] = limitName
	}
}

type runUsingChrootExecSubprocOptions struct {
	Spec       *specs.Spec
	BundlePath string
}

// RunUsingChroot runs a chrooted process, using some of the settings from the
// passed-in spec, and using the specified bundlePath to hold temporary files,
// directories, and mountpoints.
func RunUsingChroot(spec *specs.Spec, bundlePath, homeDir string, stdin io.Reader, stdout, stderr io.Writer) (err error) {
	var confwg sync.WaitGroup
	var homeFound bool
	for _, env := range spec.Process.Env {
		if strings.HasPrefix(env, "HOME=") {
			homeFound = true
			break
		}
	}
	if !homeFound {
		spec.Process.Env = append(spec.Process.Env, fmt.Sprintf("HOME=%s", homeDir))
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Write the runtime configuration, mainly for debugging.
	specbytes, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	if err = ioutils.AtomicWriteFile(filepath.Join(bundlePath, "config.json"), specbytes, 0600); err != nil {
		return fmt.Errorf("error storing runtime configuration: %w", err)
	}
	logrus.Debugf("config = %v", string(specbytes))

	// Default to using stdin/stdout/stderr if we weren't passed objects to use.
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	// Create a pipe for passing configuration down to the next process.
	preader, pwriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creating configuration pipe: %w", err)
	}
	config, conferr := json.Marshal(runUsingChrootSubprocOptions{
		Spec:       spec,
		BundlePath: bundlePath,
	})
	if conferr != nil {
		return fmt.Errorf("error encoding configuration for %q: %w", runUsingChrootCommand, conferr)
	}

	// Set our terminal's mode to raw, to pass handling of special
	// terminal input to the terminal in the container.
	if spec.Process.Terminal && term.IsTerminal(unix.Stdin) {
		state, err := term.MakeRaw(unix.Stdin)
		if err != nil {
			logrus.Warnf("error setting terminal state: %v", err)
		} else {
			defer func() {
				if err = term.Restore(unix.Stdin, state); err != nil {
					logrus.Errorf("unable to restore terminal state: %v", err)
				}
			}()
		}
	}

	// Raise any resource limits that are higher than they are now, before
	// we drop any more privileges.
	if err = setRlimits(spec, false, true); err != nil {
		return err
	}

	// Start the grandparent subprocess.
	cmd := unshare.Command(runUsingChrootCommand)
	setPdeathsig(cmd.Cmd)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
	cmd.Dir = "/"
	cmd.Env = []string{fmt.Sprintf("LOGLEVEL=%d", logrus.GetLevel())}

	interrupted := make(chan os.Signal, 100)
	cmd.Hook = func(int) error {
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
	cmd.ExtraFiles = append([]*os.File{preader}, cmd.ExtraFiles...)
	err = cmd.Run()
	confwg.Wait()
	signal.Stop(interrupted)
	close(interrupted)
	if err == nil {
		return conferr
	}
	return err
}

// main() for grandparent subprocess.  Its main job is to shuttle stdio back
// and forth, managing a pseudo-terminal if we want one, for our child, the
// parent subprocess.
func runUsingChrootMain() {
	var options runUsingChrootSubprocOptions

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

	if options.Spec == nil || options.Spec.Process == nil {
		fmt.Fprintf(os.Stderr, "invalid options spec in runUsingChrootMain\n")
		os.Exit(1)
	}

	// Prepare to shuttle stdio back and forth.
	rootUID32, rootGID32, err := util.GetHostRootIDs(options.Spec)
	if err != nil {
		logrus.Errorf("error determining ownership for container stdio")
		os.Exit(1)
	}
	rootUID := int(rootUID32)
	rootGID := int(rootGID32)
	relays := make(map[int]int)
	closeOnceRunning := []*os.File{}
	var ctty *os.File
	var stdin io.Reader
	var stdinCopy io.WriteCloser
	var stdout io.Writer
	var stderr io.Writer
	fdDesc := make(map[int]string)
	if options.Spec.Process.Terminal {
		ptyMasterFd, ptyFd, err := getPtyDescriptors()
		if err != nil {
			logrus.Errorf("error opening PTY descriptors: %v", err)
			os.Exit(1)
		}
		// Make notes about what's going where.
		relays[ptyMasterFd] = unix.Stdout
		relays[unix.Stdin] = ptyMasterFd
		fdDesc[ptyMasterFd] = "container terminal"
		fdDesc[unix.Stdin] = "stdin"
		fdDesc[unix.Stdout] = "stdout"
		winsize := &unix.Winsize{}
		// Set the pseudoterminal's size to the configured size, or our own.
		if options.Spec.Process.ConsoleSize != nil {
			// Use configured sizes.
			winsize.Row = uint16(options.Spec.Process.ConsoleSize.Height)
			winsize.Col = uint16(options.Spec.Process.ConsoleSize.Width)
		} else {
			if term.IsTerminal(unix.Stdin) {
				// Use the size of our terminal.
				winsize, err = unix.IoctlGetWinsize(unix.Stdin, unix.TIOCGWINSZ)
				if err != nil {
					logrus.Debugf("error reading current terminal's size")
					winsize.Row = 0
					winsize.Col = 0
				}
			}
		}
		if winsize.Row != 0 && winsize.Col != 0 {
			if err = unix.IoctlSetWinsize(int(ptyFd), unix.TIOCSWINSZ, winsize); err != nil {
				logrus.Warnf("error setting terminal size for pty")
			}
			// FIXME - if we're connected to a terminal, we should
			// be passing the updated terminal size down when we
			// receive a SIGWINCH.
		}
		// Open an *os.File object that we can pass to our child.
		ctty = os.NewFile(uintptr(ptyFd), "/dev/tty")
		// Set ownership for the PTY.
		if err = ctty.Chown(rootUID, rootGID); err != nil {
			var cttyInfo unix.Stat_t
			err2 := unix.Fstat(int(ptyFd), &cttyInfo)
			from := ""
			op := "setting"
			if err2 == nil {
				op = "changing"
				from = fmt.Sprintf("from %d/%d ", cttyInfo.Uid, cttyInfo.Gid)
			}
			logrus.Warnf("error %s ownership of container PTY %sto %d/%d: %v", op, from, rootUID, rootGID, err)
		}
		// Set permissions on the PTY.
		if err = ctty.Chmod(0620); err != nil {
			logrus.Errorf("error setting permissions of container PTY: %v", err)
			os.Exit(1)
		}
		// Make a note that our child (the parent subprocess) should
		// have the PTY connected to its stdio, and that we should
		// close it once it's running.
		stdin = ctty
		stdout = ctty
		stderr = ctty
		closeOnceRunning = append(closeOnceRunning, ctty)
	} else {
		// Create pipes for stdio.
		stdinRead, stdinWrite, err := os.Pipe()
		if err != nil {
			logrus.Errorf("error opening pipe for stdin: %v", err)
		}
		stdoutRead, stdoutWrite, err := os.Pipe()
		if err != nil {
			logrus.Errorf("error opening pipe for stdout: %v", err)
		}
		stderrRead, stderrWrite, err := os.Pipe()
		if err != nil {
			logrus.Errorf("error opening pipe for stderr: %v", err)
		}
		// Make notes about what's going where.
		relays[unix.Stdin] = int(stdinWrite.Fd())
		relays[int(stdoutRead.Fd())] = unix.Stdout
		relays[int(stderrRead.Fd())] = unix.Stderr
		fdDesc[int(stdinWrite.Fd())] = "container stdin pipe"
		fdDesc[int(stdoutRead.Fd())] = "container stdout pipe"
		fdDesc[int(stderrRead.Fd())] = "container stderr pipe"
		fdDesc[unix.Stdin] = "stdin"
		fdDesc[unix.Stdout] = "stdout"
		fdDesc[unix.Stderr] = "stderr"
		// Set ownership for the pipes.
		if err = stdinRead.Chown(rootUID, rootGID); err != nil {
			logrus.Errorf("error setting ownership of container stdin pipe: %v", err)
			os.Exit(1)
		}
		if err = stdoutWrite.Chown(rootUID, rootGID); err != nil {
			logrus.Errorf("error setting ownership of container stdout pipe: %v", err)
			os.Exit(1)
		}
		if err = stderrWrite.Chown(rootUID, rootGID); err != nil {
			logrus.Errorf("error setting ownership of container stderr pipe: %v", err)
			os.Exit(1)
		}
		// Make a note that our child (the parent subprocess) should
		// have the pipes connected to its stdio, and that we should
		// close its ends of them once it's running.
		stdin = stdinRead
		stdout = stdoutWrite
		stderr = stderrWrite
		closeOnceRunning = append(closeOnceRunning, stdinRead, stdoutWrite, stderrWrite)
		stdinCopy = stdinWrite
		defer stdoutRead.Close()
		defer stderrRead.Close()
	}
	for readFd, writeFd := range relays {
		if err := unix.SetNonblock(readFd, true); err != nil {
			logrus.Errorf("error setting descriptor %d (%s) non-blocking: %v", readFd, fdDesc[readFd], err)
			return
		}
		if err := unix.SetNonblock(writeFd, false); err != nil {
			logrus.Errorf("error setting descriptor %d (%s) blocking: %v", relays[writeFd], fdDesc[writeFd], err)
			return
		}
	}
	if err := unix.SetNonblock(relays[unix.Stdin], true); err != nil {
		logrus.Errorf("error setting %d to nonblocking: %v", relays[unix.Stdin], err)
	}
	go func() {
		buffers := make(map[int]*bytes.Buffer)
		for _, writeFd := range relays {
			buffers[writeFd] = new(bytes.Buffer)
		}
		pollTimeout := -1
		stdinClose := false
		for len(relays) > 0 {
			fds := make([]unix.PollFd, 0, len(relays))
			for fd := range relays {
				fds = append(fds, unix.PollFd{Fd: int32(fd), Events: unix.POLLIN | unix.POLLHUP})
			}
			_, err := unix.Poll(fds, pollTimeout)
			if !util.LogIfNotRetryable(err, fmt.Sprintf("poll: %v", err)) {
				return
			}
			removeFds := make(map[int]struct{})
			for _, rfd := range fds {
				if rfd.Revents&unix.POLLHUP == unix.POLLHUP {
					removeFds[int(rfd.Fd)] = struct{}{}
				}
				if rfd.Revents&unix.POLLNVAL == unix.POLLNVAL {
					logrus.Debugf("error polling descriptor %s: closed?", fdDesc[int(rfd.Fd)])
					removeFds[int(rfd.Fd)] = struct{}{}
				}
				if rfd.Revents&unix.POLLIN == 0 {
					if stdinClose && stdinCopy == nil {
						continue
					}
					continue
				}
				b := make([]byte, 8192)
				nread, err := unix.Read(int(rfd.Fd), b)
				util.LogIfNotRetryable(err, fmt.Sprintf("read %s: %v", fdDesc[int(rfd.Fd)], err))
				if nread > 0 {
					if wfd, ok := relays[int(rfd.Fd)]; ok {
						nwritten, err := buffers[wfd].Write(b[:nread])
						if err != nil {
							logrus.Debugf("buffer: %v", err)
							continue
						}
						if nwritten != nread {
							logrus.Debugf("buffer: expected to buffer %d bytes, wrote %d", nread, nwritten)
							continue
						}
					}
					// If this is the last of the data we'll be able to read
					// from this descriptor, read as much as there is to read.
					for rfd.Revents&unix.POLLHUP == unix.POLLHUP {
						nr, err := unix.Read(int(rfd.Fd), b)
						util.LogIfUnexpectedWhileDraining(err, fmt.Sprintf("read %s: %v", fdDesc[int(rfd.Fd)], err))
						if nr <= 0 {
							break
						}
						if wfd, ok := relays[int(rfd.Fd)]; ok {
							nwritten, err := buffers[wfd].Write(b[:nr])
							if err != nil {
								logrus.Debugf("buffer: %v", err)
								break
							}
							if nwritten != nr {
								logrus.Debugf("buffer: expected to buffer %d bytes, wrote %d", nr, nwritten)
								break
							}
						}
					}
				}
				if nread == 0 {
					removeFds[int(rfd.Fd)] = struct{}{}
				}
			}
			pollTimeout = -1
			for wfd, buffer := range buffers {
				if buffer.Len() > 0 {
					nwritten, err := unix.Write(wfd, buffer.Bytes())
					util.LogIfNotRetryable(err, fmt.Sprintf("write %s: %v", fdDesc[wfd], err))
					if nwritten >= 0 {
						_ = buffer.Next(nwritten)
					}
				}
				if buffer.Len() > 0 {
					pollTimeout = 100
				}
				if wfd == relays[unix.Stdin] && stdinClose && buffer.Len() == 0 {
					stdinCopy.Close()
					delete(relays, unix.Stdin)
				}
			}
			for rfd := range removeFds {
				if rfd == unix.Stdin {
					buffer, found := buffers[relays[unix.Stdin]]
					if found && buffer.Len() > 0 {
						stdinClose = true
						continue
					}
				}
				if !options.Spec.Process.Terminal && rfd == unix.Stdin {
					stdinCopy.Close()
				}
				delete(relays, rfd)
			}
		}
	}()

	// Set up mounts and namespaces, and run the parent subprocess.
	status, err := runUsingChroot(options.Spec, options.BundlePath, ctty, stdin, stdout, stderr, closeOnceRunning)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running subprocess: %v\n", err)
		os.Exit(1)
	}

	// Pass the process's exit status back to the caller by exiting with the same status.
	if status.Exited() {
		if status.ExitStatus() != 0 {
			fmt.Fprintf(os.Stderr, "subprocess exited with status %d\n", status.ExitStatus())
		}
		os.Exit(status.ExitStatus())
	} else if status.Signaled() {
		fmt.Fprintf(os.Stderr, "subprocess exited on %s\n", status.Signal())
		os.Exit(1)
	}
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

	// Apologize for the namespace configuration that we're about to ignore.
	logNamespaceDiagnostics(spec)

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
	if err := setPlatformUnshareOptions(spec, cmd); err != nil {
		return 1, fmt.Errorf("error setting platform unshare options: %w", err)

	}
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
