// +build go1.10

package archive

import (
	"archive/tar"
)

func copyPassHeader(hdr *tar.Header) {
	hdr.Format = tar.FormatPAX
}
