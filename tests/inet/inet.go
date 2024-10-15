package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
)

// This is similar to netcat's listen mode, except it logs which port it's
// assigned if it's told to attempt to bind to port 0.  Or it's similar to
// inetd, if it wasn't a daemon, and it wasn't as well-written.
func main() {
	pidFile := ""
	portFile := ""
	detach := false
	flag.BoolVar(&detach, "detach", false, "detach from terminal")
	flag.StringVar(&portFile, "port-file", "", "file to write listening port number")
	flag.StringVar(&pidFile, "pid-file", "", "file to write process ID to")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf("Usage: %s [-port-file filename] [-pid-file filename] command ...\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	// Start listening without specifying a port number.
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{})
	if err != nil {
		logrus.Fatalf("listening: %v", err)
	}
	// Retrieve the address we ended up bound to and write the port number
	// part to the specified file, if one was specified.
	addrString := ln.Addr().String()
	_, portString, err := net.SplitHostPort(addrString)
	if err != nil {
		logrus.Fatalf("finding the port number in %q: %v", addrString, err)
	}
	if portFile != "" {
		if err := os.WriteFile(portFile, []byte(portString), 0o644); err != nil {
			logrus.Fatalf("writing listening port to %q: %v", portFile, err)
		}
		defer os.Remove(portFile)
	}
	// Write our process ID to the specified file, if one was specified.
	if pidFile != "" {
		pid := strconv.Itoa(os.Getpid())
		if err := os.WriteFile(pidFile, []byte(pid), 0o644); err != nil {
			logrus.Fatalf("writing pid %d to %q: %v", os.Getpid(), pidFile, err)
		}
		defer os.Remove(pidFile)
	}
	// Now we can log which port we're listening on.
	fmt.Printf("process %d listening on port %s\n", os.Getpid(), portString)
	closeCloser := func(closer io.Closer) {
		if err := closer.Close(); err != nil {
			logrus.Errorf("closing: %v", err)
		}
	}
	// Helper function to shuttle data between a reader and a writer.
	relay := func(reader io.Reader, writer io.Writer) error {
		buffer := make([]byte, 1024)
		for {
			nr, err := reader.Read(buffer)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			if nr == 0 {
				return nil
			}
			if nr < 0 {
				// no error?
				break
			}
			nw, err := writer.Write(buffer[:nr])
			if err != nil {
				return nil
			}
			if nw != nr {
				return fmt.Errorf("short write: %d != %d", nw, nr)
			}
		}
		return nil
	}
	for {
		// Accept the next incoming connection.
		conn, err := ln.AcceptTCP()
		if err != nil {
			logrus.Errorf("accepting new connection: %v", err)
			continue
		}
		if conn == nil {
			logrus.Error("no new connection?")
			continue
		}
		go func() {
			defer closeCloser(conn)
			rawConn, err := conn.SyscallConn()
			if err != nil {
				logrus.Errorf("getting underlying connection: %v", err)
				return
			}
			var setNonblockError error
			if err := rawConn.Control(func(fd uintptr) {
				setNonblockError = syscall.SetNonblock(int(fd), true)
			}); err != nil {
				logrus.Errorf("marking connection nonblocking (outer): %v", err)
				return
			}
			if setNonblockError != nil {
				logrus.Errorf("marking connection nonblocking (inner): %v", setNonblockError)
				return
			}
			// Create pipes for the subprocess's stdio.
			stdinReader, stdinWriter, err := os.Pipe()
			if err != nil {
				logrus.Errorf("opening pipe for stdin: %v", err)
				return
			}
			defer closeCloser(stdinWriter)
			stdoutReader, stdoutWriter, err := os.Pipe()
			if err != nil {
				logrus.Errorf("opening pipe for stdout: %v", err)
				closeCloser(stdinReader)
				return
			}
			defer closeCloser(stdoutReader)
			if err := syscall.SetNonblock(int(stdoutReader.Fd()), true); err != nil {
				logrus.Errorf("marking stdout reader nonblocking: %v", err)
				closeCloser(stdinReader)
				closeCloser(stdoutWriter)
				return
			}
			// Start the subprocess.
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Stdin = stdinReader
			cmd.Stdout = stdoutWriter
			cmd.Stderr = os.Stderr
			if err := cmd.Start(); err != nil {
				logrus.Errorf("starting %v: %v", args, err)
				closeCloser(stdinReader)
				closeCloser(stdoutWriter)
				return
			}
			// Process the subprocess's stdio and wait for it to exit,
			// presumably when it runs out of data.
			var relayGroup multierror.Group
			relayGroup.Go(func() error {
				err := relay(conn, stdinWriter)
				closeCloser(stdinWriter)
				return err
			})
			relayGroup.Go(func() error {
				err := relay(stdoutReader, conn)
				closeCloser(stdoutReader)
				return err
			})
			relayGroup.Go(func() error {
				err := cmd.Wait()
				closeCloser(conn)
				return err
			})
			merr := relayGroup.Wait()
			if merr != nil && merr.ErrorOrNil() != nil {
				logrus.Errorf("%v\n", merr)
			}
		}()
	}
}
