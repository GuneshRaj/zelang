package codegen

import (
	"fmt"
	"strings"

	"github.com/gunesh/zelang/pkg/ast"
)

type CGenerator struct {
	structs  []*ast.StructDecl
	pages    []*ast.PageDecl
	handlers []*ast.HandlerDecl
	output   strings.Builder
	hasWeb   bool
}

func New() *CGenerator {
	return &CGenerator{
		structs:  []*ast.StructDecl{},
		pages:    []*ast.PageDecl{},
		handlers: []*ast.HandlerDecl{},
		hasWeb:   false,
	}
}

func (g *CGenerator) Generate(program *ast.Program) string {
	g.output.Reset()

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

	// Generate headers
	g.generateHeaders()

	// Generate struct definitions
	for _, s := range g.structs {
		g.generateStruct(s)
	}

	// Generate CRUD functions
	for _, s := range g.structs {
		g.generateCRUD(s)
	}

	// Generate HTTP server if web features are used
	if g.hasWeb {
		g.generateWebServer()
	} else {
		// Generate main function for non-web apps
		g.generateMain()
	}

	return g.output.String()
}

func (g *CGenerator) generateHeaders() {
	g.output.WriteString(`#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sqlite3.h>
`)
	if g.hasWeb {
		g.output.WriteString(`#include <ctype.h>
#include <microhttpd.h>
`)
	}
	g.output.WriteString(`
// Global database connection
sqlite3 *db = NULL;

`)
	if g.hasWeb {
		g.output.WriteString(`// Global HTTP server
struct MHD_Daemon *http_daemon = NULL;

`)
	}
}

func (g *CGenerator) generateStruct(s *ast.StructDecl) {
	g.output.WriteString(fmt.Sprintf("// Struct: %s\n", s.Name))
	g.output.WriteString(fmt.Sprintf("typedef struct %s {\n", s.Name))

	for _, field := range s.Fields {
		cType := g.mapType(field.Type)
		if field.IsArray {
			g.output.WriteString(fmt.Sprintf("    %s* %s;\n", cType, field.Name))
			g.output.WriteString(fmt.Sprintf("    int %s_count;\n", field.Name))
		} else {
			g.output.WriteString(fmt.Sprintf("    %s %s;\n", cType, field.Name))
		}
	}

	g.output.WriteString(fmt.Sprintf("} %s;\n\n", s.Name))
}

func (g *CGenerator) mapType(zlType string) string {
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

func (g *CGenerator) generateCRUD(s *ast.StructDecl) {
	tableName := g.getTableName(s)

	// Generate CREATE TABLE function
	g.generateInitTable(s, tableName)

	// Generate CREATE function
	g.generateCreate(s, tableName)

	// Generate FIND function
	g.generateFind(s, tableName)

	// Generate ALL function
	g.generateAll(s, tableName)

	// Generate DELETE function
	g.generateDelete(s, tableName)
}

func (g *CGenerator) getTableName(s *ast.StructDecl) string {
	// Check for @table decorator
	for _, dec := range s.Decorators {
		if dec.Name == "table" && len(dec.Args) > 0 {
			return strings.Trim(dec.Args[0], `"`)
		}
	}
	// Default: lowercase struct name + s
	return strings.ToLower(s.Name) + "s"
}

func (g *CGenerator) generateInitTable(s *ast.StructDecl, tableName string) {
	g.output.WriteString(fmt.Sprintf("void %s_init_table() {\n", s.Name))
	g.output.WriteString("    char *sql = \"CREATE TABLE IF NOT EXISTS " + tableName + " (\"\n")

	fields := []string{}
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}

		sqlType := g.mapSQLType(field.Type)
		constraints := g.getFieldConstraints(field)
		fields = append(fields, fmt.Sprintf("        \"%s %s%s\"", field.Name, sqlType, constraints))
	}

	for i, field := range fields {
		if i < len(fields)-1 {
			g.output.WriteString(field + "\n")
			g.output.WriteString("        \",\"\n")
		} else {
			g.output.WriteString(field + "\n")
		}
	}

	g.output.WriteString("        \")\";\n")
	g.output.WriteString("    \n")
	g.output.WriteString("    char *err_msg = NULL;\n")
	g.output.WriteString("    int rc = sqlite3_exec(db, sql, NULL, NULL, &err_msg);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"SQL error: %s\\n\", err_msg);\n")
	g.output.WriteString("        sqlite3_free(err_msg);\n")
	g.output.WriteString("    } else {\n")
	g.output.WriteString(fmt.Sprintf("        printf(\"Table %s created successfully\\n\");\n", tableName))
	g.output.WriteString("    }\n")
	g.output.WriteString("}\n\n")
}

