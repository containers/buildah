// +build !windows

package copier

import (
	"testing"
)

func TestPutChroot(t *testing.T) {
	if uid != 0 {
		t.Skipf("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testPut(t)
	canChroot = couldChroot
}

func TestStatChroot(t *testing.T) {
	if uid != 0 {
		t.Skipf("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testStat(t)
	canChroot = couldChroot
}

func TestGetSingleChroot(t *testing.T) {
	if uid != 0 {
		t.Skipf("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testGetSingle(t)
	canChroot = couldChroot
}

func TestGetMultipleChroot(t *testing.T) {
	if uid != 0 {
		t.Skipf("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testGetMultiple(t)
	canChroot = couldChroot
}

func TestMkdirChroot(t *testing.T) {
	if uid != 0 {
		t.Skipf("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testMkdir(t)
	canChroot = couldChroot
}
