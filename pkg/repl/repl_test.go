package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/mwantia/vega/pkg/vm"
)

func newReplTest(stdin io.Reader, stdout, stderr io.Writer) *REPL {
	return NewWithVM(vm.NewVirtualMachine(), stdin, stdout, stderr, false)
}

func TestREPLExecute(t *testing.T) {
	input := "x = 42\nprintln(x)\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "42") {
		t.Errorf("expected output to contain '42', got: %s", outputStr)
	}
}

func TestREPLArithmetic(t *testing.T) {
	input := "println(1 + 2 * 3)\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "7") {
		t.Errorf("expected output to contain '7', got: %s", outputStr)
	}
}

func TestREPLMultiline(t *testing.T) {
	// Multiline for loop
	input := "for i in [1, 2, 3] {\nprintln(i)\n}\n\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "1") || !strings.Contains(outputStr, "2") || !strings.Contains(outputStr, "3") {
		t.Errorf("expected output to contain 1, 2, 3, got: %s", outputStr)
	}
}

func TestREPLHelp(t *testing.T) {
	input := "help\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Commands:") {
		t.Errorf("expected help output, got: %s", outputStr)
	}
}

func TestREPLHistory(t *testing.T) {
	input := "x = 1\ny = 2\nhistory\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "x = 1") || !strings.Contains(outputStr, "y = 2") {
		t.Errorf("expected history to show commands, got: %s", outputStr)
	}
}

func TestREPLSyntaxError(t *testing.T) {
	input := "x = \nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	// Should contain an error message but continue running
	if !strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "Error") {
		t.Errorf("expected error message, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "Goodbye") {
		t.Errorf("expected REPL to continue and quit gracefully, got: %s", outputStr)
	}
}

func TestREPLFunction(t *testing.T) {
	input := "fn double(x) {\nreturn x * 2\n}\n\nprintln(double(21))\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "42") {
		t.Errorf("expected output to contain '42', got: %s", outputStr)
	}
}

func TestREPLExit(t *testing.T) {
	input := "exit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "Goodbye") {
		t.Errorf("expected goodbye message, got: %s", outputStr)
	}
}

func TestREPLPersistentState(t *testing.T) {
	// Variables should persist across commands
	input := "x = 10\ny = x + 5\nprintln(y)\nquit\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	r := newReplTest(reader, &output, &output)
	err := r.Run(t.Context())
	if err != nil {
		t.Fatalf("REPL error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "15") {
		t.Errorf("expected output to contain '15', got: %s", outputStr)
	}
}