func (g *CGenerator) mapSQLType(zlType string) string {
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

func (g *CGenerator) getFieldConstraints(field *ast.FieldDecl) string {
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

func (g *CGenerator) generateCreate(s *ast.StructDecl, tableName string) {
	g.output.WriteString(fmt.Sprintf("%s* %s_create(", s.Name, s.Name))

	// Parameters
	params := []string{}
	nonAutoFields := []*ast.FieldDecl{}
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}
		// Skip auto fields
		isAuto := false
		for _, dec := range field.Decorators {
			if dec.Name == "autoincrement" || dec.Name == "timestamp" {
				isAuto = true
			}
		}
		if !isAuto {
			cType := g.mapType(field.Type)
			params = append(params, fmt.Sprintf("%s %s", cType, field.Name))
			nonAutoFields = append(nonAutoFields, field)
		}
	}
	g.output.WriteString(strings.Join(params, ", "))
	g.output.WriteString(") {\n")

	// Generate INSERT SQL
	g.output.WriteString("    char sql[1024];\n")
	g.output.WriteString(fmt.Sprintf("    sprintf(sql, \"INSERT INTO %s (", tableName))

	fieldNames := []string{}
	for _, field := range nonAutoFields {
		fieldNames = append(fieldNames, field.Name)
	}
	g.output.WriteString(strings.Join(fieldNames, ", "))
	g.output.WriteString(") VALUES (")

	placeholders := []string{}
	for range nonAutoFields {
		placeholders = append(placeholders, "?")
	}
	g.output.WriteString(strings.Join(placeholders, ", "))
	g.output.WriteString(")\");\n\n")

	// Prepare statement
	g.output.WriteString("    sqlite3_stmt *stmt;\n")
	g.output.WriteString("    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to prepare statement: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        return NULL;\n")
	g.output.WriteString("    }\n\n")

	// Bind parameters
	for i, field := range nonAutoFields {
		bindIndex := i + 1
		cType := g.mapType(field.Type)

		switch cType {
		case "int64_t":
			g.output.WriteString(fmt.Sprintf("    sqlite3_bind_int64(stmt, %d, %s);\n", bindIndex, field.Name))
		case "double":
			g.output.WriteString(fmt.Sprintf("    sqlite3_bind_double(stmt, %d, %s);\n", bindIndex, field.Name))
		case "char*":
			g.output.WriteString(fmt.Sprintf("    sqlite3_bind_text(stmt, %d, %s, -1, SQLITE_TRANSIENT);\n", bindIndex, field.Name))
		}
	}

	// Execute
	g.output.WriteString("\n    rc = sqlite3_step(stmt);\n")
	g.output.WriteString("    if (rc != SQLITE_DONE) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to insert: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        sqlite3_finalize(stmt);\n")
	g.output.WriteString("        return NULL;\n")
	g.output.WriteString("    }\n\n")

	// Get last insert ID
	g.output.WriteString("    int64_t last_insert_id = sqlite3_last_insert_rowid(db);\n")
	g.output.WriteString("    sqlite3_finalize(stmt);\n\n")

	// Create and populate struct
	g.output.WriteString(fmt.Sprintf("    %s* obj = (%s*)malloc(sizeof(%s));\n", s.Name, s.Name, s.Name))

	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}
		isPrimary := false
		isAuto := false
		for _, dec := range field.Decorators {
			if dec.Name == "primary" {
				isPrimary = true
			}
			if dec.Name == "autoincrement" {
				isAuto = true
			}
		}

		if (field.Name == "id" || isPrimary) && isAuto {
			g.output.WriteString(fmt.Sprintf("    obj->%s = last_insert_id;\n", field.Name))
		} else {
			cType := g.mapType(field.Type)
			if cType == "char*" {
				g.output.WriteString(fmt.Sprintf("    obj->%s = strdup(%s);\n", field.Name, field.Name))
			} else {
				g.output.WriteString(fmt.Sprintf("    obj->%s = %s;\n", field.Name, field.Name))
			}
		}
	}

	g.output.WriteString("\n    return obj;\n")
	g.output.WriteString("}\n\n")
}

