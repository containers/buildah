// +build linux

package imagebuildah

import (
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
)

// main() for resolveSymlink()'s subprocess.
func resolveChrootedSymlinks() {
	status := 0
	flag.Parse()
	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "%s needs two arguments\n", symlinkChrootedCommand)
		os.Exit(1)
	}
	// Our first parameter is the directory to chroot into.
	if err := unix.Chdir(flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "chdir(): %v\n", err)
		os.Exit(1)
	}
	if err := unix.Chroot(flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "chroot(): %v\n", err)
		os.Exit(1)
	}

	// Our second parameter is the path name to evaluate for symbolic links
	symLink, err := getSymbolicLink(flag.Arg(1))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting symbolic links: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.WriteString(symLink); err != nil {
		fmt.Fprintf(os.Stderr, "error writing string to stdout: %v\n", err)
		os.Exit(1)
	}
	os.Exit(status)
}

// main() for grandparent subprocess.  Its main job is to shuttle stdio back
// and forth, managing a pseudo-terminal if we want one, for our child, the
// parent subprocess.
func resolveSymlinkTimeModified() {
	status := 0
	flag.Parse()
	if len(flag.Args()) < 1 {
		os.Exit(1)
	}
	// Our first parameter is the directory to chroot into.
	if err := unix.Chdir(flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "chdir(): %v\n", err)
		os.Exit(1)
	}
	if err := unix.Chroot(flag.Arg(0)); err != nil {
		fmt.Fprintf(os.Stderr, "chroot(): %v\n", err)
		os.Exit(1)
	}

	// Our second parameter is the path name to evaluate for symbolic links.
	// Our third parameter is the time the cached intermediate image was created.
	// We check whether the modified time of the filepath we provide is after the time the cached image was created.
	timeIsGreater, err := modTimeIsGreater(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error checking if modified time of resolved symbolic link is greater: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.WriteString(fmt.Sprintf("%v", timeIsGreater)); err != nil {
		fmt.Fprintf(os.Stderr, "error writing string to stdout: %v\n", err)
		os.Exit(1)
	}
	os.Exit(status)
}
