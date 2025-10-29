package ast

// Node represents any node in the AST
type Node interface {
	TokenLiteral() string
}

// Program is the root node
type Program struct {
	Statements []Node
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// Decorator represents @decorator annotations
type Decorator struct {
	Name   string
	Args   []string
	KVArgs map[string]string // For key: value arguments
}

func (d *Decorator) TokenLiteral() string { return "@" + d.Name }

// StructDecl represents a struct definition
type StructDecl struct {
	Name       string
	Decorators []*Decorator
	Fields     []*FieldDecl
}

func (s *StructDecl) TokenLiteral() string { return "struct" }

// FieldDecl represents a field in a struct
type FieldDecl struct {
	Name       string
	Type       string
	IsArray    bool
	Decorators []*Decorator
}

func (f *FieldDecl) TokenLiteral() string { return f.Name }

// PageDecl represents a Page UI component
type PageDecl struct {
	Name       string
	Route      string
	Decorators []*Decorator
	Properties map[string]string
	Body       []Node
}

func (p *PageDecl) TokenLiteral() string { return "Page" }

// SectionDecl represents a Section UI component
type SectionDecl struct {
	Properties map[string]string
	Body       []Node
}

func (s *SectionDecl) TokenLiteral() string { return "Section" }

// RowDecl represents a Row UI component
type RowDecl struct {
	Properties map[string]string
	Body       []Node
}

func (r *RowDecl) TokenLiteral() string { return "Row" }

// ColumnDecl represents a Column UI component
type ColumnDecl struct {
	Properties map[string]string
	Body       []Node
}

func (c *ColumnDecl) TokenLiteral() string { return "Column" }

// DataListDecl represents a DataList UI component
type DataListDecl struct {
	Properties map[string]interface{}
}

func (d *DataListDecl) TokenLiteral() string { return "DataList" }

// FormDecl represents a Form UI component
type FormDecl struct {
	Properties map[string]string
	Body       []Node
}

func (f *FormDecl) TokenLiteral() string { return "Form" }

// InputDecl represents an Input UI component
type InputDecl struct {
	Properties map[string]string
}

func (i *InputDecl) TokenLiteral() string { return "Input" }

// ButtonDecl represents a Button UI component
type ButtonDecl struct {
	Properties map[string]string
}

func (b *ButtonDecl) TokenLiteral() string { return "Button" }

// HandlerDecl represents a handler function
type HandlerDecl struct {
	Path       string
	Method     string
	Name       string
	Parameters []*Parameter
	Body       []Node
	Decorators []*Decorator
}

func (h *HandlerDecl) TokenLiteral() string { return "handler" }

// Parameter represents a function parameter
type Parameter struct {
	Name string
	Type string
}

// FunctionDecl represents a function
type FunctionDecl struct {
	Name       string
	ReturnType string
	Parameters []*Parameter
	Body       []Node
}

func (f *FunctionDecl) TokenLiteral() string { return f.Name }

// MainDecl represents the main function
type MainDecl struct {
	Body []Node
}

func (m *MainDecl) TokenLiteral() string { return "main" }
