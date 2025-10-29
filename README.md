# ZeLang - Database-First Compiled Language

ZeLang is a compiled programming language for building database-driven CRUD applications with web interfaces. Define your data structures with decorators, declare your UI components, compile to a native binary, and get automatic database operations, CRUD APIs, and Bootstrap-styled web pages.

## Status

**Phase 1 Working** - Structs, CRUD operations, HTTP server, and Bootstrap UI generation fully functional!

## Quick Start

### Prerequisites

- Go 1.19+ (for building the compiler)
- GCC or Clang (for compiling generated C code)
- SQLite3 library
- libmicrohttpd (for web server features)

Install on macOS:
```bash
brew install go sqlite3 libmicrohttpd
xcode-select --install
```

### Build the Compiler

```bash
# Clone the repository
cd zelang

# Build the compiler
make

# Or manually:
cd cmd/zelang && go build -o ../../zelang
```

### Run Example

```bash
# Build a simple CRUD example
./zelang build examples/student.zl
./student

# Build the web todo app
./zelang build examples/todo.zl
./todo
# Open http://localhost:8080 in your browser
```

---

## Language Specification

### Data Types

ZeLang supports the following primitive types:

| Type | Description | Maps to C | Maps to SQL |
|------|-------------|-----------|-------------|
| `int` | 64-bit integer | `int64_t` | `INTEGER` |
| `float` | Double precision | `double` | `REAL` |
| `string` | Text string | `char*` | `TEXT` |
| `bool` | Boolean | `int` | `INTEGER` |
| `date` | Date string | `char*` | `TEXT` |
| `datetime` | Datetime string | `char*` | `TEXT` |

### Struct Declaration

Define data structures that map to database tables:

```c
@storage(sqlite)
@table("products")
struct Product {
    @primary
    @autoincrement
    int id;

    @required
    @length(max: 200)
    string name;

    @required
    float price;

    int quantity;
}
```

### Decorators

#### Struct-Level Decorators

| Decorator | Arguments | Purpose | Example |
|-----------|-----------|---------|---------|
| `@storage` | Backend name | Specify storage backend | `@storage(sqlite)` |
| `@table` | Table name | Custom table name | `@table("products")` |

#### Field-Level Decorators

| Decorator | Arguments | Purpose | SQL Effect |
|-----------|-----------|---------|------------|
| `@primary` | None | Mark as primary key | `PRIMARY KEY` |
| `@autoincrement` | None | Auto-increment field | `AUTOINCREMENT` |
| `@required` | None | NOT NULL constraint | `NOT NULL` |
| `@unique` | None | Unique constraint | `UNIQUE` |
| `@length` | `max: n` | Max length validation | *(validation only)* |

### Web UI Components

#### Page Declaration

Define a web page with route and UI components:

```c
@route("/")
Page TodoApp {
    title: "Todo Manager";

    DataList {
        source: Todo.all();
        columns: ["id", "title", "completed"];
        actions: ["edit", "delete"];
    }

    Form {
        action: "/todo/create";
        fields: {
            title: { type: "text", required: true },
            description: { type: "textarea", required: true },
            completed: { type: "checkbox" }
        };
        submit: "Add Todo";
    }
}
```

#### DataList Component

Displays database records in a Bootstrap table:

```c
DataList {
    source: Model.all();           // Data source
    columns: ["field1", "field2"]; // Columns to show
    actions: ["edit", "delete"];   // Action buttons
}
```

#### Form Component

Generates Bootstrap forms for data entry:

```c
Form {
    action: "/path/create";        // Form submit URL
    fields: {
        fieldName: {
            type: "text",          // text, textarea, checkbox, number
            required: true,        // Validation
            placeholder: "..."     // Placeholder text
        }
    };
    submit: "Button Text";         // Submit button text
}
```

### Route Handlers

Define custom request handlers:

```c
@route("/todo/create")
@method(POST)
handler createTodo(Request req, Response res) {
    Todo* todo = Todo_create(
        req.form["title"],
        req.form["description"],
        req.form["completed"] == "on"
    );
    res.redirect("/");
}

@route("/todo/delete/:id")
handler deleteTodo(Request req, Response res) {
    int id = req.params["id"];
    Todo_delete(id);
    res.redirect("/");
}
```

#### Handler Decorators

| Decorator | Purpose | Example |
|-----------|---------|---------|
| `@route` | URL path | `@route("/users")` |
| `@method` | HTTP method | `@method(POST)`, `@method(GET)` |

### Auto-Generated CRUD Functions

For each struct, ZeLang generates:

```c
// Create a new record
Model* Model_create(field1, field2, ...);

// Find record by ID
Model* Model_find(int64_t id);

// Get all records
Model** Model_all(int* count);

// Delete record by ID
int Model_delete(int64_t id);

// Initialize table (called automatically)
void Model_init_table();
```

### Build and Run

```bash
# Build your application
./zelang build myapp.zl

# This generates:
# 1. myapp.c - Generated C code with CRUD and HTTP server
# 2. myapp - Native binary (~50KB)
```

**For CLI applications** (no web UI):
```bash
./myapp
# Creates database and tables
# Runs demo CRUD operations
```

