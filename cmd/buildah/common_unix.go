// +build linux darwin

package main

import (
	"syscall"

	"github.com/sirupsen/logrus"
)

func checkUmask() {
	oldUmask := syscall.Umask(0022)
	if (oldUmask & ^0022) != 0 {
		logrus.Debugf("umask value too restrictive.  Forcing it to 022")
	}
}
