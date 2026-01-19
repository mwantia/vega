// Package value defines the runtime value types for Vega.
package value

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// StreamValue represents an I/O stream (stdin, stdout, file handle, etc.)
type StreamValue struct {
	name     string
	reader   io.Reader
	writer   io.Writer
	closer   io.Closer
	bufRead  *bufio.Reader
	canRead  bool
	canWrite bool
	closed   bool
	mu       sync.Mutex
}

// NewInputStream creates a read-only stream.
func NewInputStream(name string, r io.Reader) *StreamValue {
	return &StreamValue{
		name:     name,
		reader:   r,
		bufRead:  bufio.NewReader(r),
		canRead:  true,
		canWrite: false,
	}
}

// NewOutputStream creates a write-only stream.
func NewOutputStream(name string, w io.Writer) *StreamValue {
	return &StreamValue{
		name:     name,
		writer:   w,
		canRead:  false,
		canWrite: true,
	}
}

// NewStream creates a stream with custom read/write capabilities.
func NewStream(name string, r io.Reader, w io.Writer, c io.Closer) *StreamValue {
	s := &StreamValue{
		name:     name,
		reader:   r,
		writer:   w,
		closer:   c,
		canRead:  r != nil,
		canWrite: w != nil,
	}
	if r != nil {
		s.bufRead = bufio.NewReader(r)
	}
	return s
}

// Type returns "stream".
func (s *StreamValue) Type() string {
	return TypeStream
}

// String returns a string representation.
func (s *StreamValue) String() string {
	status := ""
	if s.closed {
		status = " (closed)"
	}
	return fmt.Sprintf("<stream:%s%s>", s.name, status)
}

// Boolean returns true if the stream is open.
func (s *StreamValue) Boolean() bool {
	return !s.closed
}

// Equal compares two streams by identity.
func (s *StreamValue) Equal(other Value) bool {
	if o, ok := other.(*StreamValue); ok {
		return s == o
	}
	return false
}

// Name returns the stream name.
func (s *StreamValue) Name() string {
	return s.name
}

// CanRead returns true if the stream supports reading.
func (s *StreamValue) CanRead() bool {
	return s.canRead && !s.closed
}

// CanWrite returns true if the stream supports writing.
func (s *StreamValue) CanWrite() bool {
	return s.canWrite && !s.closed
}

// IsClosed returns true if the stream is closed.
func (s *StreamValue) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Read reads all available data from the stream.
func (s *StreamValue) Read() (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Nil, fmt.Errorf("stream is closed")
	}
	if !s.canRead {
		return Nil, fmt.Errorf("stream is not readable")
	}

	data, err := io.ReadAll(s.reader)
	if err != nil && err != io.EOF {
		return Nil, err
	}
	return NewString(string(data)), nil
}

// ReadLine reads a single line from the stream.
func (s *StreamValue) ReadLine() (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Nil, nil // EOF-like behavior
	}
	if !s.canRead {
		return Nil, fmt.Errorf("stream is not readable")
	}

	line, err := s.bufRead.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			if len(line) > 0 {
				return NewString(line), nil
			}
			return Nil, nil // Signal end of stream
		}
		return Nil, err
	}

	// Remove trailing newline
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	// Also remove \r if present (Windows line endings)
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	return NewString(line), nil
}

// ReadN reads n bytes from the stream.
func (s *StreamValue) ReadN(n int) (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Nil, fmt.Errorf("stream is closed")
	}
	if !s.canRead {
		return Nil, fmt.Errorf("stream is not readable")
	}

	buf := make([]byte, n)
	read, err := io.ReadFull(s.reader, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return Nil, err
	}

	return NewString(string(buf[:read])), nil
}

// Write writes data to the stream.
func (s *StreamValue) Write(data Value) (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Nil, fmt.Errorf("stream is closed")
	}
	if !s.canWrite {
		return Nil, fmt.Errorf("stream is not writable")
	}

	str := data.String()
	n, err := s.writer.Write([]byte(str))
	if err != nil {
		return Nil, err
	}

	return NewInt(int64(n)), nil
}

// WriteLine writes data followed by a newline.
func (s *StreamValue) WriteLine(data Value) (Value, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return Nil, fmt.Errorf("stream is closed")
	}
	if !s.canWrite {
		return Nil, fmt.Errorf("stream is not writable")
	}

	str := data.String() + "\n"
	n, err := s.writer.Write([]byte(str))
	if err != nil {
		return Nil, err
	}

	return NewInt(int64(n)), nil
}

// Close closes the stream.
func (s *StreamValue) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// Flush flushes any buffered data.
func (s *StreamValue) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if f, ok := s.writer.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// Reader returns the underlying reader (for piping).
func (s *StreamValue) Reader() io.Reader {
	return s.reader
}

// Writer returns the underlying writer (for piping).
func (s *StreamValue) Writer() io.Writer {
	return s.writer
}
