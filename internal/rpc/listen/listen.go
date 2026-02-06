package listen

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/containers/buildah/internal/tmpdir"
)

func Listen(location string) (net.Listener, func() error, error) {
	cleanup := func() error { return nil }
	if location == "" {
		newParentDir, err := os.MkdirTemp(tmpdir.GetTempDir(), "buildah-socket")
		if err != nil {
			return nil, nil, fmt.Errorf("creating a temporary directory to hold a listening socket: %w", err)
		}
		location = filepath.Join(newParentDir, "build.sock")
		cleanup = func() error { return os.RemoveAll(newParentDir) }
	}
	l, err := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: location})
	if err != nil {
		cerr := cleanup()
		return nil, nil, errors.Join(err, cerr)
	}
	closeAndCleanup := func() error {
		lerr := l.Close()
		if lerr != nil && errors.Is(lerr, net.ErrClosed) {
			// if the listening socket was used by an rpc server that was explicitly
			// stopped, the rpc server will have closed the socket
			lerr = nil
		}
		cerr := cleanup()
		return errors.Join(lerr, cerr)
	}
	return l, closeAndCleanup, nil
}
