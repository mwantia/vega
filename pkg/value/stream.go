// Package value defines the runtime value types for Vega.
package value

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

// Stream represents an I/O stream (stdin, stdout, file handle, etc.)
type Stream struct {
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

var _ Methodable = (*Stream)(nil)

// NewInputStream creates a read-only stream.
func NewInputStream(name string, r io.Reader) *Stream {
	return &Stream{
		name:     name,
		reader:   r,
		bufRead:  bufio.NewReader(r),
		canRead:  true,
		canWrite: false,
	}
}

// NewOutputStream creates a write-only stream.
func NewOutputStream(name string, w io.Writer) *Stream {
	return &Stream{
		name:     name,
		writer:   w,
		canRead:  false,
		canWrite: true,
	}
}

// NewStream creates a stream with custom read/write capabilities.
func NewStream(name string, r io.Reader, w io.Writer, c io.Closer) *Stream {
	s := &Stream{
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
func (s *Stream) Type() string {
	return TypeStream
}

// String returns a string representation.
func (s *Stream) String() string {
	status := ""
	if s.closed {
		status = " (closed)"
	}
	return fmt.Sprintf("<stream:%s%s>", s.name, status)
}

// Boolean returns true if the stream is open.
func (s *Stream) Boolean() bool {
	return !s.closed
}

// Equal compares two streams by identity.
func (s *Stream) Equal(other Value) bool {
	if o, ok := other.(*Stream); ok {
		return s == o
	}
	return false
}

func (v *Stream) Method(name string, args []Value) (Value, error) {
	switch name {
	case "canread":
		// returns true if the stream supports reading
		return NewBoolean(v.CanRead()), nil
	case "canwrite":
		// returns true if the stream supports writing
		return NewBoolean(v.CanWrite()), nil
	case "isclosed":
		// returns true if the stream is closed
		return NewBoolean(v.IsClosed()), nil
	case "read":
		// reads all available data from the stream
		return v.Read()
	case "write":
		// writes data to the stream
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		return v.Write(args[0])
	case "readln":
		// reads a single line from the stream
		return v.ReadLine()
	case "writeln":
		// writes data followed by a newline
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		return v.WriteLine(args[0])
	case "readn":
		// reads n bytes from the stream
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		if i, ok := args[0].(*Integer); ok {
			n := int(i.Value)
			return v.ReadN(n)
		}
		return nil, fmt.Errorf("method '%s' first argument must be 'int', got '%s'", name, args[0].Type())
	case "copy":
		// copy all data from this stream to dest stream
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		dest, ok := args[0].(*Stream)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'stream', got '%s'", name, args[0].Type())
		}
		if !v.CanRead() {
			return nil, fmt.Errorf("source stream is not readable")
		}
		if !dest.CanWrite() {
			return nil, fmt.Errorf("destination stream is not writable")
		}
		n, err := io.Copy(dest.Writer(), v.Reader())
		if err != nil {
			return nil, fmt.Errorf("copy failed: %w", err)
		}
		return NewInteger(n), nil
	case "flush":
		// flushes any buffered data
		return NewNil(), v.Flush()
	case "close":
		// closes the stream
		return NewNil(), v.Close()
	}

	return nil, fmt.Errorf("unknown method-map: '%s'", name)
}

// Name returns the stream name.
func (s *Stream) Name() string {
	return s.name
}

// CanRead returns true if the stream supports reading.
func (s *Stream) CanRead() bool {
	return s.canRead && !s.closed
}

// CanWrite returns true if the stream supports writing.
func (s *Stream) CanWrite() bool {
	return s.canWrite && !s.closed
}

// IsClosed returns true if the stream is closed.
func (s *Stream) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Read reads all available data from the stream.
func (s *Stream) Read() (Value, error) {
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
func (s *Stream) ReadLine() (Value, error) {
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
func (s *Stream) ReadN(n int) (Value, error) {
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
func (s *Stream) Write(data Value) (Value, error) {
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

	return NewInteger(int64(n)), nil
}

// WriteLine writes data followed by a newline.
func (s *Stream) WriteLine(data Value) (Value, error) {
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

	return NewInteger(int64(n)), nil
}

// Close closes the stream.
func (s *Stream) Close() error {
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
func (s *Stream) Flush() error {
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
func (s *Stream) Reader() io.Reader {
	return s.reader
}

// Writer returns the underlying writer (for piping).
func (s *Stream) Writer() io.Writer {
	return s.writer
}
