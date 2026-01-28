// Package repl implements the TUI-based REPL for Vega.
package repl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/lexer"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/value"
	"github.com/mwantia/vega/pkg/vm"
)

// NewTUI creates a new TUI REPL model.
func NewTUI(v *vm.VirtualMachine, disasm bool) Model {
	ti := textinput.New()
	ti.Prompt = "" // Remove default "> " prompt
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 80

	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 100
	si.Width = 40

	return Model{
		vm:           v,
		textInput:    ti,
		searchInput:  si,
		history:      make([]HistoryEntry, 0),
		historyIndex: -1,
		commandIndex: 0,
		output:       make([]OutputLine, 0),
		showDisasm:   disasm,
		focus:        FocusInput,
		status:       StatusReady,
		statusMsg:    "Ready",
		width:        80,
		height:       24,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tea.SetWindowTitle("Vega REPL"),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = m.inputWidth()
		m.searchInput.Width = min(40, m.width-10)
		return m, nil
	}

	// Update text input
	if m.focus == FocusInput && !m.searchMode {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.searchMode {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode first
	if m.searchMode {
		return m.handleSearchKey(msg)
	}

	// Handle autocomplete
	if m.showAutocomplete {
		return m.handleAutocompleteKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		if m.status == StatusExecuting {
			// Cancel the running execution
			if m.execCancel != nil {
				m.execCancel()
			}
			m.vm.Cancel()
			m.status = StatusReady
			m.statusMsg = "Interrupted"
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "ctrl+d":
		if m.textInput.Value() == "" && !m.inMultiline {
			// Toggle disasm if at empty prompt
			m.showDisasm = !m.showDisasm
			if m.showDisasm {
				m.status = StatusDisasm
				m.statusMsg = "Disasm ON"
			} else {
				m.status = StatusReady
				m.statusMsg = "Disasm OFF"
			}
			return m, nil
		}
		// Delete character at cursor (handled by textinput)

	case "ctrl+r":
		m.searchMode = true
		m.focus = FocusSearch
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		m.searchResults = m.filterHistory("")
		m.searchCursor = 0
		return m, textinput.Blink

	case "ctrl+l":
		m.output = nil
		m.scrollOffset = 0
		return m, nil

	case "ctrl+o":
		// Toggle showing expression results
		m.showResults = !m.showResults
		if m.showResults {
			m.statusMsg = "Results ON"
		} else {
			m.statusMsg = "Results OFF"
		}
		return m, nil

	case "ctrl+u":
		m.textInput.SetValue("")
		return m, nil

	case "ctrl+k":
		// Kill to end of line
		val := m.textInput.Value()
		pos := m.textInput.Position()
		m.textInput.SetValue(val[:pos])
		return m, nil

	case "up":
		return m.navigateHistory(1), nil

	case "down":
		return m.navigateHistory(-1), nil

	case "tab":
		return m.triggerAutocomplete(), nil

	case "enter":
		return m.handleEnter()

	case "?":
		if m.textInput.Value() == "" {
			m.printHelp()
			return m, nil
		}
	}

	// Default: pass to text input
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		// Scroll up (show older content)
		m.scrollOffset += 3
		maxScroll := len(m.output)
		if m.scrollOffset > maxScroll {
			m.scrollOffset = maxScroll
		}
		return m, nil

	case tea.MouseButtonWheelDown:
		// Scroll down (show newer content)
		m.scrollOffset -= 3
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchMode = false
		m.focus = FocusInput
		m.textInput.Focus()
		return m, textinput.Blink

	case "enter":
		if len(m.searchResults) > 0 && m.searchCursor < len(m.searchResults) {
			idx := m.searchResults[m.searchCursor]
			if idx < len(m.history) {
				m.textInput.SetValue(m.history[idx].Input)
				m.textInput.CursorEnd()
			}
		}
		m.searchMode = false
		m.focus = FocusInput
		m.textInput.Focus()
		return m, textinput.Blink

	case "up":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
		return m, nil

	case "down":
		if m.searchCursor < len(m.searchResults)-1 {
			m.searchCursor++
		}
		return m, nil

	default:
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		m.searchResults = m.filterHistory(m.searchInput.Value())
		m.searchCursor = 0
		return m, cmd
	}
}

func (m Model) handleAutocompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showAutocomplete = false
		return m, nil

	case "tab", "enter":
		if len(m.suggestions) > 0 && m.suggestionCursor < len(m.suggestions) {
			m.insertSuggestion(m.suggestions[m.suggestionCursor])
		}
		m.showAutocomplete = false
		return m, nil

	case "up":
		if m.suggestionCursor > 0 {
			m.suggestionCursor--
		}
		return m, nil

	case "down":
		if m.suggestionCursor < len(m.suggestions)-1 {
			m.suggestionCursor++
		}
		return m, nil

	default:
		m.showAutocomplete = false
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	line := m.textInput.Value()
	m.textInput.SetValue("")

	// Handle special commands
	if !m.inMultiline {
		if handled, cmd := m.handleCommand(line); handled {
			return m, cmd
		}
	}

	// Track braces
	for _, ch := range line {
		switch ch {
		case '{':
			m.braceCount++
		case '}':
			m.braceCount--
		}
	}

	// Accumulate input
	if m.inMultiline {
		m.inputBuffer.WriteString("\n")
	}
	m.inputBuffer.WriteString(line)

	// Check if we need more input
	if m.braceCount > 0 {
		m.inMultiline = true
		return m, nil
	}

	// Execute complete input
	input := m.inputBuffer.String()
	m.inputBuffer.Reset()
	m.inMultiline = false
	m.braceCount = 0

	if strings.TrimSpace(input) != "" {
		m.execute(input)
	}

	return m, nil
}

func (m *Model) handleCommand(line string) (bool, tea.Cmd) {
	line = strings.TrimSpace(line)

	switch line {
	case "quit":
		m.quitting = true
		return true, tea.Quit

	case "help":
		m.printHelp()
		return true, nil

	case "history":
		m.printHistory()
		return true, nil

	case "clear":
		m.output = nil
		m.scrollOffset = 0
		return true, nil

	case "vars":
		m.addOutput("Variable inspection not yet implemented.", OutputInfo, -1)
		return true, nil
	}

	return false, nil
}

func (m *Model) execute(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	// Add command to output
	cmdIdx := m.commandIndex
	m.addOutput(fmt.Sprintf("[%d] vega $ %s", cmdIdx, input), OutputCommand, cmdIdx)

	// Add to history
	m.history = append(m.history, HistoryEntry{
		Index: cmdIdx,
		Input: input,
		Exec:  time.Now(),
	})
	m.commandIndex++
	m.historyIndex = -1

	// Lexer
	l := lexer.New(input)
	tokens, err := l.Tokenize()
	if err != nil {
		m.addOutput(fmt.Sprintf("Syntax error: %v", err), OutputError, cmdIdx)
		m.status = StatusError
		m.statusMsg = "Syntax error"
		return
	}

	// Parser
	p := parser.New(tokens)
	program, err := p.Parse()
	if err != nil {
		m.addOutput(fmt.Sprintf("Parse error: %v", err), OutputError, cmdIdx)
		m.status = StatusError
		m.statusMsg = "Parse error"
		return
	}

	// Compiler
	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		m.addOutput(fmt.Sprintf("Compile error: %v", err), OutputError, cmdIdx)
		m.status = StatusError
		m.statusMsg = "Compile error"
		return
	}

	// Store bytecode for disasm
	m.lastBytecode = bytecode.Disassemble()
	m.disasmScroll = 0

	// Capture output
	m.outputCapture.Reset()
	m.errorCapture.Reset()

	// Create output writer that captures to our buffer
	m.vm.SetStdout(&m.outputCapture)
	m.vm.SetStderr(&m.errorCapture)

	// Create a fresh cancellable context for this execution
	m.execCtx, m.execCancel = context.WithCancel(context.Background())
	m.vm.SetContext(m.execCtx)

	// Execute
	m.status = StatusExecuting
	m.statusMsg = "Executing..."

	_, err = m.vm.Run(bytecode)

	// Check if context was cancelled before cleanup (indicates user interrupt)
	wasInterrupted := m.execCtx.Err() == context.Canceled

	// Clean up execution context
	if m.execCancel != nil {
		m.execCancel()
		m.execCancel = nil
	}

	if err != nil {
		// Check if this was a cancellation (user interrupted)
		if err == context.Canceled || wasInterrupted {
			m.addOutput("Execution interrupted", OutputInfo, cmdIdx)
			m.status = StatusReady
			m.statusMsg = "Interrupted"
			return
		}
		m.addOutput(fmt.Sprintf("Runtime error: %v", err), OutputError, cmdIdx)
		m.status = StatusError
		m.statusMsg = "Runtime error"
		return
	}

	// Capture stdout output (always shown - this is from print calls)
	if out := m.outputCapture.String(); out != "" {
		for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
			m.addOutput(line, OutputNormal, cmdIdx)
		}
	}

	// Capture stderr output (always shown)
	if out := m.errorCapture.String(); out != "" {
		for _, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
			m.addOutput(line, OutputError, cmdIdx)
		}
	}

	// Show expression result only if showResults is enabled
	if m.showResults {
		if result := m.vm.LastPopped(); result != nil {
			if result.Type() != "nil" {
				formatted := m.formatValue(result, 0)
				for _, line := range strings.Split(formatted, "\n") {
					m.addOutput(line, OutputNormal, cmdIdx)
				}
			}
		}
	}

	m.status = StatusReady
	m.statusMsg = "Ready"

	if m.showDisasm {
		m.status = StatusDisasm
		m.statusMsg = "Disasm ON"
	}
}