func (g *CGenerator) generateFind(s *ast.StructDecl, tableName string) {
	g.output.WriteString(fmt.Sprintf("%s* %s_find(int64_t id) {\n", s.Name, s.Name))

	// Build SELECT query
	g.output.WriteString(fmt.Sprintf("    char *sql = \"SELECT * FROM %s WHERE id = ?\";\n", tableName))
	g.output.WriteString("    sqlite3_stmt *stmt;\n\n")

	g.output.WriteString("    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to prepare statement: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        return NULL;\n")
	g.output.WriteString("    }\n\n")

	g.output.WriteString("    sqlite3_bind_int64(stmt, 1, id);\n\n")

	g.output.WriteString("    rc = sqlite3_step(stmt);\n")
	g.output.WriteString("    if (rc != SQLITE_ROW) {\n")
	g.output.WriteString("        sqlite3_finalize(stmt);\n")
	g.output.WriteString("        return NULL;\n")
	g.output.WriteString("    }\n\n")

	// Create object
	g.output.WriteString(fmt.Sprintf("    %s* obj = (%s*)malloc(sizeof(%s));\n", s.Name, s.Name, s.Name))

	// Read columns
	colIndex := 0
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}

		cType := g.mapType(field.Type)
		switch cType {
		case "int64_t":
			g.output.WriteString(fmt.Sprintf("    obj->%s = sqlite3_column_int64(stmt, %d);\n", field.Name, colIndex))
		case "double":
			g.output.WriteString(fmt.Sprintf("    obj->%s = sqlite3_column_double(stmt, %d);\n", field.Name, colIndex))
		case "char*":
			g.output.WriteString(fmt.Sprintf("    obj->%s = strdup((const char*)sqlite3_column_text(stmt, %d));\n", field.Name, colIndex))
		}
		colIndex++
	}

	g.output.WriteString("\n    sqlite3_finalize(stmt);\n")
	g.output.WriteString("    return obj;\n")
	g.output.WriteString("}\n\n")
}

func (g *CGenerator) generateAll(s *ast.StructDecl, tableName string) {
	g.output.WriteString(fmt.Sprintf("%s** %s_all(int* count) {\n", s.Name, s.Name))

	g.output.WriteString(fmt.Sprintf("    char *sql = \"SELECT * FROM %s\";\n", tableName))
	g.output.WriteString("    sqlite3_stmt *stmt;\n\n")

	g.output.WriteString("    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to prepare statement: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        *count = 0;\n")
	g.output.WriteString("        return NULL;\n")
	g.output.WriteString("    }\n\n")

	// Allocate array
	g.output.WriteString("    int capacity = 10;\n")
	g.output.WriteString(fmt.Sprintf("    %s** results = (%s**)malloc(capacity * sizeof(%s*));\n", s.Name, s.Name, s.Name))
	g.output.WriteString("    int n = 0;\n\n")

	// Fetch all rows
	g.output.WriteString("    while ((rc = sqlite3_step(stmt)) == SQLITE_ROW) {\n")
	g.output.WriteString("        if (n >= capacity) {\n")
	g.output.WriteString("            capacity *= 2;\n")
	g.output.WriteString(fmt.Sprintf("            results = (%s**)realloc(results, capacity * sizeof(%s*));\n", s.Name, s.Name))
	g.output.WriteString("        }\n\n")

	// Create object
	g.output.WriteString(fmt.Sprintf("        %s* obj = (%s*)malloc(sizeof(%s));\n", s.Name, s.Name, s.Name))

	// Read columns
	colIndex := 0
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}

		cType := g.mapType(field.Type)
		switch cType {
		case "int64_t":
			g.output.WriteString(fmt.Sprintf("        obj->%s = sqlite3_column_int64(stmt, %d);\n", field.Name, colIndex))
		case "double":
			g.output.WriteString(fmt.Sprintf("        obj->%s = sqlite3_column_double(stmt, %d);\n", field.Name, colIndex))
		case "char*":
			g.output.WriteString(fmt.Sprintf("        obj->%s = strdup((const char*)sqlite3_column_text(stmt, %d));\n", field.Name, colIndex))
		}
		colIndex++
	}

	g.output.WriteString("\n        results[n++] = obj;\n")
	g.output.WriteString("    }\n\n")

	g.output.WriteString("    sqlite3_finalize(stmt);\n")
	g.output.WriteString("    *count = n;\n")
	g.output.WriteString("    return results;\n")
	g.output.WriteString("}\n\n")
}

func (g *CGenerator) generateDelete(s *ast.StructDecl, tableName string) {
	g.output.WriteString(fmt.Sprintf("int %s_delete(int64_t id) {\n", s.Name))

	g.output.WriteString(fmt.Sprintf("    char *sql = \"DELETE FROM %s WHERE id = ?\";\n", tableName))
	g.output.WriteString("    sqlite3_stmt *stmt;\n\n")

	g.output.WriteString("    int rc = sqlite3_prepare_v2(db, sql, -1, &stmt, NULL);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to prepare statement: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        return 0;\n")
	g.output.WriteString("    }\n\n")

	g.output.WriteString("    sqlite3_bind_int64(stmt, 1, id);\n\n")

	g.output.WriteString("    rc = sqlite3_step(stmt);\n")
	g.output.WriteString("    sqlite3_finalize(stmt);\n\n")

	g.output.WriteString("    if (rc != SQLITE_DONE) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to delete: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        return 0;\n")
	g.output.WriteString("    }\n\n")

	g.output.WriteString("    return 1;\n")
	g.output.WriteString("}\n\n")
}

