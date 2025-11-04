package codegen

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/gunesh/zelang/pkg/ast"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// TemplateGenerator generates C code using templates
type TemplateGenerator struct {
	templates *template.Template
	structs   []*ast.StructDecl
	pages     []*ast.PageDecl
	handlers  []*ast.HandlerDecl
	hasWeb    bool
}

// Template data structures
type FieldData struct {
	Name            string
	CType           string
	SQLType         string
	Constraints     string
	Title           string
	IsBool          bool
	IsArray         bool
	IsAutoIncrement bool
}

type ParamData struct {
	Type string
	Name string
}

type FormFieldData struct {
	Name      string
	Label     string
	InputType string
	Required  bool
}

type HTMLTemplateData struct {
	PageNameLower string
	PageTitle     string
	HasDataList   bool
	HasForm       bool
	StructName    string
	TableName     string
	Fields        []FieldData
	FormFields    []FormFieldData
}

type CRUDTemplateData struct {
	StructName   string
	TableName    string
	Params       []ParamData
	BindFields   []FieldData
	AllFields    []FieldData
	Fields       []FieldData
	FieldNames   string
	Placeholders string
}

// NewTemplateGenerator creates a new template-based generator
func NewTemplateGenerator() (*TemplateGenerator, error) {
	// Custom template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"lt":     func(a, b int) bool { return a < b },
		"len":    func(v interface{}) int { return len(v.([]FieldData)) },
		"title":  strings.Title,
		"printf": fmt.Sprintf,
	}

	// Parse all templates
	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &TemplateGenerator{
		templates: tmpl,
		structs:   []*ast.StructDecl{},
		pages:     []*ast.PageDecl{},
		handlers:  []*ast.HandlerDecl{},
		hasWeb:    false,
	}, nil
}

// Generate generates C code using templates
func (g *TemplateGenerator) Generate(program *ast.Program) (string, error) {
	var output bytes.Buffer

	// Collect statements
	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *ast.StructDecl:
			g.structs = append(g.structs, s)
		case *ast.PageDecl:
			g.pages = append(g.pages, s)
			g.hasWeb = true
		case *ast.HandlerDecl:
			g.handlers = append(g.handlers, s)
			g.hasWeb = true
		}
	}

	// Generate headers (still using direct code for now)
	g.generateHeaders(&output)

	// Generate struct definitions using templates
	for _, s := range g.structs {
		if err := g.generateStructWithTemplate(&output, s); err != nil {
			return "", err
		}
	}

	// Generate CRUD functions using templates
	for _, s := range g.structs {
		if err := g.generateCRUDWithTemplates(&output, s); err != nil {
			return "", err
		}
	}

	// Generate web server if needed
	if g.hasWeb {
		if err := g.generateWebServerWithTemplates(&output); err != nil {
			return "", err
		}
	} else {
		g.generateMainOld(&output)
	}

	return output.String(), nil
}

// generateHeaders generates C headers
func (g *TemplateGenerator) generateHeaders(output *bytes.Buffer) {
	output.WriteString(`#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sqlite3.h>
`)
	if g.hasWeb {
		output.WriteString(`#include <ctype.h>
#include <microhttpd.h>
`)
	}
	output.WriteString(`
// Global database connection
sqlite3 *db = NULL;

`)
	if g.hasWeb {
		output.WriteString(`// Global HTTP server
struct MHD_Daemon *http_daemon = NULL;

`)
	}
}

// generateCRUDWithTemplates generates CRUD functions using templates
func (g *TemplateGenerator) generateCRUDWithTemplates(output *bytes.Buffer, s *ast.StructDecl) error {
	tableName := g.getTableName(s)

	// Prepare data for templates
	data := g.prepareCRUDData(s, tableName)

	// Generate CREATE function
	if err := g.templates.ExecuteTemplate(output, "crud_create.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute crud_create template: %w", err)
	}
	output.WriteString("\n\n")

	// Generate FIND function
	if err := g.templates.ExecuteTemplate(output, "crud_find.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute crud_find template: %w", err)
	}
	output.WriteString("\n\n")

	// Generate ALL function
	if err := g.templates.ExecuteTemplate(output, "crud_all.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute crud_all template: %w", err)
	}
	output.WriteString("\n\n")

	// Generate DELETE function
	if err := g.templates.ExecuteTemplate(output, "crud_delete.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute crud_delete template: %w", err)
	}
	output.WriteString("\n\n")

	// Generate INIT_TABLE function
	if err := g.templates.ExecuteTemplate(output, "crud_init_table.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute crud_init_table template: %w", err)
	}
	output.WriteString("\n\n")

	return nil
}