func (m *Model) addOutput(text string, typ OutputType, histIdx int) {
	m.output = append(m.output, OutputLine{
		Text:       text,
		Type:       typ,
		HistoryIdx: histIdx,
	})
	// Reset scroll to bottom when new output is added
	m.scrollOffset = 0
}

func (m *Model) printHelp() {
	m.addOutput("Commands:", OutputInfo, -1)
	m.addOutput("  help     - Show this help message", OutputInfo, -1)
	m.addOutput("  quit     - Exit the REPL (also: exit)", OutputInfo, -1)
	m.addOutput("  history  - Show command history", OutputInfo, -1)
	m.addOutput("  clear    - Clear the screen", OutputInfo, -1)
	m.addOutput("  vars     - Show defined variables", OutputInfo, -1)
	m.addOutput("", OutputInfo, -1)
	m.addOutput("Key bindings:", OutputInfo, -1)
	m.addOutput("  Ctrl+R   - Search history", OutputInfo, -1)
	m.addOutput("  Ctrl+L   - Clear screen", OutputInfo, -1)
	m.addOutput("  Ctrl+O   - Toggle expression results (e.g. readdir output)", OutputInfo, -1)
	m.addOutput("  Ctrl+D   - Toggle disasm (at empty prompt)", OutputInfo, -1)
	m.addOutput("  Ctrl+U   - Clear current line", OutputInfo, -1)
	m.addOutput("  Ctrl+C   - Quit", OutputInfo, -1)
	m.addOutput("  Tab      - Autocomplete", OutputInfo, -1)
	m.addOutput("  Up/Down  - Navigate history", OutputInfo, -1)
	m.addOutput("  Scroll   - Mouse wheel to scroll output", OutputInfo, -1)
	m.addOutput("", OutputInfo, -1)
	m.addOutput("Tip: Hold Shift while selecting text to copy in terminal", OutputInfo, -1)
}

