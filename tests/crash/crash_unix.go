package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Launch a child process and behave, rather unconvincingly, as an OCI runtime.
func main() {
	if err := exec.Command("sh", "-c", "sleep 0 &").Start(); err != nil {
		logrus.Fatalf("%v", err)
	}
	for _, arg := range os.Args {
		switch arg {
		case "create":
			logrus.Info("created\n")
			os.Exit(0)
		case "delete":
			logrus.Info("deleted\n")
			os.Exit(0)
		case "kill":
			logrus.Info("killed\n")
			os.Exit(0)
		case "start":
			logrus.Info("starting\n")
			// crash here, so that our caller, being run under
			// "wait", will have to reap us and our errant child
			// process, lest "wait" complain
			if err := syscall.Kill(os.Getpid(), syscall.SIGSEGV); err != nil {
				logrus.Fatalf("awkward: error sending SIGSEGV to myself: %v", err)
			}
		}
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGSEGV); err != nil {
		logrus.Fatalf("awkward: error sending SIGSEGV to myself: %v", err)
	}
}
