package parse

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

	//test bogus permissions
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
	assert.Equal(t, dev.Major, int64(1))
	assert.Equal(t, dev.Minor, int64(3))
	assert.Equal(t, dev.Permissions, "rwm")
	assert.Equal(t, dev.Uid, uint32(0))
	assert.Equal(t, dev.Gid, uint32(0))

	// Path does not exists
	_, err = DeviceFromPath("/dev/BOGUS")
	assert.Error(t, err)

	// Path exists but is not a device
	_, err = DeviceFromPath("/dev/pts")
	assert.Error(t, err)
}