func (m *Model) printHistory() {
	if len(m.history) == 0 {
		m.addOutput("No history.", OutputInfo, -1)
		return
	}
	for _, h := range m.history {
		m.addOutput(fmt.Sprintf("[%d] %s", h.Index, h.Input), OutputInfo, -1)
	}
}

func (m Model) navigateHistory(direction int) Model {
	if len(m.history) == 0 {
		return m
	}

	// Save current input when starting navigation
	if m.historyIndex == -1 {
		m.savedInput = m.textInput.Value()
	}

	newIndex := m.historyIndex + direction

	if newIndex < -1 {
		newIndex = -1
	}
	if newIndex >= len(m.history) {
		newIndex = len(m.history) - 1
	}

	m.historyIndex = newIndex

	if m.historyIndex == -1 {
		m.textInput.SetValue(m.savedInput)
	} else {
		// Navigate from most recent (len-1) backwards
		actualIdx := len(m.history) - 1 - m.historyIndex
		if actualIdx >= 0 && actualIdx < len(m.history) {
			m.textInput.SetValue(m.history[actualIdx].Input)
		}
	}
	m.textInput.CursorEnd()

	return m
}

func (m Model) filterHistory(query string) []int {
	var results []int
	query = strings.ToLower(query)
	for i := len(m.history) - 1; i >= 0; i-- {
		if query == "" || strings.Contains(strings.ToLower(m.history[i].Input), query) {
			results = append(results, i)
		}
	}
	return results
}

