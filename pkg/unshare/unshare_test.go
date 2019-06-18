// +build linux

package unshare

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/containers/storage/pkg/reexec"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func init() {
	reexec.Register("report", report)
}

var (
	CloneFlags = map[string]int{
		"ipc":  syscall.CLONE_NEWIPC,
		"net":  syscall.CLONE_NEWNET,
		"mnt":  syscall.CLONE_NEWNS,
		"user": syscall.CLONE_NEWUSER,
		"uts":  syscall.CLONE_NEWUTS,
	}
)

type Report struct {
	Namespaces  map[string]string
	UIDMappings []specs.LinuxIDMapping
	GIDMappings []specs.LinuxIDMapping
	Pgrp        int
	Sid         int
	OOMScoreAdj int
}

func report() {
	var report Report
	report.Namespaces = make(map[string]string)

	for name := range CloneFlags {
		linkTarget, err := os.Readlink("/proc/self/ns/" + name)
		if err != nil {
			logrus.Errorf("error reading link /proc/self/ns/%s: %v", name, err)
			os.Exit(1)
		}
		report.Namespaces[name] = linkTarget
	}

	report.Pgrp = syscall.Getpgrp()

	sid, err := unix.Getsid(unix.Getpid())
	if err != nil {
		logrus.Errorf("error reading current session ID: %v", err)
		os.Exit(1)
	}
	report.Sid = sid

	oomBytes, err := ioutil.ReadFile("/proc/self/oom_score_adj")
	if err != nil {
		logrus.Errorf("error reading current oom_score_adj: %v", err)
		os.Exit(1)
	}
	oomFields := strings.Fields(string(oomBytes))
	if len(oomFields) != 1 {
		logrus.Errorf("error parsing current oom_score_adj %q: wrong number of fields", string(oomBytes))
		os.Exit(1)
	}
	oom, err := strconv.Atoi(oomFields[0])
	if err != nil {
		logrus.Errorf("error parsing current oom_score_adj %q: %v", oomFields[0], err)
		os.Exit(1)
	}
	report.OOMScoreAdj = oom

	uidmap, gidmap, err := GetHostIDMappings("")
	if err != nil {
		logrus.Errorf("error reading current ID mappings: %v", err)
		os.Exit(1)
	}
	report.UIDMappings = append(report.UIDMappings, uidmap...)
	report.GIDMappings = append(report.GIDMappings, gidmap...)

	if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
		logrus.Error(err.Error())
		os.Exit(1)
	}
}

func TestUnshareNamespaces(t *testing.T) {
	for name, flag := range CloneFlags {
		var report Report
		buf := new(bytes.Buffer)
		cmd := Command("report")
		cmd.UnshareFlags = syscall.CLONE_NEWUSER | flag
		cmd.UidMappings = []specs.LinuxIDMapping{{HostID: uint32(syscall.Getuid()), ContainerID: 0, Size: 1}}
		cmd.GidMappings = []specs.LinuxIDMapping{{HostID: uint32(syscall.Getgid()), ContainerID: 0, Size: 1}}
		cmd.Stdout = buf
		cmd.Stderr = buf
		err := cmd.Run()
		if err != nil {
			t.Fatalf("run %q: %v: %s", name, err, buf.String())
			break
		}
		if err = json.Unmarshal(buf.Bytes(), &report); err != nil {
			t.Fatalf("error parsing results: %v", err)
			break
		}
		for ns := range CloneFlags {
			linkTarget, err := os.Readlink("/proc/self/ns/" + ns)
			if err != nil {
				t.Fatalf("error reading link /proc/self/ns/%s: %v", ns, err)
				os.Exit(1)
			}
			if ns == name || ns == "user" { // we always create a new user namespace
				if report.Namespaces[ns] == linkTarget {
					t.Fatalf("child is still in our %q namespace", name)
					os.Exit(1)
				}
			} else {
				if report.Namespaces[ns] != linkTarget {
					t.Fatalf("child is not in our %q namespace", name)
					os.Exit(1)
				}
			}
		}
	}
}

