// +build linux

package chroot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/containers/buildah/tests/testreport/types"
	"github.com/containers/buildah/util"
	"github.com/containers/storage/pkg/reexec"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
)

const (
	reportCommand = "testreport"
)

func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func testMinimal(t *testing.T, modify func(g *generate.Generator, rootDir, bundleDir string), verify func(t *testing.T, report *types.TestReport)) {
	g, err := generate.New("linux")
	if err != nil {
		t.Fatalf("generate.New(%q): %v", "linux", err)
	}

	tempDir, err := ioutil.TempDir("", "chroot-test")
	if err != nil {
		t.Fatalf("ioutil.TempDir(%q, %q): %v", "", "chrootTest", err)
	}
	defer os.RemoveAll(tempDir)
	info, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("error checking permissions on %q: %v", tempDir, err)
	}
	if err = os.Chmod(tempDir, info.Mode()|0111); err != nil {
		t.Fatalf("error loosening permissions on %q: %v", tempDir, err)
	}

	rootDir := filepath.Join(tempDir, "root")
	if err := os.Mkdir(rootDir, 0711); err != nil {
		t.Fatalf("os.Mkdir(%q): %v", rootDir, err)
	}

	specPath := filepath.Join("..", "tests", reportCommand, reportCommand)
	specBinarySource, err := os.Open(specPath)
	if err != nil {
		t.Fatalf("open(%q): %v", specPath, err)
	}
	defer specBinarySource.Close()
	specBinary, err := os.OpenFile(filepath.Join(rootDir, reportCommand), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0711)
	if err != nil {
		t.Fatalf("open(%q): %v", filepath.Join(rootDir, reportCommand), err)
	}
	io.Copy(specBinary, specBinarySource)
	specBinary.Close()

	g.SetRootPath(rootDir)
	g.SetProcessArgs([]string{"/" + reportCommand})

	bundleDir := filepath.Join(tempDir, "bundle")
	if err := os.Mkdir(bundleDir, 0700); err != nil {
		t.Fatalf("os.Mkdir(%q): %v", bundleDir, err)
	}

	if modify != nil {
		modify(&g, rootDir, bundleDir)
	}

	uid, gid, err := util.GetHostRootIDs(g.Config)
	if err != nil {
		t.Fatalf("GetHostRootIDs: %v", err)
	}
	if err := os.Chown(rootDir, int(uid), int(gid)); err != nil {
		t.Fatalf("os.Chown(%q): %v", rootDir, err)
	}

	output := new(bytes.Buffer)
	if err := RunUsingChroot(g.Config, bundleDir, "/", new(bytes.Buffer), output, output); err != nil {
		t.Fatalf("run: %v: %s", err, output.String())
	}

	var report types.TestReport
	if json.Unmarshal(output.Bytes(), &report) != nil {
		t.Fatalf("decode: %v", err)
	}

	if verify != nil {
		verify(t, &report)
	}
}

func testNoop(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t, nil, nil)
}

func testMinimalSkeleton(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
		},
		func(t *testing.T, report *types.TestReport) {
		})
}

func TestProcessTerminal(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	for _, terminal := range []bool{false, true} {
		testMinimal(t,
			func(g *generate.Generator, rootDir, bundleDir string) {
				g.SetProcessTerminal(terminal)
			},
			func(t *testing.T, report *types.TestReport) {
				if report.Spec.Process.Terminal != terminal {
					t.Fatalf("expected terminal = %v, got %v", terminal, report.Spec.Process.Terminal)
				}
			})
	}
}

func TestProcessConsoleSize(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	for _, size := range [][2]uint{{80, 25}, {132, 50}} {
		testMinimal(t,
			func(g *generate.Generator, rootDir, bundleDir string) {
				g.SetProcessTerminal(true)
				g.SetProcessConsoleSize(size[0], size[1])
			},
			func(t *testing.T, report *types.TestReport) {
				if report.Spec.Process.ConsoleSize.Width != size[0] {
					t.Fatalf("expected console width = %v, got %v", size[0], report.Spec.Process.ConsoleSize.Width)
				}
				if report.Spec.Process.ConsoleSize.Height != size[1] {
					t.Fatalf("expected console height = %v, got %v", size[1], report.Spec.Process.ConsoleSize.Height)
				}
			})
	}
}

func TestProcessUser(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	for _, id := range []uint32{0, 1000} {
		testMinimal(t,
			func(g *generate.Generator, rootDir, bundleDir string) {
				g.SetProcessUID(id)
				g.SetProcessGID(id + 1)
				g.AddProcessAdditionalGid(id + 2)
			},
			func(t *testing.T, report *types.TestReport) {
				if report.Spec.Process.User.UID != id {
					t.Fatalf("expected UID %v, got %v", id, report.Spec.Process.User.UID)
				}
				if report.Spec.Process.User.GID != id+1 {
					t.Fatalf("expected GID %v, got %v", id+1, report.Spec.Process.User.GID)
				}
			})
	}
}

