//go:build !unix

package main

// This is really only here to prevent complaints about the source directory
// for a helper that's used in a Unix-specific test not having something that
// will compile on non-Unix platforms.
func main() {
}
