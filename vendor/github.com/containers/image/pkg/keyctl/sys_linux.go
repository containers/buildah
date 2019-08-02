// Copyright 2015 Jesse Sipprell. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux
// +build 386 amd64

package keyctl

import (
	"syscall"
	"unsafe"
)

type keyctlCommand int

type keyID int32

const (
	keySpecSessionKeyring keyID = -3
	keySpecUserKeyring    keyID = -4
)

const (
	keyctlGetKeyringID keyctlCommand = 0
	keyctlSetPerm      keyctlCommand = 5
	keyctlLink         keyctlCommand = 8
	keyctlUnlink       keyctlCommand = 9
	keyctlSearch       keyctlCommand = 10
	keyctlRead         keyctlCommand = 11
)

func (id keyID) ID() int32 {
	return int32(id)
}

func keyctl(cmd keyctlCommand, args ...uintptr) (r1 int32, r2 int32, err error) {
	a := make([]uintptr, 6)
	l := len(args)
	if l > 5 {
		l = 5
	}
	a[0] = uintptr(cmd)
	for idx, v := range args[:l] {
		a[idx+1] = v
	}
	v1, v2, errno := syscall.Syscall6(syscallKeyctl, a[0], a[1], a[2], a[3], a[4], a[5])
	if errno != 0 {
		err = errno
		return
	}

	r1 = int32(v1)
	r2 = int32(v2)
	return
}

func addkey(keyType, keyDesc string, payload []byte, id int32) (int32, error) {
	var (
		err    error
		errno  syscall.Errno
		b1, b2 *byte
		r1     uintptr
		pptr   unsafe.Pointer
	)

	if b1, err = syscall.BytePtrFromString(keyType); err != nil {
		return 0, err
	}

	if b2, err = syscall.BytePtrFromString(keyDesc); err != nil {
		return 0, err
	}

	if len(payload) > 0 {
		pptr = unsafe.Pointer(&payload[0])
	}
	r1, _, errno = syscall.Syscall6(syscallAddKey,
		uintptr(unsafe.Pointer(b1)),
		uintptr(unsafe.Pointer(b2)),
		uintptr(pptr),
		uintptr(len(payload)),
		uintptr(id),
		0)

	if errno != 0 {
		err = errno
		return 0, err
	}
	return int32(r1), nil
}

func newKeyring(id keyID) (*keyring, error) {
	r1, _, err := keyctl(keyctlGetKeyringID, uintptr(id), uintptr(1))
	if err != nil {
		return nil, err
	}

	if id < 0 {
		r1 = int32(id)
	}
	return &keyring{id: keyID(r1)}, nil
}

func searchKeyring(id keyID, name, keyType string) (keyID, error) {
	var (
		r1     int32
		b1, b2 *byte
		err    error
	)

	if b1, err = syscall.BytePtrFromString(keyType); err != nil {
		return 0, err
	}
	if b2, err = syscall.BytePtrFromString(name); err != nil {
		return 0, err
	}

	r1, _, err = keyctl(keyctlSearch, uintptr(id), uintptr(unsafe.Pointer(b1)), uintptr(unsafe.Pointer(b2)))
	return keyID(r1), err
}
