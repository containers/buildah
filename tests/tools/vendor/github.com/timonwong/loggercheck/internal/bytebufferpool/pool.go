package bytebufferpool

import (
	"bytes"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func Get() *bytes.Buffer {
	buf := pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func Put(buf *bytes.Buffer) {
	pool.Put(buf)
}
