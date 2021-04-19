// +build !windows

package copier

import (
	"testing"
)

func TestPutChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testPut(t)
	canChroot = couldChroot
}

func TestStatChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testStat(t)
	canChroot = couldChroot
}

func TestGetSingleChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testGetSingle(t)
	canChroot = couldChroot
}

func TestGetMultipleChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testGetMultiple(t)
	canChroot = couldChroot
}

func TestEvalChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testEval(t)
	canChroot = couldChroot
}

func TestMkdirChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testMkdir(t)
	canChroot = couldChroot
}

func TestRemoveChroot(t *testing.T) {
	if uid != 0 {
		t.Skip("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testRemove(t)
	canChroot = couldChroot
}
