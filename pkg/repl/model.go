package repl

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/mwantia/vega/pkg/vm"
)

// Model is the Bubble Tea model for the TUI REPL.
type Model struct {
	// VM and execution
	vm *vm.VirtualMachine

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
	output        []OutputLine
	scrollOffset  int
	showResults   bool // When true, show expression results (like readdir output)

	// Bytecode disasm
	lastBytecode string
	showDisasm   bool
	disasmScroll int

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
