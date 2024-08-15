package parse

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/containers/buildah/define"
	"github.com/containers/image/v5/types"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommonBuildOptionsFromFlagSet(t *testing.T) {
	fs := pflag.NewFlagSet("testme", pflag.PanicOnError)
	fs.String("memory", "1GB", "")
	fs.String("shm-size", "5TB", "")
	fs.String("cpuset-cpus", "1", "")
	fs.String("cpuset-mems", "2", "")
	fs.String("cgroup-parent", "none", "")
	err := fs.Parse([]string{"--memory", "2GB"})
	assert.NoError(t, err)
	cbo, err := CommonBuildOptionsFromFlagSet(fs, fs.Lookup)
	assert.NoError(t, err)
	assert.Equal(t, cbo.Memory, int64(2147483648))
}

// TestDeviceParser verifies the given device strings is parsed correctly
func TestDeviceParser(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Devices is only supported on Linux")
	}

	// Test defaults
	src, dest, permissions, err := Device("/dev/foo")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/foo")
	assert.Equal(t, permissions, "rwm")

	// Test defaults, different dest
	src, dest, permissions, err = Device("/dev/foo:/dev/bar")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/bar")
	assert.Equal(t, permissions, "rwm")

	// Test fully specified
	src, dest, permissions, err = Device("/dev/foo:/dev/bar:rm")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/bar")
	assert.Equal(t, permissions, "rm")

	// Test device, permissions
	src, dest, permissions, err = Device("/dev/foo:rm")
	assert.NoError(t, err)
	assert.Equal(t, src, "/dev/foo")
	assert.Equal(t, dest, "/dev/foo")
	assert.Equal(t, permissions, "rm")

	// test bogus permissions
	_, _, _, err = Device("/dev/fuse1:BOGUS")
	assert.Error(t, err)

	_, _, _, err = Device("")
	assert.Error(t, err)

	_, _, _, err = Device("/dev/foo:/dev/bar:rm:")
	assert.Error(t, err)

	_, _, _, err = Device("/dev/foo::rm")
	assert.Error(t, err)
}

func TestIsValidDeviceMode(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Devices is only supported on Linux")
	}
	assert.False(t, isValidDeviceMode("BOGUS"))
	assert.False(t, isValidDeviceMode("rwx"))
	assert.True(t, isValidDeviceMode("r"))
	assert.True(t, isValidDeviceMode("rw"))
	assert.True(t, isValidDeviceMode("rm"))
	assert.True(t, isValidDeviceMode("rwm"))
}

func TestDeviceFromPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Devices is only supported on Linux")
	}
	// Path is valid
	dev, err := DeviceFromPath("/dev/null")
	assert.NoError(t, err)
	assert.Equal(t, len(dev), 1)
	assert.Equal(t, dev[0].Major, int64(1))
	assert.Equal(t, dev[0].Minor, int64(3))
	assert.Equal(t, string(dev[0].Permissions), "rwm")
	assert.Equal(t, dev[0].Uid, uint32(0))
	assert.Equal(t, dev[0].Gid, uint32(0))

	// Path does not exists
	_, err = DeviceFromPath("/dev/BOGUS")
	assert.Error(t, err)

	// Path is a directory of devices
	_, err = DeviceFromPath("/dev/pts")
	assert.NoError(t, err)

	// path of directory has no device
	_, err = DeviceFromPath("/etc/passwd")
	assert.Error(t, err)
}

func TestIDMappingOptions(t *testing.T) {
	fs := pflag.NewFlagSet("testme", pflag.PanicOnError)
	pfs := pflag.NewFlagSet("persist", pflag.PanicOnError)
	fs.String("userns-uid-map-user", "", "")
	fs.String("userns-gid-map-group", "", "")
	fs.String("userns-uid-map", "", "")
	fs.String("userns-gid-map", "", "")
	fs.String("userns", "", "")
	err := fs.Parse([]string{})
	assert.NoError(t, err)
	uos, _, err := IDMappingOptionsFromFlagSet(fs, pfs, fs.Lookup)
	assert.NoError(t, err)
	nso := uos.Find(string(specs.UserNamespace))
	assert.Equal(t, *nso, define.NamespaceOption{
		Host: true,
		Name: string(specs.UserNamespace),
	})
}