func TestUnsharePgrp(t *testing.T) {
	for _, same := range []bool{false, true} {
		var report Report
		buf := new(bytes.Buffer)
		cmd := Command("report")
		cmd.Setpgrp = !same
		cmd.Stdout = buf
		cmd.Stderr = buf
		err := cmd.Run()
		if err != nil {
			t.Fatalf("run: %v: %s", err, buf.String())
			break
		}
		if err = json.Unmarshal(buf.Bytes(), &report); err != nil {
			t.Fatalf("error parsing results: %v", err)
			break
		}
		if (report.Pgrp == syscall.Getpgrp()) != same {
			t.Fatalf("expected %d == %d to be %v", report.Pgrp, syscall.Getpgrp(), same)
		}
	}
}

func TestUnshareSid(t *testing.T) {
	sid, err := unix.Getsid(unix.Getpid())
	if err != nil {
		t.Fatalf("error reading current session ID: %v", err)
	}
	for _, same := range []bool{false, true} {
		var report Report
		buf := new(bytes.Buffer)
		cmd := Command("report")
		cmd.Setsid = !same
		cmd.Stdout = buf
		cmd.Stderr = buf
		err := cmd.Run()
		if err != nil {
			t.Fatalf("run: %v: %s", err, buf.String())
			break
		}
		if err = json.Unmarshal(buf.Bytes(), &report); err != nil {
			t.Fatalf("error parsing results: %v", err)
			break
		}
		if (report.Sid == sid) != same {
			t.Fatalf("expected %d == %d to be %v", report.Sid, sid, same)
		}
	}
}

func TestUnshareOOMScoreAdj(t *testing.T) {
	for _, adj := range []int{0, 1, 2, 3} {
		var report Report
		buf := new(bytes.Buffer)
		cmd := Command("report")
		cmd.OOMScoreAdj = &adj
		cmd.Stdout = buf
		cmd.Stderr = buf
		err := cmd.Run()
		if err != nil {
			t.Fatalf("run: %v: %s", err, buf.String())
			break
		}
		if err = json.Unmarshal(buf.Bytes(), &report); err != nil {
			t.Fatalf("error parsing results: %v", err)
			break
		}
		if report.OOMScoreAdj != adj {
			t.Fatalf("saw oom_score_adj %d to be %v", adj, report.OOMScoreAdj)
		}
	}
}

func TestUnshareIDMappings(t *testing.T) {
	var report Report
	buf := new(bytes.Buffer)
	cmd := Command("report")
	cmd.UnshareFlags = syscall.CLONE_NEWUSER
	cmd.UidMappings = []specs.LinuxIDMapping{{HostID: uint32(syscall.Getuid()), ContainerID: 0, Size: 1}}
	cmd.GidMappings = []specs.LinuxIDMapping{{HostID: uint32(syscall.Getgid()), ContainerID: 0, Size: 1}}
	cmd.Stdout = buf
	cmd.Stderr = buf
	err := cmd.Run()
	if err != nil {
		t.Fatalf("run: %v: %s", err, buf.String())
	}
	if err = json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("error parsing results: %v", err)
	}
	if len(cmd.UidMappings) != len(report.UIDMappings) {
		t.Fatalf("set %d UID mappings, read %d instead", len(cmd.UidMappings), len(report.UIDMappings))
	}
	for i := range cmd.UidMappings {
		if cmd.UidMappings[i].ContainerID != report.UIDMappings[i].ContainerID ||
			cmd.UidMappings[i].HostID != report.UIDMappings[i].HostID ||
			cmd.UidMappings[i].Size != report.UIDMappings[i].Size {
			t.Fatalf("uid map entry %#v != %#v", cmd.UidMappings[i], report.UIDMappings[i])
		}
	}
	if len(cmd.GidMappings) != len(report.GIDMappings) {
		t.Fatalf("set %d GID mappings, read %d instead", len(cmd.GidMappings), len(report.GIDMappings))
	}
	for i := range cmd.GidMappings {
		if cmd.GidMappings[i].ContainerID != report.GIDMappings[i].ContainerID ||
			cmd.GidMappings[i].HostID != report.GIDMappings[i].HostID ||
			cmd.GidMappings[i].Size != report.GIDMappings[i].Size {
			t.Fatalf("gid map entry %#v != %#v", cmd.GidMappings[i], report.GIDMappings[i])
		}
	}
}