func (m Model) triggerAutocomplete() Model {
	input := m.textInput.Value()
	pos := m.textInput.Position()

	// Find the word being typed
	start := pos
	for start > 0 && isWordChar(input[start-1]) {
		start--
	}
	prefix := input[start:pos]

	if prefix == "" {
		return m
	}

	// Get suggestions
	m.suggestions = m.getSuggestions(prefix)
	if len(m.suggestions) > 0 {
		m.showAutocomplete = true
		m.suggestionCursor = 0
	}

	return m
}

func (m *Model) insertSuggestion(suggestion string) {
	input := m.textInput.Value()
	pos := m.textInput.Position()

	// Find the word being typed
	start := pos
	for start > 0 && isWordChar(input[start-1]) {
		start--
	}

	// Replace prefix with suggestion
	newInput := input[:start] + suggestion + input[pos:]
	m.textInput.SetValue(newInput)
	m.textInput.SetCursor(start + len(suggestion))
}

func (m Model) getSuggestions(prefix string) []string {
	keywords := []string{
		"fn", "if", "else", "for", "while", "return", "true", "false", "nil",
		"in", "range", "break", "continue", "let", "const",
	}

	builtins := []string{
		"stdin", "stdout", "stderr", "print", "println", "input",
		"type", "string", "int", "float", "bool", "len", "push", "pop", "keys",
		"upper", "lower", "trim", "split", "join", "contains", "startswith", "endswith",
		"replace", "index", "range", "assert", "read", "write", "stat", "lookup",
		"readdir", "createdir", "remdir", "unlink", "rename", "open", "exec", "sexec", "capture",
	}

	var suggestions []string
	prefix = strings.ToLower(prefix)

	for _, kw := range keywords {
		if strings.HasPrefix(strings.ToLower(kw), prefix) {
			suggestions = append(suggestions, kw)
		}
	}

	for _, b := range builtins {
		if strings.HasPrefix(strings.ToLower(b), prefix) {
			suggestions = append(suggestions, b)
		}
	}

	// Limit suggestions
	if len(suggestions) > 8 {
		suggestions = suggestions[:8]
	}

	return suggestions
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func (m Model) inputWidth() int {
	if m.showDisasm {
		return int(float64(m.width) * 0.65)
	}
	return m.width - 2
}

// View implements tea.Model.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Calculate dimensions
	mainWidth := m.width
	disasmWidth := 0
	if m.showDisasm {
		disasmWidth = int(float64(m.width) * 0.35)
		mainWidth = m.width - disasmWidth - 1
	}

	// Reserve height for status bar
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render main pane (output + input)
	mainPane := m.renderMainPane(mainWidth, contentHeight)

	// Render disasm pane if enabled
	var disasmPane string
	if m.showDisasm {
		disasmPane = m.renderDisasmPane(disasmWidth, contentHeight)
	}

	// Combine panes horizontally
	if m.showDisasm {
		mainLines := strings.Split(mainPane, "\n")
		disasmLines := strings.Split(disasmPane, "\n")

		// Pad to same height
		for len(mainLines) < contentHeight {
			mainLines = append(mainLines, "")
		}
		for len(disasmLines) < contentHeight {
			disasmLines = append(disasmLines, "")
		}

		separator := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")

		for i := 0; i < contentHeight; i++ {
			ml := padOrTruncate(mainLines[i], mainWidth)
			dl := padOrTruncate(disasmLines[i], disasmWidth)

			b.WriteString(ml)
			b.WriteString(separator)
			b.WriteString(dl)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(mainPane)
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(m.renderStatusBar())

	// Overlay search modal if active
	if m.searchMode {
		return m.renderSearchOverlay(b.String())
	}

	// Overlay autocomplete if active
	if m.showAutocomplete {
		return m.renderAutocompleteOverlay(b.String())
	}

	return b.String()
}

func (m Model) renderMainPane(width, height int) string {
	var lines []string

	// Add output lines with wrapping
	for _, out := range m.output {
		switch out.Type {
		case OutputCommand:
			// Split styling: [N] in grey, "vega $" bold, command in grey
			// Format is "[N] vega $ command"
			text := out.Text
			if idx := strings.Index(text, "vega $ "); idx != -1 {
				prefix := text[:idx]             // "[N] "
				prompt := "vega $ "              // "vega $ "
				cmd := text[idx+len("vega $ "):] // the actual command

				styledLine := indexStyle.Render(prefix) +
					promptStyle.Render(prompt) +
					historyCommandStyle.Render(cmd)
				lines = append(lines, styledLine)
			} else {
				lines = append(lines, historyCommandStyle.Render(text))
			}
		case OutputError:
			wrapped := wrapText(out.Text, width)
			for _, wl := range wrapped {
				lines = append(lines, errorStyle.Render(wl))
			}
		case OutputInfo:
			wrapped := wrapText(out.Text, width)
			for _, wl := range wrapped {
				lines = append(lines, infoStyle.Render(wl))
			}
		default:
			// Normal output - wrap long lines
			wrapped := wrapText(out.Text, width)
			for _, wl := range wrapped {
				lines = append(lines, resultStyle.Render(wl))
			}
		}
	}

	// Add current input with prompt
	prompt := m.renderPrompt()

	// Use the textinput's built-in view which handles cursor properly
	inputLine := prompt + m.textInput.View()
	lines = append(lines, inputLine)

	// Apply scroll offset and height constraints
	totalLines := len(lines)
	if totalLines > height {
		// Calculate the visible window based on scroll offset
		// scrollOffset = 0 means we're at the bottom (most recent)
		// scrollOffset > 0 means we're scrolled up
		endIdx := totalLines - m.scrollOffset
		if endIdx < height {
			endIdx = height
		}
		if endIdx > totalLines {
			endIdx = totalLines
		}
		startIdx := endIdx - height
		if startIdx < 0 {
			startIdx = 0
		}
		lines = lines[startIdx:endIdx]
	}

	// Pad remaining height
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderPrompt() string {
	if m.inMultiline {
		return continuePromptStyle.Render("      >> ")
	}
	return indexStyle.Render(fmt.Sprintf("[%d] ", m.commandIndex)) + promptStyle.Render("vega $ ")
}

func (m Model) renderDisasmPane(width, height int) string {
	var lines []string

	// Title
	title := disasmTitleStyle.Render("Bytecode (Last Exec)")
	lines = append(lines, padOrTruncate(title, width))

	// Disasm content
	if m.lastBytecode == "" {
		lines = append(lines, disasmStyle.Render("(no bytecode)"))
	} else {
		disasmLines := strings.Split(m.lastBytecode, "\n")
		for _, line := range disasmLines {
			styled := m.styleBytecode(line)
			lines = append(lines, truncateToWidth(styled, width))
		}
	}

	// Pad to height
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:height], "\n")
}

func (m Model) styleBytecode(line string) string {
	// Simple styling: address in gray, opcode in cyan, operand in yellow
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return line
	}

	var styled []string
	for i, part := range parts {
		if i == 0 && strings.HasPrefix(part, "0x") {
			styled = append(styled, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(part))
		} else if i == 1 {
			styled = append(styled, opcodeStyle.Render(part))
		} else {
			styled = append(styled, operandStyle.Render(part))
		}
	}
	return strings.Join(styled, " ")
}