func (g *CGenerator) generateMain() {
	g.output.WriteString("int main(int argc, char *argv[]) {\n")
	g.output.WriteString("    // Initialize database\n")
	g.output.WriteString("    int rc = sqlite3_open(\"app.db\", &db);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"Cannot open database: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        return 1;\n")
	g.output.WriteString("    }\n")
	g.output.WriteString("    printf(\"Database opened successfully\\n\\n\");\n\n")

	// Initialize tables
	for _, s := range g.structs {
		g.output.WriteString(fmt.Sprintf("    %s_init_table();\n", s.Name))
	}

	g.output.WriteString("\n    // ===== CRUD DEMO =====\n")
	g.output.WriteString("    printf(\"\\n===== CRUD Operations Demo =====\\n\\n\");\n\n")

	// Demo for first struct only
	if len(g.structs) > 0 {
		s := g.structs[0]
		tableName := strings.ToLower(s.Name)

		// CREATE demo
		g.output.WriteString("    // CREATE: Insert records\n")
		g.output.WriteString("    printf(\"Creating records...\\n\");\n")

		// Generate 3 sample inserts based on struct fields
		nonAutoFields := []*ast.FieldDecl{}
		for _, field := range s.Fields {
			isAuto := false
			for _, dec := range field.Decorators {
				if dec.Name == "autoincrement" || dec.Name == "timestamp" {
					isAuto = true
				}
			}
			if !isAuto && !field.IsArray {
				nonAutoFields = append(nonAutoFields, field)
			}
		}

		// Sample data for 3 records
		samples := [][]string{
			{"John Doe", "Class A"},
			{"Jane Smith", "Class B"},
			{"Bob Johnson", "Class A"},
		}

		for i, sample := range samples {
			g.output.WriteString(fmt.Sprintf("    %s* %s%d = %s_create(", s.Name, tableName, i+1, s.Name))

			args := []string{}
			for j, field := range nonAutoFields {
				cType := g.mapType(field.Type)
				if cType == "char*" {
					if j < len(sample) {
						args = append(args, fmt.Sprintf("\"%s\"", sample[j]))
					} else {
						args = append(args, "\"Sample\"")
					}
				} else {
					args = append(args, fmt.Sprintf("%d", (i+1)*10))
				}
			}

			g.output.WriteString(strings.Join(args, ", "))
			g.output.WriteString(");\n")

			g.output.WriteString(fmt.Sprintf("    if (%s%d) printf(\"  Created %s with ID: %%lld\\n\", %s%d->id);\n\n",
				tableName, i+1, s.Name, tableName, i+1))
		}

		// READ demo - find by ID
		g.output.WriteString("    // READ: Find by ID\n")
		g.output.WriteString("    printf(\"\\nFinding record by ID...\\n\");\n")
		g.output.WriteString(fmt.Sprintf("    %s* found = %s_find(1);\n", s.Name, s.Name))
		g.output.WriteString("    if (found) {\n")
		g.output.WriteString(fmt.Sprintf("        printf(\"  Found %s ID %%lld: \", found->id);\n", s.Name))

		// Print all fields
		for _, field := range s.Fields {
			if field.IsArray {
				continue
			}
			cType := g.mapType(field.Type)
			if cType == "char*" {
				g.output.WriteString(fmt.Sprintf("        printf(\"%s=%%s \", found->%s);\n", field.Name, field.Name))
			} else if cType == "int64_t" {
				g.output.WriteString(fmt.Sprintf("        printf(\"%s=%%lld \", found->%s);\n", field.Name, field.Name))
			} else if cType == "double" {
				g.output.WriteString(fmt.Sprintf("        printf(\"%s=%%f \", found->%s);\n", field.Name, field.Name))
			}
		}

		g.output.WriteString("        printf(\"\\n\");\n")
		g.output.WriteString("    }\n\n")

		// READ demo - all records
		g.output.WriteString("    // READ: Get all records\n")
		g.output.WriteString("    printf(\"\\nGetting all records...\\n\");\n")
		g.output.WriteString("    int count = 0;\n")
		g.output.WriteString(fmt.Sprintf("    %s** all = %s_all(&count);\n", s.Name, s.Name))
		g.output.WriteString("    printf(\"  Found %d records:\\n\", count);\n")
		g.output.WriteString("    for (int i = 0; i < count; i++) {\n")
		g.output.WriteString("        printf(\"    [%d] ID=%lld\", i+1, all[i]->id);\n")

		// Print first string field
		for _, field := range s.Fields {
			if field.IsArray {
				continue
			}
			cType := g.mapType(field.Type)
			if cType == "char*" {
				g.output.WriteString(fmt.Sprintf("        printf(\" %s=%%s\", all[i]->%s);\n", field.Name, field.Name))
				break
			}
		}

		g.output.WriteString("        printf(\"\\n\");\n")
		g.output.WriteString("    }\n\n")

		// DELETE demo
		g.output.WriteString("    // DELETE: Remove a record\n")
		g.output.WriteString("    printf(\"\\nDeleting record with ID=2...\\n\");\n")
		g.output.WriteString(fmt.Sprintf("    int deleted = %s_delete(2);\n", s.Name))
		g.output.WriteString("    if (deleted) printf(\"  Record deleted successfully\\n\");\n\n")

		// Verify deletion
		g.output.WriteString("    // Verify deletion\n")
		g.output.WriteString("    printf(\"\\nVerifying deletion...\\n\");\n")
		g.output.WriteString(fmt.Sprintf("    all = %s_all(&count);\n", s.Name))
		g.output.WriteString("    printf(\"  Remaining records: %d\\n\", count);\n\n")
	}

	g.output.WriteString("    printf(\"\\n===== Demo Complete =====\\n\");\n\n")

	g.output.WriteString("    // Close database\n")
	g.output.WriteString("    sqlite3_close(db);\n")
	g.output.WriteString("    return 0;\n")
	g.output.WriteString("}\n")
}

func (g *CGenerator) generateWebServer() {
	// Generate HTML rendering functions
	g.generateHTMLFunctions()

	// Generate route handler
	g.generateRouteHandler()

	// Generate main function with HTTP server
	g.generateWebMain()
}

func (g *CGenerator) generateHTMLFunctions() {
	// Generate Bootstrap HTML header
	g.output.WriteString(`
// HTML generation functions
const char* html_header =
    "<!DOCTYPE html>\n"
    "<html lang='en'>\n"
    "<head>\n"
    "    <meta charset='UTF-8'>\n"
    "    <meta name='viewport' content='width=device-width, initial-scale=1.0'>\n"
    "    <title>%s</title>\n"
    "    <link href='https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css' rel='stylesheet'>\n"
    "</head>\n"
    "<body>\n"
    "    <div class='container mt-5'>\n";

const char* html_footer =
    "    </div>\n"
    "    <script src='https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js'></script>\n"
    "</body>\n"
    "</html>\n";

`)

	// Generate page rendering function for the first page
	if len(g.pages) > 0 {
		page := g.pages[0]
		g.output.WriteString(fmt.Sprintf("char* render_%s_page() {\n", strings.ToLower(page.Name)))
		g.output.WriteString("    char* html = (char*)malloc(65536);\n")
		g.output.WriteString("    int offset = 0;\n\n")

		// Get page title
		title := page.Name
		for _, dec := range page.Decorators {
			if dec.Name == "route" && len(dec.Args) > 0 {
				// Use route as indicator
			}
		}

		// Write header
		g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, html_header, \"%s\");\n", title))

		// Write page title
		g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<h1 class='mb-4'>%s</h1>\\n\");\n\n", title))

		// Generate DataList (table) if struct exists
		if len(g.structs) > 0 {
			s := g.structs[0]
			g.generateDataListHTML(s)
		}

		// Generate Form
		if len(g.structs) > 0 {
			s := g.structs[0]
			g.generateFormHTML(s)
		}

		// Write footer
		g.output.WriteString("    offset += sprintf(html + offset, \"%s\", html_footer);\n")
		g.output.WriteString("    return html;\n")
		g.output.WriteString("}\n\n")
	}
}