func TestProcessEnv(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	e := fmt.Sprintf("PARENT_TEST_PID=%d", syscall.Getpid())
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.ClearProcessEnv()
			g.AddProcessEnv("PARENT_TEST_PID", fmt.Sprintf("%d", syscall.Getpid()))
		},
		func(t *testing.T, report *types.TestReport) {
			for _, ev := range report.Spec.Process.Env {
				if ev == e {
					return
				}
			}
			t.Fatalf("expected environment variable %q", e)
		})
}

func TestProcessCwd(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			if err := os.Mkdir(filepath.Join(rootDir, "/no-such-directory"), 0700); err != nil {
				t.Fatalf("mkdir(%q): %v", filepath.Join(rootDir, "/no-such-directory"), err)
			}
			g.SetProcessCwd("/no-such-directory")
		},
		func(t *testing.T, report *types.TestReport) {
			if report.Spec.Process.Cwd != "/no-such-directory" {
				t.Fatalf("expected %q, got %q", "/no-such-directory", report.Spec.Process.Cwd)
			}
		})
}

func TestProcessCapabilities(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.ClearProcessCapabilities()
		},
		func(t *testing.T, report *types.TestReport) {
			if len(report.Spec.Process.Capabilities.Permitted) != 0 {
				t.Fatalf("expected no permitted capabilities, got %#v", report.Spec.Process.Capabilities.Permitted)
			}
		})
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.ClearProcessCapabilities()
			g.AddProcessCapabilityEffective("CAP_IPC_LOCK")
			g.AddProcessCapabilityPermitted("CAP_IPC_LOCK")
			g.AddProcessCapabilityInheritable("CAP_IPC_LOCK")
			g.AddProcessCapabilityBounding("CAP_IPC_LOCK")
			g.AddProcessCapabilityAmbient("CAP_IPC_LOCK")
		},
		func(t *testing.T, report *types.TestReport) {
			if len(report.Spec.Process.Capabilities.Permitted) != 1 {
				t.Fatalf("expected one permitted capability, got %#v", report.Spec.Process.Capabilities.Permitted)
			}
			if report.Spec.Process.Capabilities.Permitted[0] != "CAP_IPC_LOCK" {
				t.Fatalf("expected one capability CAP_IPC_LOCK, got %#v", report.Spec.Process.Capabilities.Permitted)
			}
		})
}

func TestProcessRlimits(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	for _, limit := range []int64{100 * 1024 * 1024 * 1024, 200 * 1024 * 1024 * 1024, syscall.RLIM_INFINITY} {
		testMinimal(t,
			func(g *generate.Generator, rootDir, bundleDir string) {
				g.ClearProcessRlimits()
				if limit != syscall.RLIM_INFINITY {
					g.AddProcessRlimits("rlimit_as", uint64(limit), uint64(limit))
				}
			},
			func(t *testing.T, report *types.TestReport) {
				var rlim *specs.POSIXRlimit
				for i := range report.Spec.Process.Rlimits {
					if strings.ToUpper(report.Spec.Process.Rlimits[i].Type) == "RLIMIT_AS" {
						rlim = &report.Spec.Process.Rlimits[i]
					}
				}
				if limit == syscall.RLIM_INFINITY && !(rlim == nil || (int64(rlim.Soft) == syscall.RLIM_INFINITY && int64(rlim.Hard) == syscall.RLIM_INFINITY)) {
					t.Fatalf("wasn't supposed to set limit on number of open files: %#v", rlim)
				}
				if limit != syscall.RLIM_INFINITY && rlim == nil {
					t.Fatalf("was supposed to set limit on number of open files")
				}
				if rlim != nil {
					if int64(rlim.Soft) != limit {
						t.Fatalf("soft limit was set to %d, not %d", rlim.Soft, limit)
					}
					if int64(rlim.Hard) != limit {
						t.Fatalf("hard limit was set to %d, not %d", rlim.Hard, limit)
					}
				}
			})
	}
}

func TestProcessNoNewPrivileges(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	for _, nope := range []bool{false, true} {
		testMinimal(t,
			func(g *generate.Generator, rootDir, bundleDir string) {
				g.SetProcessNoNewPrivileges(nope)
			},
			func(t *testing.T, report *types.TestReport) {
				if report.Spec.Process.NoNewPrivileges != nope {
					t.Fatalf("expected no-new-prives to be %v, got %v", nope, report.Spec.Process.NoNewPrivileges)
				}
			})
	}
}

func TestProcessOOMScoreAdj(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	for _, adj := range []int{0, 1, 2, 3} {
		testMinimal(t,
			func(g *generate.Generator, rootDir, bundleDir string) {
				g.SetProcessOOMScoreAdj(adj)
			},
			func(t *testing.T, report *types.TestReport) {
				adjusted := 0
				if report.Spec.Process.OOMScoreAdj != nil {
					adjusted = *report.Spec.Process.OOMScoreAdj
				}
				if adjusted != adj {
					t.Fatalf("expected oom-score-adj to be %v, got %v", adj, adjusted)
				}
			})
	}
}

