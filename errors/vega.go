// Package errors defines error types for the Vega language.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the category of a Vega error.
type ErrorType string

// Error types
const (
	SyntaxError  ErrorType = "SyntaxError"
	ParseError   ErrorType = "ParseError"
	CompileError ErrorType = "CompileError"
	TypeError    ErrorType = "TypeError"
	RuntimeError ErrorType = "RuntimeError"
)

// VegaError represents an error in Vega processing.
type VegaError struct {
	Type    ErrorType
	Message string
	Line    int
	Column  int
	Context string // Code snippet around error
}

// Error implements the error interface.
func (e *VegaError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s at line %d, column %d: %s", e.Type, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// WithContext returns the error with a code context snippet.
func (e *VegaError) WithContext(source string) *VegaError {
	if e.Line <= 0 {
		return e
	}

	lines := strings.Split(source, "\n")
	if e.Line > len(lines) {
		return e
	}

	// Get the line containing the error (1-indexed)
	contextLine := lines[e.Line-1]
	e.Context = contextLine

	return e
}

// Format returns a formatted error message with context.
func (e *VegaError) Format() string {
	var sb strings.Builder

	sb.WriteString(e.Error())

	if e.Context != "" {
		sb.WriteString("\n")
		sb.WriteString(e.Context)
		sb.WriteString("\n")

		// Add pointer to error location
		if e.Column > 0 {
			for i := 1; i < e.Column; i++ {
				sb.WriteString(" ")
			}
			sb.WriteString("^")
		}
	}

	return sb.String()
}

// New creates a new VegaError.
func New(errType ErrorType, message string, line, column int) *VegaError {
	return &VegaError{
		Type:    errType,
		Message: message,
		Line:    line,
		Column:  column,
	}
}

// NewSyntaxError creates a new syntax error.
func NewSyntaxError(message string, line, column int) *VegaError {
	return New(SyntaxError, message, line, column)
}

// NewParseError creates a new parse error.
func NewParseError(message string, line, column int) *VegaError {
	return New(ParseError, message, line, column)
}

// NewCompileError creates a new compile error.
func NewCompileError(message string, line, column int) *VegaError {
	return New(CompileError, message, line, column)
}

// NewTypeError creates a new type error.
func NewTypeError(message string, line, column int) *VegaError {
	return New(TypeError, message, line, column)
}

// NewRuntimeError creates a new runtime error.
func NewRuntimeError(message string, line, column int) *VegaError {
	return New(RuntimeError, message, line, column)
}

// ErrorList is a collection of VegaErrors.
type ErrorList []*VegaError

// Add adds an error to the list.
func (el *ErrorList) Add(err *VegaError) {
	*el = append(*el, err)
}

// HasErrors returns true if there are any errors.
func (el ErrorList) HasErrors() bool {
	return len(el) > 0
}

// Error implements the error interface for ErrorList.
func (el ErrorList) Error() string {
	switch len(el) {
	case 0:
		return "no errors"
	case 1:
		return el[0].Error()
	default:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("%d errors:\n", len(el)))
		for _, e := range el {
			sb.WriteString("  ")
			sb.WriteString(e.Error())
			sb.WriteString("\n")
		}
		return sb.String()
	}
}