// generateStructWithTemplate generates struct definition using template
func (g *TemplateGenerator) generateStructWithTemplate(output *bytes.Buffer, s *ast.StructDecl) error {
	type StructTemplateData struct {
		StructName string
		Fields     []FieldData
	}

	data := StructTemplateData{
		StructName: s.Name,
		Fields:     []FieldData{},
	}

	for _, field := range s.Fields {
		data.Fields = append(data.Fields, FieldData{
			Name:    field.Name,
			CType:   mapType(field.Type),
			IsArray: field.IsArray,
		})
	}

	if err := g.templates.ExecuteTemplate(output, "struct_def.tmpl", data); err != nil {
		return fmt.Errorf("failed to execute struct_def template: %w", err)
	}
	output.WriteString("\n")

	return nil
}

// generateWebServerWithTemplates generates web server using templates
func (g *TemplateGenerator) generateWebServerWithTemplates(output *bytes.Buffer) error {
	// Generate HTML header constants using template
	if err := g.templates.ExecuteTemplate(output, "html_header.tmpl", nil); err != nil {
		return fmt.Errorf("failed to execute html_header template: %w", err)
	}
	output.WriteString("\n")

	// Generate page rendering function
	if len(g.pages) > 0 && len(g.structs) > 0 {
		page := g.pages[0]
		s := g.structs[0]

		data := g.prepareHTMLData(page, s)

		if err := g.templates.ExecuteTemplate(output, "html_page.tmpl", data); err != nil {
			return fmt.Errorf("failed to execute html_page template: %w", err)
		}
		output.WriteString("\n\n")
	}

	// Generate HTTP route handler using template
	if len(g.pages) > 0 && len(g.structs) > 0 {
		page := g.pages[0]
		s := g.structs[0]
		data := g.prepareHTMLData(page, s)

		// Convert FormFields to FieldData for the handler template
		handlerData := struct {
			StructName    string
			TableName     string
			PageNameLower string
			FormFields    []FieldData
		}{
			StructName:    data.StructName,
			TableName:     data.TableName,
			PageNameLower: data.PageNameLower,
			FormFields:    []FieldData{},
		}

		// Get non-auto fields for form processing
		for _, field := range s.Fields {
			if field.IsArray {
				continue
			}
			isAuto := false
			for _, dec := range field.Decorators {
				if dec.Name == "autoincrement" || dec.Name == "primary" {
					isAuto = true
				}
			}
			if !isAuto {
				handlerData.FormFields = append(handlerData.FormFields, FieldData{
					Name:   field.Name,
					CType:  mapType(field.Type),
					IsBool: field.Type == "bool",
				})
			}
		}

		if err := g.templates.ExecuteTemplate(output, "http_handler.tmpl", handlerData); err != nil {
			return fmt.Errorf("failed to execute http_handler template: %w", err)
		}
		output.WriteString("\n\n")
	}

	// Generate web main using template
	mainData := struct {
		Structs []struct {
			Name string
		}
	}{
		Structs: []struct {
			Name string
		}{},
	}

	for _, s := range g.structs {
		mainData.Structs = append(mainData.Structs, struct {
			Name string
		}{Name: s.Name})
	}

	if err := g.templates.ExecuteTemplate(output, "web_main.tmpl", mainData); err != nil {
		return fmt.Errorf("failed to execute web_main template: %w", err)
	}

	return nil
}