func (g *CGenerator) generateDataListHTML(s *ast.StructDecl) {
	tableName := g.getTableName(s)

	g.output.WriteString("    // DataList - Show all records\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"<h2>All Items</h2>\\n\");\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"<table class='table table-striped'>\\n\");\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"<thead><tr>\");\n")

	// Table headers
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}
		g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<th>%s</th>\");\n",
			strings.Title(field.Name)))
	}
	g.output.WriteString("    offset += sprintf(html + offset, \"<th>Actions</th>\");\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"</tr></thead>\\n\");\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"<tbody>\\n\");\n\n")

	// Get all records
	g.output.WriteString("    int count = 0;\n")
	g.output.WriteString(fmt.Sprintf("    %s** items = %s_all(&count);\n", s.Name, s.Name))
	g.output.WriteString("    for (int i = 0; i < count; i++) {\n")
	g.output.WriteString("        offset += sprintf(html + offset, \"<tr>\");\n")

	// Table data
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}
		cType := g.mapType(field.Type)
		if cType == "char*" {
			g.output.WriteString(fmt.Sprintf("        offset += sprintf(html + offset, \"<td>%%s</td>\", items[i]->%s);\n", field.Name))
		} else if cType == "int" {
			// bool type
			if field.Type == "bool" {
				g.output.WriteString(fmt.Sprintf("        offset += sprintf(html + offset, \"<td>%%s</td>\", items[i]->%s ? \"Yes\" : \"No\");\n", field.Name))
			} else {
				g.output.WriteString(fmt.Sprintf("        offset += sprintf(html + offset, \"<td>%%d</td>\", items[i]->%s);\n", field.Name))
			}
		} else if cType == "int64_t" {
			g.output.WriteString(fmt.Sprintf("        offset += sprintf(html + offset, \"<td>%%lld</td>\", items[i]->%s);\n", field.Name))
		} else if cType == "double" {
			g.output.WriteString(fmt.Sprintf("        offset += sprintf(html + offset, \"<td>%%f</td>\", items[i]->%s);\n", field.Name))
		}
	}

	// Delete action
	g.output.WriteString(fmt.Sprintf("        offset += sprintf(html + offset, \"<td><a href='/%s/delete?id=%%lld' class='btn btn-sm btn-danger'>Delete</a></td>\", items[i]->id);\n", tableName))
	g.output.WriteString("        offset += sprintf(html + offset, \"</tr>\\n\");\n")
	g.output.WriteString("    }\n\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"</tbody></table>\\n\");\n\n")
}

