package repl

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/mwantia/vega/pkg/alloc"
	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/vm"
)

// Model is the Bubble Tea model for the TUI REPL.
type Model struct {
	// VM and execution
	vm       vm.VirtualMachine
	compiler *compiler.Compiler

	// Input state
	textInput   textinput.Model
	inMultiline bool
	braceCount  int
	inputBuffer strings.Builder // Accumulated multiline input

	// History
	history      []HistoryEntry
	historyIndex int // -1 = current input, 0+ = history position
	savedInput   string
	commandIndex int // Next command index [N]

	// Output
	output       []OutputLine
	scrollOffset int

	// Bytecode disasm history (index 0 = most recent)
	bytecodeHistory []BytecodeEntry
	showDisasm      bool
	disasmScroll    int

	// Allocator hex viewer
	snapshotManager *alloc.SnapshotManager // persists for the session lifetime
	snapshotDelta   alloc.SnapshotDelta    // delta from the most recent Take()
	hexScroll       int

	// Search mode
	searchMode    bool
	searchInput   textinput.Model
	searchResults []int // Indices into history
	searchCursor  int

	// Autocomplete
	showAutocomplete bool
	suggestions      []string
	suggestionCursor int

	// UI state
	width     int
	height    int
	focus     Focus
	status    Status
	statusMsg string

	// Capture output
	outputCapture strings.Builder
	errorCapture  strings.Builder

	// Quit flag
	quitting bool
}