func (m Model) renderStatusBar() string {
	var icon string
	switch m.status {
	case StatusReady:
		icon = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	case StatusError:
		icon = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("✗")
	case StatusExecuting:
		icon = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("⏳")
	case StatusDisasm:
		icon = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("📊")
	}

	left := fmt.Sprintf("%s %s | History: %d", icon, m.statusMsg, len(m.history))

	// Show scroll indicator if scrolled up
	if m.scrollOffset > 0 {
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(fmt.Sprintf(" [↑%d]", m.scrollOffset))
		left += scrollInfo
	}

	// Show results indicator
	if m.showResults {
		resultsInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(" [RESULTS]")
		left += resultsInfo
	}

	var hints []string
	hints = append(hints, "[↑↓] Navigate")
	hints = append(hints, "[Ctrl+O] Results")
	hints = append(hints, "[Ctrl+R] Search")
	if m.showDisasm {
		hints = append(hints, "[Ctrl+D] Hide Disasm")
	} else {
		hints = append(hints, "[Ctrl+D] Show Disasm")
	}
	hints = append(hints, "[?] Help")

	right := strings.Join(hints, " | ")

	// Calculate padding
	padding := m.width - len(left) - len(right) - 4
	if padding < 1 {
		padding = 1
	}

	return statusBarStyle.Width(m.width).Render(left + strings.Repeat(" ", padding) + right)
}

