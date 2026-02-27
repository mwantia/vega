package repl

import "github.com/charmbracelet/lipgloss"

// Styles
var (
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true)

	indexStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	// Style for completed commands in history (grey, no bold)
	historyCommandStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	continuePromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	resultStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	disasmTitleStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("15")).
				Bold(true).
				Padding(0, 1)

	disasmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	opcodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14"))

	operandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

	searchResultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	searchSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("15"))

	autocompleteStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1)

	suggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	suggestionSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("15"))

	typeAnnotationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Italic(true)

	// Style for execution duration display
	durationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)

	// Hex viewer styles
	hexTitleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("56")).
			Foreground(lipgloss.Color("15")).
			Bold(true).
			Padding(0, 1)

	hexInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)

	// hexAddrStyle styles the 8-digit offset at the start of each row.
	hexAddrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	// hexFreeStyle styles bytes that belong to a free (unallocated) block.
	hexFreeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	// hexZeroStyle styles allocated bytes whose value is zero.
	hexZeroStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	// hexByteStyle styles allocated bytes with a non-zero value.
	hexByteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	// hexNewlyFreedStyle highlights bytes freed during the last execution.
	hexNewlyFreedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2"))

	// hexNewlyConsumedStyle highlights bytes freshly allocated during the last execution.
	hexNewlyConsumedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("9"))

	// hexWrittenStyle highlights already-allocated bytes whose value changed during the last execution.
	hexWrittenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))
)
