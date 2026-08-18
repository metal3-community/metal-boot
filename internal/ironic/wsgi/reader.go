package wsgi

import (
	"errors"
	"io"
)

// requestReader is the contract the Python wsgi.input object reads through.
type requestReader interface {
	Read(p []byte) (int, error)
	Readline(p []byte) (int, error)
	Rewind() error
}

// bufferingReader buffers an initial chunk of the request body so the (slow,
// possibly remote) client read happens before a Python worker is occupied.
// Ported from wsgo's BufferingReader.
type bufferingReader struct {
	r       io.Reader
	buf     []byte
	initial []byte
	// partial records whether data beyond the initial buffer was consumed,
	// which makes Rewind impossible.
	partial bool
}

func newBufferingReader(r io.Reader, bufLen int) *bufferingReader {
	b := &bufferingReader{r: r}
	if bufLen > 0 {
		buf := make([]byte, bufLen)
		n, _ := io.ReadFull(r, buf)
		b.buf = buf[:n]
		b.initial = b.buf
	}
	return b
}

func (b *bufferingReader) Read(p []byte) (int, error) {
	if len(b.buf) > 0 {
		toRead := min(len(b.buf), len(p))
		copy(p, b.buf[:toRead])
		b.buf = b.buf[toRead:]

		if toRead < len(p) {
			n, err := b.r.Read(p[toRead:])
			if n > 0 {
				b.partial = true
			}
			return toRead + n, err
		}
		return toRead, nil
	}

	n, err := b.r.Read(p)
	if n > 0 {
		b.partial = true
	}
	return n, err
}

func (b *bufferingReader) Readline(p []byte) (int, error) {
	total := 0
	for {
		if len(b.buf) == 0 {
			buf := make([]byte, 8192)
			n, _ := io.ReadFull(b.r, buf)
			b.buf = buf[:n]
			if n == 0 {
				return total, io.EOF
			}
			b.partial = true
		}

		var i int
		limit := min(len(p), len(b.buf))
		for i = 0; i < limit; i++ {
			p[i] = b.buf[i]
			if b.buf[i] == '\n' {
				b.buf = b.buf[i+1:]
				return total + i + 1, nil
			}
		}

		b.buf = b.buf[i:]
		p = p[i:]
		total += i

		if len(p) == 0 {
			return total, nil
		}
	}
}

func (b *bufferingReader) Rewind() error {
	if b.partial {
		return errors.New("request body was read past the buffered prefix")
	}
	b.buf = b.initial
	return nil
}
