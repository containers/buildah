// +build windows

package buildah

import (
	"github.com/containers/storage/pkg/idtools"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/opencontainers/runtime-tools/generate"
)

func setChildProcess() error {
	return ErrNotSupported
}

func runSetupBuiltinVolumes(mountLabel, mountPoint, containerDir string, copyWithTar func(srcPath, dstPath string) error, builtinVolumes []string, rootUID, rootGID int) ([]specs.Mount, error) {
	return nil, ErrNotSupported
}

func setupTerminal(g *generate.Generator, terminalPolicy TerminalPolicy, terminalSize *specs.Box) {}

func runUsingRuntimeMain() {}

func (b *Builder) generateHosts(rdir, hostname string, addHosts []string, chownOpts *idtools.IDPair) (string, error) {
	return "", ErrNotSupported
}
func (b *Builder) addNetworkConfig(rdir, hostPath string, chownOpts *idtools.IDPair, dnsServers, dnsSearch, dnsOptions []string) (string, error) {
	return "", ErrNotSupported
}

func (b *Builder) Run(command []string, options RunOptions) error {
	return ErrNotSupported
}