func (g *CGenerator) generateFormHTML(s *ast.StructDecl) {
	tableName := g.getTableName(s)

	g.output.WriteString("    // Form - Add new record\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"<h2 class='mt-5'>Add New Item</h2>\\n\");\n")
	g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<form method='POST' action='/%s/create'>\\n\");\n", tableName))

	// Generate form fields
	for _, field := range s.Fields {
		if field.IsArray {
			continue
		}

		// Skip auto fields
		isAuto := false
		for _, dec := range field.Decorators {
			if dec.Name == "autoincrement" || dec.Name == "primary" {
				isAuto = true
			}
		}
		if isAuto {
			continue
		}

		fieldLabel := strings.Title(field.Name)
		cType := g.mapType(field.Type)

		g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<div class='mb-3'>\\n\");\n"))
		g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<label class='form-label'>%s</label>\\n\");\n", fieldLabel))

		if cType == "char*" {
			// Check if description field for textarea
			if field.Name == "description" {
				g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<textarea name='%s' class='form-control' rows='3' required></textarea>\\n\");\n", field.Name))
			} else {
				g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<input type='text' name='%s' class='form-control' required>\\n\");\n", field.Name))
			}
		} else if field.Type == "bool" {
			g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<input type='checkbox' name='%s' class='form-check-input'>\\n\");\n", field.Name))
		} else {
			g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"<input type='number' name='%s' class='form-control' required>\\n\");\n", field.Name))
		}

		g.output.WriteString(fmt.Sprintf("    offset += sprintf(html + offset, \"</div>\\n\");\n"))
	}

	g.output.WriteString("    offset += sprintf(html + offset, \"<button type='submit' class='btn btn-primary'>Add Item</button>\\n\");\n")
	g.output.WriteString("    offset += sprintf(html + offset, \"</form>\\n\");\n\n")
}

