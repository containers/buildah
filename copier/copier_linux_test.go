package copier

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/containers/storage/pkg/mount"
	"github.com/containers/storage/pkg/reexec"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/gocapability/capability"
	"golang.org/x/sys/unix"
)

func init() {
	reexec.Register("get", getWrappedMain)
}

type getWrappedOptions struct {
	Root, Directory string
	GetOptions      GetOptions
	Globs           []string
	DropCaps        []capability.Cap
}

func getWrapped(root string, directory string, getOptions GetOptions, globs []string, dropCaps []capability.Cap, bulkWriter io.Writer) error {
	options := getWrappedOptions{
		Root:       root,
		Directory:  directory,
		GetOptions: getOptions,
		Globs:      globs,
		DropCaps:   dropCaps,
	}
	encoded, err := json.Marshal(&options)
	if err != nil {
		return errors.Wrapf(err, "error marshalling options")
	}
	cmd := reexec.Command("get")
	cmd.Env = append(cmd.Env, "OPTIONS="+string(encoded))
	cmd.Stdout = bulkWriter
	stderrBuf := bytes.Buffer{}
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	if stderrBuf.Len() > 0 {
		if err != nil {
			return fmt.Errorf("%v: %s", err, stderrBuf.String())
		}
		return fmt.Errorf("%s", stderrBuf.String())
	}
	return err
}

func getWrappedMain() {
	var options getWrappedOptions
	if err := json.Unmarshal([]byte(os.Getenv("OPTIONS")), &options); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	if len(options.DropCaps) > 0 {
		caps, err := capability.NewPid(0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(1)
		}
		for _, capType := range []capability.CapType{
			capability.AMBIENT,
			capability.BOUNDING,
			capability.INHERITABLE,
			capability.PERMITTED,
			capability.EFFECTIVE,
		} {
			for _, cap := range options.DropCaps {
				if caps.Get(capType, cap) {
					caps.Unset(capType, cap)
				}
			}
			if err := caps.Apply(capType); err != nil {
				fmt.Fprintf(os.Stderr, "error dropping capability %+v: %v", options.DropCaps, err)
				os.Exit(1)
			}
		}
	}
	if err := Get(options.Root, options.Directory, options.GetOptions, options.Globs, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}

func TestGetPermissionErrorNoChroot(t *testing.T) {
	couldChroot := canChroot
	canChroot = false
	testGetPermissionError(t)
	canChroot = couldChroot
}

func TestGetPermissionErrorChroot(t *testing.T) {
	if uid != 0 {
		t.Skipf("chroot() requires root privileges, skipping")
	}
	couldChroot := canChroot
	canChroot = true
	testGetPermissionError(t)
	canChroot = couldChroot
}

func testGetPermissionError(t *testing.T) {
	dropCaps := []capability.Cap{capability.CAP_DAC_OVERRIDE, capability.CAP_DAC_READ_SEARCH}
	tmp, err := ioutil.TempDir("", "copier-test-")
	require.NoErrorf(t, err, "error creating temporary directory")
	defer os.RemoveAll(tmp)
	err = os.Mkdir(filepath.Join(tmp, "unreadable-directory"), 0000)
	require.NoError(t, err, "error creating an unreadable directory")
	err = os.Mkdir(filepath.Join(tmp, "readable-directory"), 0755)
	require.NoError(t, err, "error creating a readable directory")
	err = os.Mkdir(filepath.Join(tmp, "readable-directory", "unreadable-subdirectory"), 0000)
	require.NoError(t, err, "error creating an unreadable subdirectory")
	err = ioutil.WriteFile(filepath.Join(tmp, "unreadable-file"), []byte("hi, i'm a file that you can't read"), 0000)
	require.NoError(t, err, "error creating an unreadable file")
	err = ioutil.WriteFile(filepath.Join(tmp, "readable-file"), []byte("hi, i'm also a file, and you can read me"), 0644)
	require.NoError(t, err, "error creating a readable file")
	err = ioutil.WriteFile(filepath.Join(tmp, "readable-directory", "unreadable-file"), []byte("hi, i'm also a file that you can't read"), 0000)
	require.NoError(t, err, "error creating an unreadable file in a readable directory")
	for _, ignore := range []bool{false, true} {
		t.Run(fmt.Sprintf("ignore=%v", ignore), func(t *testing.T) {
			var buf bytes.Buffer
			err = getWrapped(tmp, tmp, GetOptions{IgnoreUnreadable: ignore}, []string{"."}, dropCaps, &buf)
			if ignore {
				assert.NoError(t, err, "expected no errors")
				tr := tar.NewReader(&buf)
				items := 0
				_, err := tr.Next()
				for err == nil {
					items++
					_, err = tr.Next()
				}
				assert.True(t, errors.Is(err, io.EOF), "expected EOF to finish read contents")
				assert.Equalf(t, 2, items, "expected two readable items, got %d", items)
			} else {
				assert.Error(t, err, "expected an error")
				assert.Truef(t, errorIsPermission(err), "expected the error (%v) to be a permission error", err)
			}
		})
	}
}

func TestGetNoCrossDevice(t *testing.T) {
	if uid != 0 {
		t.Skip("test requires root privileges, skipping")
	}

	tmpdir, err := ioutil.TempDir("", "copier-test-noxdev-")
	require.NoError(t, err, "error creating temporary directory")
	defer os.RemoveAll(tmpdir)

	err = unix.Unshare(unix.CLONE_NEWNS)
	require.NoError(t, err, "error creating new mount namespace")

	subdir := filepath.Join(tmpdir, "subdir")
	err = os.Mkdir(subdir, 0755)
	require.NoErrorf(t, err, "error creating %q", subdir)

	err = mount.Mount("tmpfs", subdir, "tmpfs", "rw")
	require.NoErrorf(t, err, "error mounting tmpfs at %q", subdir)
	defer func() {
		err := mount.Unmount(subdir)
		assert.NoErrorf(t, err, "error unmounting %q", subdir)
	}()

	skipped := filepath.Join(subdir, "skipped.txt")
	err = ioutil.WriteFile(skipped, []byte("this file should have been skipped\n"), 0644)
	require.NoErrorf(t, err, "error writing file at %q", skipped)

	var buf bytes.Buffer
	err = Get(tmpdir, tmpdir, GetOptions{NoCrossDevice: true}, []string{"/"}, &buf) // grab contents of tmpdir
	require.NoErrorf(t, err, "error reading contents at %q", tmpdir)

	tr := tar.NewReader(&buf)
	th, err := tr.Next() // should be the "subdir" directory
	require.NoError(t, err, "error reading first entry archived")
	assert.Equal(t, "subdir", th.Name, `first entry in archive was not named "subdir"`)

	th, err = tr.Next()
	assert.Error(t, err, "should not have gotten a second entry in archive")
	assert.True(t, errors.Is(err, io.EOF), "expected an EOF trying to read a second entry in archive")
	if err == nil {
		t.Logf("got unexpected entry for %q", th.Name)
	}
}
