package logging

import "sync"

// buffer represents a byte slice used for temporary storage
type buffer []byte

// bufpool is a sync.Pool for reusing buffer objects
var bufpool = sync.Pool{
	New: func() any {
		b := make(buffer, 0, 1024)
		return (*buffer)(&b)
	},
}

// newBuffer returns a new buffer from the pool
func newBuffer() *buffer {
	return bufpool.Get().(*buffer)
}

// Free returns the buffer to the pool if it's small enough
func (b *buffer) Free() {
	// To reduce peak allocation, return only smaller buffers to the pool.
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufpool.Put(b)
	}
}

// Write appends the given bytes to the buffer
func (b *buffer) Write(bytes []byte) (int, error) {
	*b = append(*b, bytes...)
	return len(bytes), nil
}

// WriteByte appends a single byte to the buffer
func (b *buffer) WriteByte(char byte) error {
	*b = append(*b, char)
	return nil
}

// WriteString appends a string to the buffer
func (b *buffer) WriteString(str string) (int, error) {
	*b = append(*b, str...)
	return len(str), nil
}

// WriteStringIf appends a string to the buffer only if the condition is true
func (b *buffer) WriteStringIf(ok bool, str string) (int, error) {
	if !ok {
		return 0, nil
	}
	return b.WriteString(str)
}