func (g *CGenerator) generateRouteHandler() {
	g.output.WriteString(`
// URL decode helper
void url_decode(char *dst, const char *src) {
    char a, b;
    while (*src) {
        if ((*src == '%') && ((a = src[1]) && (b = src[2])) && (isxdigit(a) && isxdigit(b))) {
            if (a >= 'a') a -= 'a'-'A';
            if (a >= 'A') a -= ('A' - 10);
            else a -= '0';
            if (b >= 'a') b -= 'a'-'A';
            if (b >= 'A') b -= ('A' - 10);
            else b -= '0';
            *dst++ = 16*a+b;
            src+=3;
        } else if (*src == '+') {
            *dst++ = ' ';
            src++;
        } else {
            *dst++ = *src++;
        }
    }
    *dst++ = '\0';
}

// Parse form data
void parse_form_data(const char* data, char fields[][256], char values[][256], int* count) {
    char* datacopy = strdup(data);
    char* pair = strtok(datacopy, "&");
    *count = 0;

    while (pair != NULL && *count < 10) {
        char* eq = strchr(pair, '=');
        if (eq) {
            *eq = '\0';
            url_decode(fields[*count], pair);
            url_decode(values[*count], eq + 1);
            (*count)++;
        }
        pair = strtok(NULL, "&");
    }
    free(datacopy);
}

`)

	if len(g.structs) > 0 {
		s := g.structs[0]
		tableName := g.getTableName(s)

		g.output.WriteString(`
// HTTP request handler
enum MHD_Result handle_request(void *cls, struct MHD_Connection *connection,
                   const char *url, const char *method,
                   const char *version, const char *upload_data,
                   size_t *upload_data_size, void **con_cls) {

    struct MHD_Response *response;
    int ret;

`)

		// Handle POST for create
		g.output.WriteString(fmt.Sprintf("    if (strcmp(url, \"/%s/create\") == 0 && strcmp(method, \"POST\") == 0) {\n", tableName))
		g.output.WriteString("        // First call: set up\n")
		g.output.WriteString("        if (*con_cls == NULL) {\n")
		g.output.WriteString("            *con_cls = (void*)1;\n")
		g.output.WriteString("            return MHD_YES;\n")
		g.output.WriteString("        }\n\n")

		g.output.WriteString("        // Process POST data\n")
		g.output.WriteString("        if (*upload_data_size != 0) {\n")
		g.output.WriteString("            char fields[10][256];\n")
		g.output.WriteString("            char values[10][256];\n")
		g.output.WriteString("            int count;\n")
		g.output.WriteString("            parse_form_data(upload_data, fields, values, &count);\n\n")

		// Extract form values and create record
		g.output.WriteString("            // Extract form values\n")
		nonAutoFields := []*ast.FieldDecl{}
		for _, field := range s.Fields {
			isAuto := false
			for _, dec := range field.Decorators {
				if dec.Name == "autoincrement" || dec.Name == "primary" {
					isAuto = true
				}
			}
			if !isAuto && !field.IsArray {
				nonAutoFields = append(nonAutoFields, field)
				cType := g.mapType(field.Type)
				if cType == "char*" {
					g.output.WriteString(fmt.Sprintf("            char* %s = \"\";\n", field.Name))
				} else if field.Type == "bool" {
					g.output.WriteString(fmt.Sprintf("            int %s = 0;\n", field.Name))
				} else {
					g.output.WriteString(fmt.Sprintf("            int64_t %s = 0;\n", field.Name))
				}
			}
		}

		g.output.WriteString("            for (int i = 0; i < count; i++) {\n")
		for _, field := range nonAutoFields {
			cType := g.mapType(field.Type)
			if cType == "char*" {
				g.output.WriteString(fmt.Sprintf("                if (strcmp(fields[i], \"%s\") == 0) %s = strdup(values[i]);\n",
					field.Name, field.Name))
			} else if field.Type == "bool" {
				g.output.WriteString(fmt.Sprintf("                if (strcmp(fields[i], \"%s\") == 0) %s = 1;\n",
					field.Name, field.Name))
			} else {
				g.output.WriteString(fmt.Sprintf("                if (strcmp(fields[i], \"%s\") == 0) %s = atoll(values[i]);\n",
					field.Name, field.Name))
			}
		}
		g.output.WriteString("            }\n\n")

		// Call create function
		g.output.WriteString(fmt.Sprintf("            %s_create(", s.Name))
		args := []string{}
		for _, field := range nonAutoFields {
			args = append(args, field.Name)
		}
		g.output.WriteString(strings.Join(args, ", "))
		g.output.WriteString(");\n\n")

		g.output.WriteString("            *upload_data_size = 0;\n")
		g.output.WriteString("            return MHD_YES;\n")
		g.output.WriteString("        }\n\n")

		g.output.WriteString("        // Send redirect response\n")
		g.output.WriteString("        const char* redirect = \"<html><head><meta http-equiv='refresh' content='0;url=/'></head></html>\";\n")
		g.output.WriteString("        response = MHD_create_response_from_buffer(strlen(redirect), (void*)redirect, MHD_RESPMEM_PERSISTENT);\n")
		g.output.WriteString("        ret = MHD_queue_response(connection, MHD_HTTP_SEE_OTHER, response);\n")
		g.output.WriteString("        MHD_add_response_header(response, \"Location\", \"/\");\n")
		g.output.WriteString("        MHD_destroy_response(response);\n")
		g.output.WriteString("        return ret;\n")
		g.output.WriteString("    }\n\n")

		// Handle GET for delete
		g.output.WriteString(fmt.Sprintf("    if (strncmp(url, \"/%s/delete\", %d) == 0 && strcmp(method, \"GET\") == 0) {\n",
			tableName, len("/"+tableName+"/delete")))
		g.output.WriteString("        const char* id_str = MHD_lookup_connection_value(connection, MHD_GET_ARGUMENT_KIND, \"id\");\n")
		g.output.WriteString("        if (id_str) {\n")
		g.output.WriteString("            int64_t id = atoll(id_str);\n")
		g.output.WriteString(fmt.Sprintf("            %s_delete(id);\n", s.Name))
		g.output.WriteString("        }\n")
		g.output.WriteString("        const char* redirect = \"<html><head><meta http-equiv='refresh' content='0;url=/'></head></html>\";\n")
		g.output.WriteString("        response = MHD_create_response_from_buffer(strlen(redirect), (void*)redirect, MHD_RESPMEM_PERSISTENT);\n")
		g.output.WriteString("        ret = MHD_queue_response(connection, MHD_HTTP_OK, response);\n")
		g.output.WriteString("        MHD_destroy_response(response);\n")
		g.output.WriteString("        return ret;\n")
		g.output.WriteString("    }\n\n")

		// Handle root path - show page
		if len(g.pages) > 0 {
			page := g.pages[0]
			g.output.WriteString("    if (strcmp(url, \"/\") == 0 && strcmp(method, \"GET\") == 0) {\n")
			g.output.WriteString(fmt.Sprintf("        char* html = render_%s_page();\n", strings.ToLower(page.Name)))
			g.output.WriteString("        response = MHD_create_response_from_buffer(strlen(html), (void*)html, MHD_RESPMEM_MUST_FREE);\n")
			g.output.WriteString("        MHD_add_response_header(response, \"Content-Type\", \"text/html\");\n")
			g.output.WriteString("        ret = MHD_queue_response(connection, MHD_HTTP_OK, response);\n")
			g.output.WriteString("        MHD_destroy_response(response);\n")
			g.output.WriteString("        return ret;\n")
			g.output.WriteString("    }\n\n")
		}

		g.output.WriteString("    // 404\n")
		g.output.WriteString("    const char* not_found = \"<h1>404 Not Found</h1>\";\n")
		g.output.WriteString("    response = MHD_create_response_from_buffer(strlen(not_found), (void*)not_found, MHD_RESPMEM_PERSISTENT);\n")
		g.output.WriteString("    ret = MHD_queue_response(connection, MHD_HTTP_NOT_FOUND, response);\n")
		g.output.WriteString("    MHD_destroy_response(response);\n")
		g.output.WriteString("    return ret;\n")
		g.output.WriteString("}\n\n")
	}
}

