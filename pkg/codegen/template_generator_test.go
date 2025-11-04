package codegen

import (
	"strings"
	"testing"

	"github.com/gunesh/zelang/pkg/ast"
)

func TestTemplateGenerator(t *testing.T) {
	// Create a simple test program
	program := &ast.Program{
		Statements: []ast.Node{
			&ast.StructDecl{
				Name: "Todo",
				Decorators: []*ast.Decorator{
					{Name: "storage", Args: []string{"sqlite"}},
					{Name: "table", Args: []string{`"todos"`}},
				},
				Fields: []*ast.FieldDecl{
					{
						Name: "id",
						Type: "int",
						Decorators: []*ast.Decorator{
							{Name: "primary"},
							{Name: "autoincrement"},
						},
					},
					{
						Name: "title",
						Type: "string",
						Decorators: []*ast.Decorator{
							{Name: "required"},
						},
					},
					{
						Name: "completed",
						Type: "bool",
					},
				},
			},
			&ast.PageDecl{
				Name: "TodoApp",
				Decorators: []*ast.Decorator{
					{Name: "route", Args: []string{`"/"`}},
				},
			},
		},
	}

	// Create template generator
	gen, err := NewTemplateGenerator()
	if err != nil {
		t.Fatalf("Failed to create template generator: %v", err)
	}

	// Generate code
	code, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Failed to generate code: %v", err)
	}

	// Check that code was generated
	if len(code) == 0 {
		t.Error("Generated code is empty")
	}

	// Check for expected patterns in generated code
	expectedPatterns := []string{
		"typedef struct Todo",
		"Todo* Todo_create(",
		"Todo* Todo_find(",
		"int64_t id",
		"char* title",
		"int completed",
	}

	for _, pattern := range expectedPatterns {
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code missing expected pattern: %s", pattern)
		}
	}

	t.Logf("Generated code (first 500 chars):\n%s", code[:min(500, len(code))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
