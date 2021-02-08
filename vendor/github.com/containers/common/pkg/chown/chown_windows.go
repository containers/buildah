package chown

// DangerousHostPath validates if a host path is dangerous and should not be modified
func DangerousHostPath(path string) (bool, error) {
	return false, nil
}

// ChangeHostPathOwnership changes the uid and gid ownership of a directory or file within the host.
// This is used by the volume U flag to change source volumes ownership
func ChangeHostPathOwnership(path string, recursive bool, uid, gid int) error {
	return nil
}