func (g *CGenerator) generateWebMain() {
	g.output.WriteString("int main(int argc, char *argv[]) {\n")
	g.output.WriteString("    // Initialize database\n")
	g.output.WriteString("    int rc = sqlite3_open(\"app.db\", &db);\n")
	g.output.WriteString("    if (rc != SQLITE_OK) {\n")
	g.output.WriteString("        fprintf(stderr, \"Cannot open database: %s\\n\", sqlite3_errmsg(db));\n")
	g.output.WriteString("        return 1;\n")
	g.output.WriteString("    }\n")
	g.output.WriteString("    printf(\"Database opened successfully\\n\");\n\n")

	// Initialize tables
	for _, s := range g.structs {
		g.output.WriteString(fmt.Sprintf("    %s_init_table();\n", s.Name))
	}

	g.output.WriteString("\n    // Start HTTP server\n")
	g.output.WriteString("    http_daemon = MHD_start_daemon(MHD_USE_SELECT_INTERNALLY, 8080, NULL, NULL,\n")
	g.output.WriteString("                                    &handle_request, NULL, MHD_OPTION_END);\n")
	g.output.WriteString("    if (http_daemon == NULL) {\n")
	g.output.WriteString("        fprintf(stderr, \"Failed to start HTTP server\\n\");\n")
	g.output.WriteString("        return 1;\n")
	g.output.WriteString("    }\n\n")

	g.output.WriteString("    printf(\"\\n========================================\\n\");\n")
	g.output.WriteString("    printf(\"Server running on http://localhost:8080\\n\");\n")
	g.output.WriteString("    printf(\"Press ENTER to stop the server...\\n\");\n")
	g.output.WriteString("    printf(\"========================================\\n\\n\");\n\n")

	g.output.WriteString("    getchar();\n\n")

	g.output.WriteString("    // Stop HTTP server\n")
	g.output.WriteString("    MHD_stop_daemon(http_daemon);\n\n")

	g.output.WriteString("    // Close database\n")
	g.output.WriteString("    sqlite3_close(db);\n")
	g.output.WriteString("    printf(\"Server stopped\\n\");\n")
	g.output.WriteString("    return 0;\n")
	g.output.WriteString("}\n")
}
