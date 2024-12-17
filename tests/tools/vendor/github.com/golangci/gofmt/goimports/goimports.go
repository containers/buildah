// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goimports

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Extracted from golang.org/x/tools@v0.24.0/cmd/goimports/goimports.go

func writeTempFile(dir, prefix string, data []byte) (string, error) {
	file, err := os.CreateTemp(dir, prefix)
	if err != nil {
		return "", err
	}
	_, err = file.Write(data)
	if err1 := file.Close(); err == nil {
		err = err1
	}
	if err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func diff(b1, b2 []byte, filename string) (data []byte, err error) {
	f1, err := writeTempFile("", "gofmt", b1)
	if err != nil {
		return
	}
	defer os.Remove(f1)

	f2, err := writeTempFile("", "gofmt", b2)
	if err != nil {
		return
	}
	defer os.Remove(f2)

	cmd := "diff"
	if runtime.GOOS == "plan9" {
		cmd = "/bin/ape/diff"
	}

	data, err = exec.Command(cmd, "-u", f1, f2).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		return replaceTempFilename(data, filename)
	}
	return
}

// replaceTempFilename replaces temporary filenames in diff with actual one.
//
// --- /tmp/gofmt316145376	2017-02-03 19:13:00.280468375 -0500
// +++ /tmp/gofmt617882815	2017-02-03 19:13:00.280468375 -0500
// ...
// ->
// --- path/to/file.go.orig	2017-02-03 19:13:00.280468375 -0500
// +++ path/to/file.go	2017-02-03 19:13:00.280468375 -0500
// ...
func replaceTempFilename(diff []byte, filename string) ([]byte, error) {
	bs := bytes.SplitN(diff, []byte{'\n'}, 3)
	if len(bs) < 3 {
		return nil, fmt.Errorf("got unexpected diff for %s", filename)
	}
	// Preserve timestamps.
	var t0, t1 []byte
	if i := bytes.LastIndexByte(bs[0], '\t'); i != -1 {
		t0 = bs[0][i:]
	}
	if i := bytes.LastIndexByte(bs[1], '\t'); i != -1 {
		t1 = bs[1][i:]
	}
	// Always print filepath with slash separator.
	f := filepath.ToSlash(filename)
	bs[0] = []byte(fmt.Sprintf("--- %s%s", f+".orig", t0))
	bs[1] = []byte(fmt.Sprintf("+++ %s%s", f, t1))
	return bytes.Join(bs, []byte{'\n'}), nil
}