func (m Model) renderSearchOverlay(base string) string {
	// Build search box content
	var content strings.Builder
	content.WriteString("Search history:\n")
	content.WriteString("> " + m.searchInput.View() + "\n\n")
	content.WriteString("Results:\n")

	maxResults := 5
	for i, idx := range m.searchResults {
		if i >= maxResults {
			break
		}
		style := searchResultStyle
		if i == m.searchCursor {
			style = searchSelectedStyle
		}
		if idx < len(m.history) {
			content.WriteString(style.Render(fmt.Sprintf(" [%d] %s", m.history[idx].Index, m.history[idx].Input)) + "\n")
		}
	}

	content.WriteString("\n[Enter] Select | [Esc] Cancel")

	box := searchBoxStyle.Render(content.String())

	// Center the box
	boxLines := strings.Split(box, "\n")
	baseLines := strings.Split(base, "\n")

	startY := (m.height - len(boxLines)) / 2
	startX := (m.width - lipgloss.Width(boxLines[0])) / 2
	if startX < 0 {
		startX = 0
	}

	for i, boxLine := range boxLines {
		y := startY + i
		if y >= 0 && y < len(baseLines) {
			baseLine := baseLines[y]
			// Overlay the box line
			newLine := ""
			if startX > 0 && len(baseLine) > 0 {
				newLine = baseLine[:min(startX, len(baseLine))]
			}
			for len(newLine) < startX {
				newLine += " "
			}
			newLine += boxLine
			baseLines[y] = newLine
		}
	}

	return strings.Join(baseLines, "\n")
}

func (m Model) renderAutocompleteOverlay(base string) string {
	if len(m.suggestions) == 0 {
		return base
	}

	var content strings.Builder
	for i, s := range m.suggestions {
		style := suggestionStyle
		if i == m.suggestionCursor {
			style = suggestionSelectedStyle
		}
		content.WriteString(style.Render(s) + "\n")
	}

	box := autocompleteStyle.Render(strings.TrimSuffix(content.String(), "\n"))
	boxLines := strings.Split(box, "\n")
	baseLines := strings.Split(base, "\n")

	// Position autocomplete near cursor
	promptLen := len(fmt.Sprintf("[%d] vega $ ", m.commandIndex))
	cursorX := promptLen + m.textInput.Position()
	cursorY := len(m.output)

	startY := cursorY + 1
	startX := cursorX

	// Clamp to screen
	if startY+len(boxLines) > m.height {
		startY = cursorY - len(boxLines)
	}
	if startX+lipgloss.Width(boxLines[0]) > m.width {
		startX = m.width - lipgloss.Width(boxLines[0])
	}

	for i, boxLine := range boxLines {
		y := startY + i
		if y >= 0 && y < len(baseLines) {
			baseLine := baseLines[y]
			newLine := ""
			if startX > 0 && len(baseLine) > 0 {
				newLine = baseLine[:min(startX, len(baseLine))]
			}
			for len(newLine) < startX {
				newLine += " "
			}
			newLine += boxLine
			baseLines[y] = newLine
		}
	}

	return strings.Join(baseLines, "\n")
}

// padOrTruncate ensures a string is exactly the given width.
// Uses ANSI-aware width calculation.
func padOrTruncate(s string, width int) string {
	currentWidth := lipgloss.Width(s)
	if currentWidth == width {
		return s
	}
	if currentWidth < width {
		return s + strings.Repeat(" ", width-currentWidth)
	}
	// Truncate - need to be careful with ANSI codes
	return truncateToWidth(s, width)
}

// truncateToWidth truncates a string to fit within the given width.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}

	// Strip ANSI codes for measurement, then rebuild
	var result strings.Builder
	currentWidth := 0

	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > width {
			break
		}
		result.WriteRune(r)
		currentWidth += runeWidth
	}

	return result.String()
}