// prepareCRUDData prepares data for CRUD templates
func (g *TemplateGenerator) prepareCRUDData(s *ast.StructDecl, tableName string) CRUDTemplateData {
	data := CRUDTemplateData{
		StructName: s.Name,
		TableName:  tableName,
		Params:     []ParamData{},
		BindFields: []FieldData{},
		AllFields:  []FieldData{},
		Fields:     []FieldData{},
	}

	fieldNames := []string{}
	placeholders := []string{}

	// Process fields
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}

		isAuto := false
		isPrimary := false
		for _, dec := range field.Decorators {
			if dec.Name == "autoincrement" || dec.Name == "timestamp" {
				isAuto = true
			}
			if dec.Name == "primary" {
				isPrimary = true
			}
		}

		cType := mapType(field.Type)
		fieldData := FieldData{
			Name:            field.Name,
			CType:           cType,
			SQLType:         mapSQLType(field.Type),
			Constraints:     getFieldConstraints(field),
			IsAutoIncrement: isAuto && isPrimary,
			IsBool:          field.Type == "bool",
		}

		data.AllFields = append(data.AllFields, fieldData)
		data.Fields = append(data.Fields, fieldData)

		if !isAuto {
			data.Params = append(data.Params, ParamData{
				Type: cType,
				Name: field.Name,
			})
			data.BindFields = append(data.BindFields, fieldData)
			fieldNames = append(fieldNames, field.Name)
			placeholders = append(placeholders, "?")
		}
	}

	data.FieldNames = strings.Join(fieldNames, ", ")
	data.Placeholders = strings.Join(placeholders, ", ")

	return data
}

// prepareHTMLData prepares data for HTML templates
func (g *TemplateGenerator) prepareHTMLData(page *ast.PageDecl, s *ast.StructDecl) HTMLTemplateData {
	tableName := g.getTableName(s)

	data := HTMLTemplateData{
		PageNameLower: strings.ToLower(page.Name),
		PageTitle:     page.Name,
		HasDataList:   true,
		HasForm:       true,
		StructName:    s.Name,
		TableName:     tableName,
		Fields:        []FieldData{},
		FormFields:    []FormFieldData{},
	}

	// Prepare field data
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}

		cType := mapType(field.Type)
		data.Fields = append(data.Fields, FieldData{
			Name:        field.Name,
			CType:       cType,
			SQLType:     mapSQLType(field.Type),
			Constraints: getFieldConstraints(field),
			Title:       strings.Title(field.Name),
			IsBool:      field.Type == "bool",
		})

		// Skip auto fields in forms
		isAuto := false
		for _, dec := range field.Decorators {
			if dec.Name == "autoincrement" || dec.Name == "primary" {
				isAuto = true
			}
		}

		if !isAuto {
			inputType := "text"
			if field.Name == "description" {
				inputType = "textarea"
			} else if field.Type == "bool" {
				inputType = "checkbox"
			} else if field.Type == "int" {
				inputType = "number"
			}

			required := false
			for _, dec := range field.Decorators {
				if dec.Name == "required" {
					required = true
				}
			}

			data.FormFields = append(data.FormFields, FormFieldData{
				Name:      field.Name,
				Label:     strings.Title(field.Name),
				InputType: inputType,
				Required:  required,
			})
		}
	}

	return data
}

// Helper function
func (g *TemplateGenerator) getTableName(s *ast.StructDecl) string {
	for _, dec := range s.Decorators {
		if dec.Name == "table" && len(dec.Args) > 0 {
			return strings.Trim(dec.Args[0], `"`)
		}
	}
	return strings.ToLower(s.Name) + "s"
}

func mapType(zlType string) string {
	switch zlType {
	case "int":
		return "int64_t"
	case "float":
		return "double"
	case "string":
		return "char*"
	case "bool":
		return "int"
	case "date", "datetime":
		return "char*"
	default:
		return zlType
	}
}

func mapSQLType(zlType string) string {
	switch zlType {
	case "int":
		return "INTEGER"
	case "float":
		return "REAL"
	case "string":
		return "TEXT"
	case "bool":
		return "INTEGER"
	case "date", "datetime":
		return "TEXT"
	default:
		return "TEXT"
	}
}

func getFieldConstraints(field *ast.FieldDecl) string {
	constraints := ""

	for _, dec := range field.Decorators {
		switch dec.Name {
		case "primary":
			constraints += " PRIMARY KEY"
		case "autoincrement":
			constraints += " AUTOINCREMENT"
		case "required":
			constraints += " NOT NULL"
		case "unique":
			constraints += " UNIQUE"
		}
	}

	return constraints
}

// Old generation methods (temporary - to be replaced with templates)
// generateMainOld generates CLI main with demo code (temporary - for CLI apps)
func (g *TemplateGenerator) generateMainOld(output *bytes.Buffer) {
	output.WriteString("// TODO: Generate CLI main using template\n")
	output.WriteString("// For now, CLI apps use old c_generator.go\n")
}
