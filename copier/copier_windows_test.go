//go:build windows

package copier

const (
	testModeMask           = int64(0o600)
	testIgnoreSymlinkDates = true
)