// wrapText wraps text to fit within the given width.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}

	if lipgloss.Width(s) <= width {
		return []string{s}
	}

	var lines []string
	var currentLine strings.Builder
	currentWidth := 0

	words := strings.Fields(s)
	for i, word := range words {
		wordWidth := lipgloss.Width(word)

		// If word itself is too long, break it
		if wordWidth > width {
			if currentWidth > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentWidth = 0
			}
			// Break long word into chunks
			var chunk strings.Builder
			chunkWidth := 0
			for _, r := range word {
				runeWidth := lipgloss.Width(string(r))
				if currentWidth+runeWidth > width {
					break
				}
				chunk.WriteRune(r)
				currentWidth += runeWidth
			}

			if chunk.Len() > 0 {
				currentLine.WriteString(chunk.String())
				currentWidth = chunkWidth
			}
			continue
		}

		// Check if word fits on current line
		spaceWidth := 0
		if currentWidth > 0 {
			spaceWidth = 1
		}

		if currentWidth+spaceWidth+wordWidth <= width {
			if currentWidth > 0 {
				currentLine.WriteString(" ")
				currentWidth++
			}
			currentLine.WriteString(word)
			currentWidth += wordWidth
		} else {
			// Start new line
			if currentWidth > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			currentLine.WriteString(word)
			currentWidth = wordWidth
		}

		_ = i // silence unused variable warning
	}

	if currentWidth > 0 {
		lines = append(lines, currentLine.String())
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

// formatValue pretty-prints a value with indentation and colors.
func (m *Model) formatValue(v value.Value, indent int) string {
	switch val := v.(type) {
	case *value.Metadata:
		return m.formatMetadata(val, indent)
	case *value.Map:
		return m.formatMap(val, indent)
	case *value.Array:
		return m.formatArray(val, indent)
	case *value.String:
		return stringValueStyle.Render(fmt.Sprintf("%q", val.String())) + typeAnnotationStyle.Render(" (string)")
	case *value.Short:
		return numberValueStyle.Render(val.String()) + typeAnnotationStyle.Render(" (short)")
	case *value.Integer:
		return numberValueStyle.Render(val.String()) + typeAnnotationStyle.Render(" (integer)")
	case *value.Long:
		return numberValueStyle.Render(val.String()) + typeAnnotationStyle.Render(" (long)")
	case *value.Float:
		return numberValueStyle.Render(val.String()) + typeAnnotationStyle.Render(" (float)")
	case *value.Boolean:
		return boolValueStyle.Render(val.String()) + typeAnnotationStyle.Render(" (boolean)")
	case *value.NilValue:
		return nilValueStyle.Render("nil")
	default:
		return unknownValueStyle.Render(val.String()) + typeAnnotationStyle.Render(fmt.Sprintf(" (%s)", val.Type()))
	}
}

// formatMap pretty-prints a map value.
func (m *Model) formatMap(mv *value.Map, indent int) string {
	if len(mv.Pairs) == 0 {
		return bracketStyle.Render("{}")
	}

	// For small maps (1-2 items), keep on one line
	if len(mv.Pairs) <= 2 {
		var parts []string
		for _, k := range mv.Order {
			if v, ok := mv.Pairs[k]; ok {
				key := keyStyle.Render(k)
				val := m.formatValueInline(v)
				parts = append(parts, key+": "+val)
			}
		}
		return bracketStyle.Render("{") + " " + strings.Join(parts, ", ") + " " + bracketStyle.Render("}")
	}

	// For larger maps, use multiple lines with indentation
	indentStr := strings.Repeat("  ", indent)
	innerIndent := strings.Repeat("  ", indent+1)

	var lines []string
	lines = append(lines, bracketStyle.Render("{"))

	for i, k := range mv.Order {
		if v, ok := mv.Pairs[k]; ok {
			key := keyStyle.Render(k)
			val := m.formatValue(v, indent+1)

			comma := ","
			if i == len(mv.Order)-1 {
				comma = ""
			}

			// Handle multi-line values
			if strings.Contains(val, "\n") {
				lines = append(lines, innerIndent+key+": "+val+comma)
			} else {
				lines = append(lines, innerIndent+key+": "+val+comma)
			}
		}
	}

	lines = append(lines, indentStr+bracketStyle.Render("}"))
	return strings.Join(lines, "\n")
}

// formatArray pretty-prints an array value.
func (m *Model) formatArray(av *value.Array, indent int) string {
	if len(av.Elements) == 0 {
		return bracketStyle.Render("[]")
	}

	// For small arrays (1-5 simple items), keep on one line
	if len(av.Elements) <= 5 && !m.hasNestedStructures(av) {
		var parts []string
		for _, e := range av.Elements {
			parts = append(parts, m.formatValueInline(e))
		}
		return bracketStyle.Render("[") + strings.Join(parts, ", ") + bracketStyle.Render("]")
	}

	// For larger arrays, use multiple lines
	indentStr := strings.Repeat("  ", indent)
	innerIndent := strings.Repeat("  ", indent+1)

	var lines []string
	lines = append(lines, bracketStyle.Render("["))

	for i, e := range av.Elements {
		val := m.formatValue(e, indent+1)
		comma := ","
		if i == len(av.Elements)-1 {
			comma = ""
		}
		lines = append(lines, innerIndent+val+comma)
	}

	lines = append(lines, indentStr+bracketStyle.Render("]"))
	return strings.Join(lines, "\n")
}

// formatMetadata pretty-prints a metadata value.
func (m *Model) formatMetadata(mv *value.Metadata, indent int) string {
	indentStr := strings.Repeat("  ", indent)
	innerIndent := strings.Repeat("  ", indent+1)

	meta := mv.Meta
	typeStr := string(meta.GetType())

	var lines []string
	lines = append(lines, typeAnnotationStyle.Render("metadata")+" "+bracketStyle.Render("{"))

	// Core fields
	lines = append(lines, innerIndent+keyStyle.Render("key")+": "+stringValueStyle.Render(fmt.Sprintf("%q", meta.Key))+",")
	lines = append(lines, innerIndent+keyStyle.Render("type")+": "+stringValueStyle.Render(fmt.Sprintf("%q", typeStr))+",")
	lines = append(lines, innerIndent+keyStyle.Render("size")+": "+numberValueStyle.Render(fmt.Sprintf("%d", meta.Size))+",")

	// Mode and permissions
	lines = append(lines, innerIndent+keyStyle.Render("mode")+": "+numberValueStyle.Render(fmt.Sprintf("%o", meta.Mode))+",")

	// Timestamps
	lines = append(lines, innerIndent+keyStyle.Render("modified")+": "+stringValueStyle.Render(fmt.Sprintf("%q", meta.ModifyTime.Format("2006-01-02 15:04:05")))+",")
	lines = append(lines, innerIndent+keyStyle.Render("created")+": "+stringValueStyle.Render(fmt.Sprintf("%q", meta.CreateTime.Format("2006-01-02 15:04:05")))+",")

	// Content type if set
	if meta.ContentType != "" {
		lines = append(lines, innerIndent+keyStyle.Render("contentType")+": "+stringValueStyle.Render(fmt.Sprintf("%q", meta.ContentType))+",")
	}

	// ETag if set
	if meta.ETag != "" {
		lines = append(lines, innerIndent+keyStyle.Render("etag")+": "+stringValueStyle.Render(fmt.Sprintf("%q", meta.ETag))+",")
	}

	lines = append(lines, indentStr+bracketStyle.Render("}"))
	return strings.Join(lines, "\n")
}

// formatValueInline formats a value for inline display (no newlines).
func (m *Model) formatValueInline(v value.Value) string {
	switch val := v.(type) {
	case *value.String:
		return stringValueStyle.Render(val.String())
	case *value.Integer:
		return numberValueStyle.Render(val.String())
	case *value.Float:
		return numberValueStyle.Render(val.String())
	case *value.Boolean:
		return boolValueStyle.Render(val.String())
	case *value.NilValue:
		return nilValueStyle.Render("nil")
	case *value.Metadata:
		// Compact inline format for metadata
		return typeAnnotationStyle.Render("metadata") + bracketStyle.Render("{") +
			keyStyle.Render("key") + ": " + stringValueStyle.Render(fmt.Sprintf("%q", val.Meta.Key)) + ", " +
			keyStyle.Render("type") + ": " + stringValueStyle.Render(fmt.Sprintf("%q", val.Meta.GetType())) +
			bracketStyle.Render("}")
	case *value.Map:
		var parts []string
		for _, k := range val.Order {
			if vv, ok := val.Pairs[k]; ok {
				parts = append(parts, keyStyle.Render(k)+": "+m.formatValueInline(vv))
			}
		}
		return bracketStyle.Render("{") + strings.Join(parts, ", ") + bracketStyle.Render("}")
	case *value.Array:
		var parts []string
		for _, e := range val.Elements {
			parts = append(parts, m.formatValueInline(e))
		}
		return bracketStyle.Render("[") + strings.Join(parts, ", ") + bracketStyle.Render("]")
	default:
		return unknownValueStyle.Render(v.String())
	}
}

// hasNestedStructures checks if an array contains maps, arrays, or metadata.
func (m *Model) hasNestedStructures(av *value.Array) bool {
	for _, e := range av.Elements {
		switch e.(type) {
		case *value.Map, *value.Array, *value.Metadata:
			return true
		}
	}
	return false
}

// RunTUI starts the TUI REPL.
func RunTUI(v *vm.VirtualMachine, disasm bool) error {
	m := NewTUI(v, disasm)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
