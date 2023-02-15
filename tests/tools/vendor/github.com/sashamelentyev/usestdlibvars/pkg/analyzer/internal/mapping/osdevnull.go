//go:build unix || (js && wasm)

package mapping

import "os"

func init() {
	OSDevNull[os.DevNull] = "os.DevNull"
}
