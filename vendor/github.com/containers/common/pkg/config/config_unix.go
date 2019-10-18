// +build !windows

package config

// Defaults for linux/unix if none are specified
const (
	cniBinDir    = "/usr/libexec/cni:/opt/cni/bin"
	cniConfigDir = "/etc/cni/net.d/"
)
