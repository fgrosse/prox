package prox

import (
	"bytes"
	"io"
	"sync"
	"time"
)

const timestampFormat = "15:04:05"

// timestampWriter prefixes every complete line written to it with the current
// time (HH:MM:SS).
type timestampWriter struct {
	w   io.Writer
	mu  sync.Mutex
	buf []byte
}

func newTimestampWriter(w io.Writer) *timestampWriter {
	return &timestampWriter{w: w}
}

func (t *timestampWriter) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.buf = append(t.buf, p...)

	for {
		i := bytes.IndexByte(t.buf, '\n')
		if i == -1 {
			break
		}

		line := t.buf[:i+1]
		t.buf = t.buf[i+1:]

		stamped := make([]byte, 0, len(line)+9)
		stamped = append(stamped, time.Now().Format(timestampFormat)+" "...)
		stamped = append(stamped, line...)

		if _, err := t.w.Write(stamped); err != nil {
			return 0, err
		}
	}

	return len(p), nil
}
