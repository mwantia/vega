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

	// Styles for pretty-printing values
	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)

	stringValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	numberValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11"))

	unknownValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("14"))

	boolValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13"))

	nilValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	bracketStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

	typeAnnotationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8")).
				Italic(true)
)
