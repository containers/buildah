package buildah

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"testing"
	"time"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func (c *CompositeDigester) isOpen() bool {
	for _, digester := range c.digesters {
		if tarDigester, ok := digester.(*tarDigester); ok {
			if tarDigester.isOpen {
				return true
			}
		}
	}
	return false
}

func TestCompositeDigester(t *testing.T) {
	tests := []struct {
		name       string
		itemTypes  []string
		resultType string
	}{
		{
			name:       "download",
			itemTypes:  []string{""},
			resultType: "",
		},
		{
			name:       "file",
			itemTypes:  []string{"file"},
			resultType: "file",
		},
		{
			name:       "dir",
			itemTypes:  []string{"dir"},
			resultType: "dir",
		},
		{
			name:       "multiple-1",
			itemTypes:  []string{"file", "dir"},
			resultType: "multi",
		},
		{
			name:       "multiple-2",
			itemTypes:  []string{"dir", "file"},
			resultType: "multi",
		},
		{
			name:       "multiple-3",
			itemTypes:  []string{"", "dir"},
			resultType: "multi",
		},
		{
			name:       "multiple-4",
			itemTypes:  []string{"", "file"},
			resultType: "multi",
		},
		{
			name:       "multiple-5",
			itemTypes:  []string{"dir", ""},
			resultType: "multi",
		},
		{
			name:       "multiple-6",
			itemTypes:  []string{"file", ""},
			resultType: "multi",
		},
	}
	var digester CompositeDigester
	var i int
	var buf bytes.Buffer
	zero := time.Unix(0, 0)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, filtered := range []bool{false, true} {
				desc := "unfiltered"
				if filtered {
					desc = "filter"
				}
				t.Run(desc, func(t *testing.T) {
					if i > 0 {
						// restart only after it's been used some, to make sure it's not necessary otherwise
						digester.Restart()
					}
					i++
					size := int64(i * 32) // items for this archive will be bigger than the last one
					for _, itemType := range test.itemTypes {
						for int64(buf.Len()) < size {
							err := buf.WriteByte(byte(buf.Len() % 256))
							require.Nil(t, err, "error padding content buffer: %v", err)
						}
						// feed it content that it will treat either as raw data ("") or expect to
						// look like a tarball ("file"/"dir")
						digester.Start(itemType)
						hasher := digester.Hash() // returns an io.WriteCloser
						require.NotNil(t, hasher, "digester returned a null hasher?")
						if itemType == "" {
							// write something that isn't an archive
							n, err := io.Copy(hasher, &buf)
							require.Nil(t, err, "error writing tar content to digester: %v", err)
							require.Equal(t, size, n, "short write writing tar content to digester")
							continue
						}
						// write an archive
						var written bytes.Buffer // a copy of the archive we're generating and digesting
						hasher = &struct {
							io.Writer
							io.Closer
						}{
							Writer: io.MultiWriter(hasher, &written), // splice into the writer
							Closer: hasher,
						}
						if filtered {
							// wrap the WriteCloser in another WriteCloser
							hasher = newTarFilterer(hasher, func(hdr *tar.Header) { hdr.ModTime = zero })
							require.NotNil(t, hasher, "newTarFilterer returned a null WriteCloser?")
						}
						// write this item as an archive
						tw := tar.NewWriter(hasher)
						hdr := &tar.Header{
							Name:     "content",
							Size:     size,
							Mode:     0640,
							ModTime:  time.Now(),
							Typeflag: tar.TypeReg,
						}
						err := tw.WriteHeader(hdr)
						require.Nil(t, err, "error writing tar header to digester: %v", err)
						n, err := io.Copy(tw, &buf)
						require.Nil(t, err, "error writing tar content to digester: %v", err)
						require.Equal(t, size, n, "short write writing tar content to digester")
						err = tw.Flush()
						require.Nil(t, err, "error flushing tar content to digester: %v", err)
						err = tw.Close()
						require.Nil(t, err, "error closing tar archive being written digester: %v", err)
						if filtered {
							// the ContentDigester can close its own if we don't explicitly ask it to,
							// but if we wrapped it in a filter, we have to close the filter to clean
							// up the filter, so we can't skip it to exercise that logic; we have to
							// leave that for the corresponding unfiltered case to try
							hasher.Close()
						}
						// now read the archive back
						tr := tar.NewReader(&written)
						require.NotNil(t, tr, "unable to read byte buffer?")
						hdr, err = tr.Next()
						for err == nil {
							var n int64
							if filtered {
								// the filter should have set the modtime to unix 0
								require.Equal(t, zero, hdr.ModTime, "timestamp for entry should have been zero")
							} else {
								// the filter should have left modtime to "roughly now"
								require.NotEqual(t, zero, hdr.ModTime, "timestamp for entry should not have been zero")
							}
							n, err = io.Copy(ioutil.Discard, tr)
							require.Nil(t, err, "error reading tar content from buffer: %v", err)
							require.Equal(t, hdr.Size, n, "short read reading tar content")
							hdr, err = tr.Next()
						}
						require.Equal(t, io.EOF, err, "finished reading archive with %v, not EOF", err)
					}
					// check the composite digest type matches expectations and the value is not just the
					// digest of zero-length data, which is absolutely not what we wrote
					digestType, digestValue := digester.Digest()
					require.Equal(t, test.resultType, digestType, "expected to get a %q digest back for %v, got %q", test.resultType, test.itemTypes, digestType)
					require.NotEqual(t, digest.Canonical.FromBytes([]byte{}), digestValue, "digester wasn't fed any data")
					require.False(t, digester.isOpen(), "expected digester to have been closed with this usage pattern")
				})
			}
		})
	}
}