func TestHostname(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	hostname := fmt.Sprintf("host%d", syscall.Getpid())
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.SetHostname(hostname)
		},
		func(t *testing.T, report *types.TestReport) {
			if report.Spec.Hostname != hostname {
				t.Fatalf("expected %q, got %q", hostname, report.Spec.Hostname)
			}
		})
}

func TestMounts(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.AddMount(specs.Mount{
				Source:      "tmpfs",
				Destination: "/was-not-there-before",
				Type:        "tmpfs",
				Options:     []string{"ro,size=0"},
			})
		},
		func(t *testing.T, report *types.TestReport) {
			found := false
			for _, mount := range report.Spec.Mounts {
				if mount.Destination == "/was-not-there-before" && mount.Type == "tmpfs" {
					found = true
				}
			}
			if !found {
				t.Fatal("added mount not found")
			}
		})
}

func TestLinuxIDMapping(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.ClearLinuxUIDMappings()
			g.ClearLinuxGIDMappings()
			g.AddLinuxUIDMapping(uint32(syscall.Getuid()), 0, 1)
			g.AddLinuxGIDMapping(uint32(syscall.Getgid()), 0, 1)
		},
		func(t *testing.T, report *types.TestReport) {
			if len(report.Spec.Linux.UIDMappings) != 1 {
				t.Fatalf("expected 1 uid mapping, got %q", len(report.Spec.Linux.UIDMappings))
			}
			if report.Spec.Linux.UIDMappings[0].HostID != uint32(syscall.Getuid()) {
				t.Fatalf("expected host uid mapping to be %d, got %d", syscall.Getuid(), report.Spec.Linux.UIDMappings[0].HostID)
			}
			if report.Spec.Linux.UIDMappings[0].ContainerID != 0 {
				t.Fatalf("expected container uid mapping to be 0, got %d", report.Spec.Linux.UIDMappings[0].ContainerID)
			}
			if report.Spec.Linux.UIDMappings[0].Size != 1 {
				t.Fatalf("expected container uid map size to be 1, got %d", report.Spec.Linux.UIDMappings[0].Size)
			}
			if report.Spec.Linux.GIDMappings[0].HostID != uint32(syscall.Getgid()) {
				t.Fatalf("expected host uid mapping to be %d, got %d", syscall.Getgid(), report.Spec.Linux.GIDMappings[0].HostID)
			}
			if report.Spec.Linux.GIDMappings[0].ContainerID != 0 {
				t.Fatalf("expected container gid mapping to be 0, got %d", report.Spec.Linux.GIDMappings[0].ContainerID)
			}
			if report.Spec.Linux.GIDMappings[0].Size != 1 {
				t.Fatalf("expected container gid map size to be 1, got %d", report.Spec.Linux.GIDMappings[0].Size)
			}
		})
}

func TestLinuxIDMappingShift(t *testing.T) {
	if syscall.Getuid() != 0 {
		t.Skip("tests need to be run as root")
	}
	testMinimal(t,
		func(g *generate.Generator, rootDir, bundleDir string) {
			g.ClearLinuxUIDMappings()
			g.ClearLinuxGIDMappings()
			g.AddLinuxUIDMapping(uint32(syscall.Getuid())+1, 0, 1)
			g.AddLinuxGIDMapping(uint32(syscall.Getgid())+1, 0, 1)
		},
		func(t *testing.T, report *types.TestReport) {
			if len(report.Spec.Linux.UIDMappings) != 1 {
				t.Fatalf("expected 1 uid mapping, got %q", len(report.Spec.Linux.UIDMappings))
			}
			if report.Spec.Linux.UIDMappings[0].HostID != uint32(syscall.Getuid()+1) {
				t.Fatalf("expected host uid mapping to be %d, got %d", syscall.Getuid()+1, report.Spec.Linux.UIDMappings[0].HostID)
			}
			if report.Spec.Linux.UIDMappings[0].ContainerID != 0 {
				t.Fatalf("expected container uid mapping to be 0, got %d", report.Spec.Linux.UIDMappings[0].ContainerID)
			}
			if report.Spec.Linux.UIDMappings[0].Size != 1 {
				t.Fatalf("expected container uid map size to be 1, got %d", report.Spec.Linux.UIDMappings[0].Size)
			}
			if report.Spec.Linux.GIDMappings[0].HostID != uint32(syscall.Getgid()+1) {
				t.Fatalf("expected host uid mapping to be %d, got %d", syscall.Getgid()+1, report.Spec.Linux.GIDMappings[0].HostID)
			}
			if report.Spec.Linux.GIDMappings[0].ContainerID != 0 {
				t.Fatalf("expected container gid mapping to be 0, got %d", report.Spec.Linux.GIDMappings[0].ContainerID)
			}
			if report.Spec.Linux.GIDMappings[0].Size != 1 {
				t.Fatalf("expected container gid map size to be 1, got %d", report.Spec.Linux.GIDMappings[0].Size)
			}
		})
}