func TestIsolation(t *testing.T) {
	def, err := defaultIsolation()
	if err != nil {
		assert.Error(t, err)
	}

	isolations := []string{"", "default", "oci", "chroot", "rootless"}
	for _, i := range isolations {
		isolation, err := IsolationOption(i)
		if err != nil {
			assert.Error(t, fmt.Errorf("isolation %q not supported", i))
		}
		var expected string
		switch i {
		case "":
			expected = def.String()
		case "default":
			expected = "oci"
		default:
			expected = i
		}

		if isolation.String() != expected {
			assert.Error(t, fmt.Errorf("isolation %q not equal to user input %q", isolation.String(), expected))
		}
	}
}

func TestNamespaceOptions(t *testing.T) {
	fs := pflag.NewFlagSet("testme", pflag.PanicOnError)
	fs.String("cgroupns", "", "")
	err := fs.Parse([]string{"--cgroupns", "private"})
	assert.NoError(t, err)
	nsos, np, err := NamespaceOptionsFromFlagSet(fs, fs.Lookup)
	assert.NoError(t, err)
	assert.Equal(t, np, define.NetworkEnabled)
	nso := nsos.Find(string(specs.CgroupNamespace))
	assert.Equal(t, *nso, define.NamespaceOption{
		Name: string(specs.CgroupNamespace),
	})
}

func TestParsePlatform(t *testing.T) {
	os, arch, variant, err := Platform("a/b/c")
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, os, "a")
	assert.Equal(t, arch, "b")
	assert.Equal(t, variant, "c")

	os, arch, variant, err = Platform("a/b")
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, os, "a")
	assert.Equal(t, arch, "b")
	assert.Equal(t, variant, "")

	_, _, _, err = Platform("a")
	assert.Error(t, err)
}

func TestParsePullPolicy(t *testing.T) {
	testCases := map[string]bool{
		"missing":    true,
		"ifmissing":  true,
		"notpresent": true,
		"always":     true,
		"true":       true,
		"ifnewer":    true,
		"newer":      true,
		"false":      true,
		"never":      true,
		"trye":       false,
		"truth":      false,
	}
	for value, result := range testCases {
		t.Run(value, func(t *testing.T) {
			policy, err := pullPolicyWithFlags(value, false, false)
			if result {
				require.NoErrorf(t, err, "expected value %q to be recognized", value)
			} else {
				require.Errorf(t, err, "did not expect value %q to be recognized as %q", value, policy.String())
			}
		})
	}
}

func TestSplitStringWithColonEscape(t *testing.T) {
	tests := []struct {
		volume         string
		expectedResult []string
	}{
		{"/root/a:/root/test:O", []string{"/root/a", "/root/test", "O"}},
		{"/root/a\\:b/c:/root/test:O", []string{"/root/a:b/c", "/root/test", "O"}},
		{"/root/a:/root/test\\:test1/a:O", []string{"/root/a", "/root/test:test1/a", "O"}},
		{"/root/a\\:b/c:/root/test\\:test1/a:O", []string{"/root/a:b/c", "/root/test:test1/a", "O"}},
	}
	for _, args := range tests {
		val := SplitStringWithColonEscape(args.volume)
		assert.Equal(t, val, args.expectedResult)
	}
}

func TestSystemContextFromFlagSet(t *testing.T) {
	fs := pflag.NewFlagSet("testme", pflag.PanicOnError)
	fs.Bool("tls-verify", false, "")
	err := fs.Parse([]string{"--tls-verify", "false"})
	assert.NoError(t, err)
	sc, err := SystemContextFromFlagSet(fs, fs.Lookup)
	assert.NoError(t, err)
	assert.Equal(t, sc, &types.SystemContext{
		BigFilesTemporaryDir:        GetTempDir(),
		DockerInsecureSkipTLSVerify: types.OptionalBoolFalse,
		DockerRegistryUserAgent:     fmt.Sprintf("Buildah/%s", define.Version),
	})
}
