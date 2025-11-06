// Supplying our own _start that just writes the message and exits avoids
// pulling in the proper standard library, which produces a smaller binary, but
// we still end up pulling in the language runtime.
package main
