package repl

import "time"

// Focus indicates which pane has focus.
type Focus int

const (
	FocusInput Focus = iota
	FocusDisasm
	FocusSearch
)

// OutputType indicates the type of output line.
type OutputType int

const (
	OutputNormal OutputType = iota
	OutputError
	OutputInfo
	OutputCommand
)

// HistoryEntry represents a completed command.
type HistoryEntry struct {
	Index int
	Input string
	Exec  time.Time
}

// OutputLine represents a line in the output buffer.
type OutputLine struct {
	Text       string
	Type       OutputType
	HistoryIdx int
	Duration   time.Duration // Execution time for command lines
}

// Status represents the REPL status.
type Status int

const (
	StatusReady Status = iota
	StatusError
	StatusExecuting
	StatusDisasm
)
