package secrets

import (
	"github.com/containers/common/pkg/subscriptions"
	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

var (
	// DefaultMountsFile holds the default mount paths in the form
	// "host_path:container_path"
	DefaultMountsFile = subscriptions.DefaultMountsFile
	// OverrideMountsFile holds the default mount paths in the form
	// "host_path:container_path" overridden by the user
	OverrideMountsFile = subscriptions.OverrideMountsFile
	// UserOverrideMountsFile holds the default mount paths in the form
	// "host_path:container_path" overridden by the rootless user
	UserOverrideMountsFile = subscriptions.DefaultMountsFile
)

// SecretMounts copies, adds, and mounts the secrets to the container root filesystem
// Deprecated, Please use SecretMountWithUIDGID
func SecretMounts(mountLabel, containerWorkingDir, mountFile string, rootless, disableFips bool) []rspec.Mount {
	return subscriptions.MountsWithUIDGID(mountLabel, containerWorkingDir, mountFile, containerWorkingDir, 0, 0, rootless, disableFips)
}

// SecretMountsWithUIDGID copies, adds, and mounts the secrets to the container root filesystem
// mountLabel: MAC/SELinux label for container content
// containerWorkingDir: Private data for storing secrets on the host mounted in container.
// mountFile: Additional mount points required for the container.
// mountPoint: Container image mountpoint
// uid: to assign to content created for secrets
// gid: to assign to content created for secrets
// rootless: indicates whether container is running in rootless mode
// disableFips: indicates whether system should ignore fips mode
func SecretMountsWithUIDGID(mountLabel, containerWorkingDir, mountFile, mountPoint string, uid, gid int, rootless, disableFips bool) []rspec.Mount {
	return subscriptions.MountsWithUIDGID(mountLabel, containerWorkingDir, mountFile, containerWorkingDir, 0, 0, rootless, disableFips)
}
