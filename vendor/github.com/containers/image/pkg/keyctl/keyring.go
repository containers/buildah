// Copyright 2015 Jesse Sipprell. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux
// +build 386 amd64

// Package keyctl is a Go interface to linux kernel keyrings (keyctl interface)
package keyctl

// Keyring is the basic interface to a linux keyctl keyring.
type Keyring interface {
	ID
	Add(string, []byte) (*Key, error)
	Search(string) (*Key, error)
}

type keyring struct {
	id keyID
}

// ID is unique 32-bit serial number identifiers for all Keys and Keyrings have.
type ID interface {
	ID() int32
}

// Add a new key to a keyring. The key can be searched for later by name.
func (kr *keyring) Add(name string, key []byte) (*Key, error) {
	r, err := addkey("user", name, key, int32(kr.id))
	if err == nil {
		key := &Key{Name: name, id: keyID(r), ring: kr.id}
		return key, nil
	}
	return nil, err
}

// Search for a key by name, this also searches child keyrings linked to this
// one. The key, if found, is linked to the top keyring that Search() was called
// from.
func (kr *keyring) Search(name string) (*Key, error) {
	id, err := searchKeyring(kr.id, name, "user")
	if err == nil {
		return &Key{Name: name, id: id, ring: kr.id}, nil
	}
	return nil, err
}

// ID returns the 32-bit kernel identifier of a keyring
func (kr *keyring) ID() int32 {
	return int32(kr.id)
}

// SessionKeyring returns the current login session keyring
func SessionKeyring() (Keyring, error) {
	return newKeyring(keySpecSessionKeyring)
}

// UserKeyring  returns the keyring specific to the current user.
func UserKeyring() (Keyring, error) {
	return newKeyring(keySpecUserKeyring)
}

// Unlink an object from a keyring
func Unlink(parent Keyring, child ID) error {
	_, _, err := keyctl(keyctlUnlink, uintptr(child.ID()), uintptr(parent.ID()))
	return err
}

// Link a key into a keyring
func Link(parent Keyring, child ID) error {
	_, _, err := keyctl(keyctlLink, uintptr(child.ID()), uintptr(parent.ID()))
	return err
}
