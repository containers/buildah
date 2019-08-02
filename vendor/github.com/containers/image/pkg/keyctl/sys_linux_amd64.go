// Copyright 2015 Jesse Sipprell. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux

package keyctl

const (
	syscallKeyctl uintptr = 250
	syscallAddKey uintptr = 248
)
