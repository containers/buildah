//go:build !linux

package cgroups

// UserOwnsCurrentSystemdCgroup checks whether the current EUID owns the
// current cgroup.
func UserOwnsCurrentSystemdCgroup() (bool, error) {
	return false, nil
}