**For web applications** (with Page/Form components):
```bash
./myapp
# Server running on http://localhost:8080
# Press ENTER to stop the server...
```

The binary automatically:
- Creates `app.db` SQLite database
- Creates tables from struct definitions
- Initializes schema with constraints
- **For web apps:** Starts HTTP server on port 8080
- **For web apps:** Serves Bootstrap-styled pages
- **For web apps:** Handles form submissions and CRUD operations

## Project Structure

```
zelang/
├── cmd/zelang/          # Compiler CLI
├── pkg/
│   ├── lexer/          # Tokenizer
│   ├── parser/         # Parser
│   ├── ast/            # AST definitions
│   └── codegen/        # C code generator
├── examples/           # Example applications
├── Makefile           # Build system
└── README.md
```

## Complete Example: Todo Web App

```c
// todo.zl - Complete web application in 50 lines

@storage(sqlite)
@table("todos")
struct Todo {
    @primary
    @autoincrement
    int id;

    @required
    @length(max: 200)
    string title;

    @required
    string description;

    bool completed;
}

@route("/")
Page TodoApp {
    title: "Todo Manager";

    DataList {
        source: Todo.all();
        columns: ["id", "title", "completed"];
        actions: ["edit", "delete"];
    }

    Form {
        action: "/todo/create";
        fields: {
            title: { type: "text", required: true },
            description: { type: "textarea", required: true },
            completed: { type: "checkbox" }
        };
        submit: "Add Todo";
    }
}

@route("/todo/create")
@method(POST)
handler createTodo(Request req, Response res) {
    Todo_create(
        req.form["title"],
        req.form["description"],
        req.form["completed"] == "on"
    );
    res.redirect("/");
}

@route("/todo/delete/:id")
handler deleteTodo(Request req, Response res) {
    Todo_delete(req.params["id"]);
    res.redirect("/");
}
```

**Compile and run:**
```bash
./zelang build todo.zl
./todo
# Open http://localhost:8080
```

**Screenshot:**

![Todo App Screenshot](docs/todo-screenshot.png)

**You get:**
- ✅ SQLite database with `todos` table
- ✅ Full CRUD operations (Create, Read, Delete)
- ✅ Bootstrap-styled web interface
- ✅ Responsive table showing all todos
- ✅ Form for adding new todos
- ✅ Delete buttons for each item
- ✅ Native binary (~60KB)
- ✅ Zero dependencies at runtime (SQLite embedded)

## Current Features

### ✅ Compiler & Core
- ✅ Full lexer with all tokens
- ✅ Parser for structs, pages, handlers
- ✅ AST generation
- ✅ C code generation
- ✅ Decorator parsing and validation
- ✅ Automatic GCC compilation

### ✅ Database Features
- ✅ SQLite integration
- ✅ Automatic table creation
- ✅ Primary keys and auto-increment
- ✅ NOT NULL and UNIQUE constraints
- ✅ Complete CRUD operations:
  - ✅ `Model_create()` - INSERT with prepared statements
  - ✅ `Model_find()` - SELECT by ID
  - ✅ `Model_all()` - SELECT all with dynamic arrays
  - ✅ `Model_delete()` - DELETE by ID
- ✅ Type mapping (int→INTEGER, string→TEXT, bool→INTEGER)

### ✅ Web Server Features
- ✅ HTTP server with libmicrohttpd
- ✅ Route handling (GET, POST)
- ✅ Form data parsing (URL-encoded)
- ✅ Request parameters
- ✅ HTTP redirects
- ✅ Static and dynamic routing

### ✅ UI Generation
- ✅ Bootstrap 5 integration (CDN)
- ✅ Responsive HTML generation
- ✅ DataList component (tables)
- ✅ Form component with validation
- ✅ Input types: text, textarea, checkbox, number
- ✅ CRUD action buttons
- ✅ Automatic field type detection

## Roadmap

### Phase 1 ✅ (COMPLETE)
- [x] Lexer and parser
- [x] Struct parsing with decorators
- [x] C code generation
- [x] SQLite table creation
- [x] Complete CRUD implementation
- [x] HTTP server with libmicrohttpd
- [x] Page and Form UI components
- [x] DataList component
- [x] Bootstrap UI generation
- [x] Route handlers

### Phase 2 (Next - 4-6 weeks)
- [ ] UPDATE operations
- [ ] Edit forms and pages
- [ ] Field validation implementation
- [ ] Foreign keys (@foreign_key decorator)
- [ ] Relationships (one-to-many, many-to-many)
- [ ] Query filters and search
- [ ] Pagination for DataList
- [ ] File upload support
- [ ] Authentication decorators
- [ ] Custom CSS/themes

### Phase 3 (Future - 2-3 months)
- [ ] MySQL backend support
- [ ] PostgreSQL backend support
- [ ] Session management
- [ ] Cookie handling
- [ ] JSON API endpoints
- [ ] WebSocket support
- [ ] Template inheritance
- [ ] Multi-page applications
- [ ] Admin panel generation
- [ ] Migration system

## Development

### Build

```bash
make
```

### Test

```bash
make test
```

### Clean

```bash
make clean
```

## Contributing

This is an early prototype. Contributions welcome!

## License

MIT License

## Author

Gunesh - Building the future of database-first development
