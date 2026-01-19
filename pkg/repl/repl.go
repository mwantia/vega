// Package repl implements the Read-Eval-Print Loop for Vega.
package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/lexer"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/vm"
)

const (
	Prompt         = "vega> "
	ContinuePrompt = "   >> "
)

// REPL is the Read-Eval-Print Loop.
type REPL struct {
	vm      *vm.VirtualMachine
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	history []string
	running bool
	disasm  bool
}

// NewWithVM creates a REPL with an existing VM.
func NewWithVM(v *vm.VirtualMachine, stdin io.Reader, stdout, stderr io.Writer, disasm bool) *REPL {
	v.SetStdin(stdin)
	v.SetStdout(stdout)
	v.SetStderr(stderr)

	return &REPL{
		vm:      v,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		history: make([]string, 0),
		running: true,
		disasm:  disasm,
	}
}

// Run starts the REPL loop.
func (r *REPL) Run(ctx context.Context) error {
	r.printWelcome()

	scanner := bufio.NewScanner(r.stdin)
	var multilineBuffer strings.Builder
	inMultiline := false
	braceCount := 0

	for r.running {
		// Print prompt
		if inMultiline {
			fmt.Fprint(r.stdout, ContinuePrompt)
		} else {
			fmt.Fprint(r.stdout, Prompt)
		}

		// Read line
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		// Handle empty lines
		if strings.TrimSpace(line) == "" {
			if inMultiline {
				// Empty line in multiline mode - try to execute what we have
				input := multilineBuffer.String()
				multilineBuffer.Reset()
				inMultiline = false
				braceCount = 0
				r.execute(input)
			}
			continue
		}

		// Handle special commands
		if !inMultiline {
			if r.handleCommand(line) {
				continue
			}
		}

		// Track braces for multiline input
		for _, ch := range line {
			switch ch {
			case '{':
				braceCount++
			case '}':
				braceCount--
			}
		}

		// Accumulate input
		if inMultiline {
			multilineBuffer.WriteString("\n")
		}
		multilineBuffer.WriteString(line)

		// Check if we need more input
		if braceCount > 0 {
			inMultiline = true
			continue
		}

		// Execute complete input
		input := multilineBuffer.String()
		multilineBuffer.Reset()
		inMultiline = false
		braceCount = 0

		r.execute(input)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// handleCommand handles special REPL commands. Returns true if handled.
func (r *REPL) handleCommand(line string) bool {
	line = strings.TrimSpace(line)

	switch line {
	case "quit", "exit":
		r.running = false
		fmt.Fprintln(r.stdout, "Goodbye!")
		return true

	case "help":
		r.printHelp()
		return true

	case "history":
		r.printHistory()
		return true

	case "clear":
		// Clear screen (ANSI escape)
		fmt.Fprint(r.stdout, "\033[2J\033[H")
		return true

	case "vars":
		r.printVariables()
		return true
	}

	return false
}

// execute compiles and runs a line of code.
func (r *REPL) execute(input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	// Add to history
	r.history = append(r.history, input)

	// Lexer
	l := lexer.New(input)
	tokens, err := l.Tokenize()
	if err != nil {
		fmt.Fprintf(r.stderr, "Syntax error: %v\n", err)
		return
	}

	// Parser
	p := parser.New(tokens)
	program, err := p.Parse()
	if err != nil {
		fmt.Fprintf(r.stderr, "Parse error: %v\n", err)
		return
	}

	// Compiler
	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		fmt.Fprintf(r.stderr, "Compile error: %v\n", err)
		return
	}

	if r.disasm {
		fmt.Fprintln(r.stdout, bytecode.Disassemble())
		fmt.Fprintln(r.stdout, "--- Execution ---")
	}
	// Execute
	_, err = r.vm.Run(bytecode)
	if err != nil {
		fmt.Fprintf(r.stderr, "Runtime error: %v\n", err)
		return
	}

	// Print result if there's a value on the stack
	if result := r.vm.LastPopped(); result != nil {
		// Don't print nil results from statements
		if result.Type() != "nil" {
			fmt.Fprintln(r.stdout, result.String())
		}
	}
}

func (r *REPL) printWelcome() {
	fmt.Fprintln(r.stdout, "Vega - Virtual Execution & Graph Abstraction")
	fmt.Fprintln(r.stdout, "Type 'help' for available commands, 'quit' to exit.")
	fmt.Fprintln(r.stdout)
}

func (r *REPL) printHelp() {
	fmt.Fprintln(r.stdout, "Commands:")
	fmt.Fprintln(r.stdout, "  help     - Show this help message")
	fmt.Fprintln(r.stdout, "  quit     - Exit the REPL (also: exit)")
	fmt.Fprintln(r.stdout, "  history  - Show command history")
	fmt.Fprintln(r.stdout, "  clear    - Clear the screen")
	fmt.Fprintln(r.stdout, "  vars     - Show defined variables")
	fmt.Fprintln(r.stdout)
	fmt.Fprintln(r.stdout, "Multiline input:")
	fmt.Fprintln(r.stdout, "  Lines ending with '{' continue on the next line.")
	fmt.Fprintln(r.stdout, "  Press Enter on an empty line to execute.")
	fmt.Fprintln(r.stdout)
	fmt.Fprintln(r.stdout, "Examples:")
	fmt.Fprintln(r.stdout, "  x = 42")
	fmt.Fprintln(r.stdout, "  println(\"Hello, world!\")")
	fmt.Fprintln(r.stdout, "  for i in range(5) { println(i) }")
}

func (r *REPL) printHistory() {
	if len(r.history) == 0 {
		fmt.Fprintln(r.stdout, "No history.")
		return
	}
	for i, cmd := range r.history {
		fmt.Fprintf(r.stdout, "%4d: %s\n", i+1, cmd)
	}
}

func (r *REPL) printVariables() {
	// This would require exposing globals from VM
	// For now, just print a message
	fmt.Fprintln(r.stdout, "Variable inspection not yet implemented.")
}

// VM returns the underlying VM for external access.
func (r *REPL) VM() *vm.VirtualMachine {
	return r.vm
}
