package lexer

type TokenType string

const (
	// Special tokens
	EOF     TokenType = "EOF"
	ILLEGAL TokenType = "ILLEGAL"

	// Identifiers and literals
	IDENT  TokenType = "IDENT"  // variable names, function names
	INT    TokenType = "INT"    // 123
	FLOAT  TokenType = "FLOAT"  // 123.45
	STRING TokenType = "STRING" // "hello"

	// Keywords
	STRUCT   TokenType = "STRUCT"
	INT_TYPE TokenType = "INT_TYPE"    // int
	FLOAT_TYPE TokenType = "FLOAT_TYPE" // float
	STRING_TYPE TokenType = "STRING_TYPE" // string
	BOOL_TYPE TokenType = "BOOL_TYPE"   // bool
	DATE TokenType = "DATE"     // date
	DATETIME TokenType = "DATETIME" // datetime
	IF       TokenType = "IF"
	ELSE     TokenType = "ELSE"
	FOR      TokenType = "FOR"
	WHILE    TokenType = "WHILE"
	RETURN   TokenType = "RETURN"
	TRUE     TokenType = "TRUE"
	FALSE    TokenType = "FALSE"
	VOID     TokenType = "VOID"
	PAGE     TokenType = "PAGE"
	SECTION  TokenType = "SECTION"
	ROW      TokenType = "ROW"
	COLUMN   TokenType = "COLUMN"
	FORM     TokenType = "FORM"
	INPUT    TokenType = "INPUT"
	BUTTON   TokenType = "BUTTON"
	DATALIST TokenType = "DATALIST"
	HANDLER  TokenType = "HANDLER"
	REQUEST  TokenType = "REQUEST"
	RESPONSE TokenType = "RESPONSE"

	// Operators
	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	LT       TokenType = "<"
	GT       TokenType = ">"
	EQ       TokenType = "=="
	NOT_EQ   TokenType = "!="
	LTE      TokenType = "<="
	GTE      TokenType = ">="
	AND      TokenType = "&&"
	OR       TokenType = "||"
	NOT      TokenType = "!"

	// Delimiters
	COMMA     TokenType = ","
	SEMICOLON TokenType = ";"
	COLON     TokenType = ":"
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"
	AT        TokenType = "@" // For decorators

	// Decorator keywords
	DECORATOR TokenType = "DECORATOR" // @something
)

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

var keywords = map[string]TokenType{
	"struct":   STRUCT,
	"int":      INT_TYPE,
	"float":    FLOAT_TYPE,
	"string":   STRING_TYPE,
	"bool":     BOOL_TYPE,
	"date":     DATE,
	"datetime": DATETIME,
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"while":    WHILE,
	"return":   RETURN,
	"true":     TRUE,
	"false":    FALSE,
	"void":     VOID,
	"Page":     PAGE,
	"Section":  SECTION,
	"Row":      ROW,
	"Column":   COLUMN,
	"Form":     FORM,
	"Input":    INPUT,
	"Button":   BUTTON,
	"DataList": DATALIST,
	"handler":  HANDLER,
	"Request":  REQUEST,
	"Response": RESPONSE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
