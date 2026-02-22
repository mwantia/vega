package compiler_test

import (
	"context"
	"strings"
	"testing"
	"time"

	_ "github.com/mwantia/vfs/mount/service/ephemeral"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/lexer"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/value"
	"github.com/mwantia/vega/pkg/vm"
)

type TestCompilerCase struct {
	Source string
	Error  *TestCompilerError
}

type TestCompilerError struct {
	Phase   string
	Message string
	Code    int
}

type TestCompilerCaseFactory func() *TestCompilerCase

var CaseFactories = map[string]TestCompilerCaseFactory{
	// Bare expression statements should produce parse errors
	"bare-literal-short": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			42s
			`,
			Error: &TestCompilerError{
				Phase:   "parse",
				Message: "bare expression statements are not allowed",
			},
		}
	},
	"bare-literal-integer": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			68
			`,
			Error: &TestCompilerError{
				Phase:   "parse",
				Message: "bare expression statements are not allowed",
			},
		}
	},
	"bare-identifier": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			x
			`,
			Error: &TestCompilerError{
				Phase:   "parse",
				Message: "bare expression statements are not allowed",
			},
		}
	},

	// Assignment tests
	"assign-short": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42s
			`,
			Error: nil,
		}
	},
	"assign-integer": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 68
			`,
			Error: nil,
		}
	},
	"assign-long": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 92l
			`,
			Error: nil,
		}
	},
	"assign-float": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 48.2f
			`,
			Error: nil,
		}
	},
	"assign-decimal": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 23.4
			`,
			Error: nil,
		}
	},
	"assign-byte": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42b
			`,
			Error: nil,
		}
	},
	"assign-char": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 'A'
			`,
			Error: nil,
		}
	},
	"assign-bool": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = true
			y = false
			`,
			Error: nil,
		}
	},

	// Load/store: assign then re-assign from variable
	"assign-load-store": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			y = x
			`,
			Error: nil,
		}
	},

	"free-and-reuse": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			free(x)
			y = 100l
			`,
			Error: nil,
		}
	},
	"use-after-free": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			free(x)
			y = x
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "undefined variable",
			},
		}
	},

	"typed-assign-int": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x: int = 42
			`,
			Error: nil,
		}
	},
	"typed-assign-union": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			y: int|bool = 15
			`,
			Error: nil,
		}
	},
	"typed-assign-union-reassign": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			y: int|bool = 15
			y = true
			`,
			Error: nil,
		}
	},
	"typed-assign-mismatch": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			y: int = true
			`,
			Error: &TestCompilerError{
				Phase:   "runtime",
				Message: "type mismatch",
			},
		}
	},
	"typed-assign-union-mismatch": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			y: int|bool = 15
			y = 3.14f
			`,
			Error: &TestCompilerError{
				Phase:   "runtime",
				Message: "type mismatch",
			},
		}
	},
	"typed-assign-unknown-type": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			y: foobar = 15
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "unknown type name",
			},
		}
	},

	// Pointer alias tests
	"pointer-alias-int": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			y = *int(0)
			`,
			Error: nil,
		}
	},
	"pointer-alias-short": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			y = *short(0)
			`,
			Error: nil,
		}
	},
	"pointer-alias-long": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42l
			y = *long(0)
			`,
			Error: nil,
		}
	},
	"pointer-alias-bool": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = true
			y = *bool(0)
			`,
			Error: nil,
		}
	},
	"pointer-alias-byte": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42b
			y = *byte(0)
			`,
			Error: nil,
		}
	},
	"pointer-alias-unknown-type": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			y = *foobar(0)
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "unknown type name",
			},
		}
	},
	"pointer-alias-out-of-bounds": func() *TestCompilerCase {
		return &TestCompilerCase{
			// 9999999 + 4 bytes > 1MiB global allocator capacity
			Source: `
			y = *int(9999999)
			`,
			Error: &TestCompilerError{
				Phase:   "runtime",
				Message: "pointer out of bounds",
			},
		}
	},
	"pointer-alias-free-error": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			y = *int(0)
			free(y)
			`,
			Error: &TestCompilerError{
				Phase:   "runtime",
				Message: "cannot free pointer alias",
			},
		}
	},
	"pointer-alias-overlapping": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			y = *short(0)
			z = *int(0)
			`,
			Error: nil,
		}
	},
	"pointer-alias-reads-same-bytes": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			x = 42
			y = *int(0)
			z = y
			`,
			Error: nil,
		}
	},

	// Struct tests
	"struct-define-and-use": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			struct point {
				x: int
				y: int
			}
			p = point { x = 10, y = 20 }
			`,
			Error: nil,
		}
	},
	"struct-field-load": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			struct vec2 {
				x: int
				y: int
			}
			v = vec2 { x = 3, y = 7 }
			a = v.x
			b = v.y
			`,
			Error: nil,
		}
	},
	"struct-multi-field-mixed-types": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			struct record {
				id: int
				active: bool
				score: float
			}
			r = record { id = 42, active = true, score = 3.14f }
			`,
			Error: nil,
		}
	},
	"struct-unknown-type": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			r = unknown_struct { x = 1 }
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "undefined struct type",
			},
		}
	},
	"struct-unknown-field": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			struct tiny {
				x: int
			}
			t = tiny { x = 1, y = 2 }
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "has no field",
			},
		}
	},
	"struct-field-unknown-access": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			struct small {
				a: int
			}
			s = small { a = 5 }
			x = s.b
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "has no field",
			},
		}
	},
	"struct-field-type-in-definition": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			struct bad {
				x: foobar
			}
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "unknown type",
			},
		}
	},

	// Tuple tests
	"tuple-create": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			t = (42, true)
			`,
			Error: nil,
		}
	},
	"tuple-mixed-types": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			t = (10s, 20, 3.14f, true)
			`,
			Error: nil,
		}
	},
	"tuple-field-access": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			t = (42, true)
			a = t.0
			b = t.1
			`,
			Error: nil,
		}
	},
	// Uses a stencil registered from Go via RegisterStencil — no struct
	// declaration in the script source.
	"registered-stencil-use": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			r = gorecord { id = 1, active = true }
			a = r.id
			b = r.active
			`,
			Error: nil,
		}
	},
	"tuple-field-out-of-bounds": func() *TestCompilerCase {
		return &TestCompilerCase{
			Source: `
			t = (42, true)
			a = t.5
			`,
			Error: &TestCompilerError{
				Phase:   "compile",
				Message: "has no field",
			},
		}
	},
}

func TestCompilerCases(t *testing.T) {
	p := parser.NewParser()

	vm, err := vm.NewEphemeralVM()
	if err != nil {
		t.Fatalf("Failed to create ephemeral vm: %v", err)
	}

	for name, factory := range CaseFactories {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()

			// Each subtest gets a fresh compiler so scope doesn't bleed between cases.
			c := compiler.NewCompiler()
			// Register a stencil from Go code — available to all tests without a `struct` declaration in the script source.
			if err := c.RegisterStencil("gorecord", compiler.Field("id", value.TagInteger), compiler.Field("active", value.TagBoolean)); err != nil {
				t.Fatalf("Failed to register stencil: %v", err)
			}

			test := factory()
			lexer, err := lexer.NewLexer(test.Source)
			if err != nil {
				t.Fatalf("Failed to create lexer from factory: %v", err)
			}

			buffer, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("Failed to create token buffer: %v", err)
			}

			ast, err := p.MakeProgram(buffer)
			if test.Error != nil && test.Error.Phase == "parse" {
				if err == nil {
					t.Fatalf("Expected parse error containing %q, got nil", test.Error.Message)
				}
				if !strings.Contains(err.Error(), test.Error.Message) {
					t.Fatalf("Parse error = %q, want containing %q", err.Error(), test.Error.Message)
				}
				return
			} else {
				if err != nil {
					t.Fatalf("Failed to make program: %v", err)
				}
			}

			byteCode, err := c.Compile(ast)
			if test.Error != nil && test.Error.Phase == "compile" {
				if err == nil {
					t.Fatalf("Expected compile error containing %q, got nil", test.Error.Message)
				}
				if !strings.Contains(err.Error(), test.Error.Message) {
					t.Fatalf("Compile error = %q, want containing %q", err.Error(), test.Error.Message)
				}
				return
			} else {
				if err != nil {
					t.Fatalf("Failed to compile program: %v", err)
				}
			}

			_, err = vm.Run(ctx, byteCode)
			if test.Error != nil && test.Error.Phase == "runtime" {
				if err == nil {
					t.Fatalf("Expected runtime error containing %q, got nil", test.Error.Message)
				}
				if !strings.Contains(err.Error(), test.Error.Message) {
					t.Fatalf("Runtime error = %q, want containing %q", err.Error(), test.Error.Message)
				}
				return
			} else {
				if err != nil {
					t.Fatalf("Failed to run bytecode in virtual machine: %v", err)
				}
			}
		})
	}
}
