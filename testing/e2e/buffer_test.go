//go:build e2e

package e2e

import (
	"bytes"
	"io"
	"sync"
)

// syncBuffer is a bytes.Buffer that is safe for concurrent writes, which is
// required because os/exec writes stdout and stderr from separate goroutines
// and both are tee'd into a shared combined buffer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// multiWriter mirrors writes to every supplied writer.
func multiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}
