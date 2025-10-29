package parser

import (
	"fmt"
	"github.com/gunesh/zelang/pkg/ast"
	"github.com/gunesh/zelang/pkg/lexer"
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  lexer.Token
	peekToken lexer.Token
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	// Read two tokens so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("Line %d:%d: expected next token to be %s, got %s instead",
		p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

// ParseProgram parses the entire program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Node{}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Node {
	switch p.curToken.Type {
	case lexer.AT:
		return p.parseDecoratedStatement()
	case lexer.STRUCT:
		return p.parseStructDecl()
	case lexer.PAGE:
		return p.parsePageDecl()
	case lexer.HANDLER:
		return p.parseHandlerDecl()
	case lexer.INT_TYPE, lexer.FLOAT_TYPE, lexer.STRING_TYPE, lexer.BOOL_TYPE, lexer.VOID:
		return p.parseFunctionDecl()
	default:
		return nil
	}
}

func (p *Parser) parseDecoratedStatement() ast.Node {
	decorators := p.parseDecorators()

	// After decorators, determine what kind of statement follows
	switch p.curToken.Type {
	case lexer.STRUCT:
		structDecl := p.parseStructDecl()
		if structDecl != nil {
			structDecl.Decorators = decorators
		}
		return structDecl
	case lexer.PAGE:
		pageDecl := p.parsePageDecl()
		if pageDecl != nil {
			pageDecl.Decorators = decorators
		}
		return pageDecl
	case lexer.HANDLER:
		handlerDecl := p.parseHandlerDecl()
		if handlerDecl != nil {
			handlerDecl.Decorators = decorators
		}
		return handlerDecl
	default:
		return nil
	}
}

func (p *Parser) parseDecorators() []*ast.Decorator {
	decorators := []*ast.Decorator{}

	for p.curTokenIs(lexer.AT) {
		p.nextToken() // skip @

		if !p.curTokenIs(lexer.IDENT) {
			return decorators
		}

		decorator := &ast.Decorator{
			Name:   p.curToken.Literal,
			Args:   []string{},
			KVArgs: make(map[string]string),
		}

		p.nextToken()

		// Check for arguments
		if p.curTokenIs(lexer.LPAREN) {
			p.nextToken() // skip (

			// Parse arguments
			for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
				if p.curTokenIs(lexer.STRING) || p.curTokenIs(lexer.INT) || p.curTokenIs(lexer.FLOAT) || p.curTokenIs(lexer.IDENT) {
					arg := p.curToken.Literal
					p.nextToken()

					// Check if this is a key: value pair
					if p.curTokenIs(lexer.COLON) {
						p.nextToken() // skip :
						value := p.curToken.Literal
						decorator.KVArgs[arg] = value
						p.nextToken()
					} else {
						decorator.Args = append(decorator.Args, arg)
					}

					if p.curTokenIs(lexer.COMMA) {
						p.nextToken() // skip ,
					}
				} else {
					p.nextToken()
				}
			}

			// We should now be at RPAREN
			if !p.curTokenIs(lexer.RPAREN) {
				p.errors = append(p.errors, fmt.Sprintf("Line %d:%d: expected ')' after decorator arguments", p.curToken.Line, p.curToken.Column))
				return decorators
			}
			p.nextToken() // skip )
		}

		decorators = append(decorators, decorator)
	}

	return decorators
}

func (p *Parser) parseStructDecl() *ast.StructDecl {
	structDecl := &ast.StructDecl{}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	structDecl.Name = p.curToken.Literal

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken() // move to first field or closing brace

	// Parse fields
	structDecl.Fields = []*ast.FieldDecl{}
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		field := p.parseFieldDecl()
		if field != nil {
			structDecl.Fields = append(structDecl.Fields, field)
		}
		p.nextToken()
	}

	return structDecl
}

func (p *Parser) parseFieldDecl() *ast.FieldDecl {
	field := &ast.FieldDecl{}

	// Check for decorators
	if p.curTokenIs(lexer.AT) {
		field.Decorators = p.parseDecorators()
	}

	// Parse type
	if !p.isType(p.curToken.Type) {
		return nil
	}

	field.Type = p.curToken.Literal
	p.nextToken()

	// Check for array type
	if p.curTokenIs(lexer.LBRACKET) {
		field.IsArray = true
		if !p.expectPeek(lexer.RBRACKET) {
			return nil
		}
		p.nextToken()
	}

	// Parse field name
	if !p.curTokenIs(lexer.IDENT) {
		return nil
	}

	field.Name = p.curToken.Literal
	p.nextToken()

	// Expect semicolon
	if !p.curTokenIs(lexer.SEMICOLON) {
		p.errors = append(p.errors, fmt.Sprintf("Line %d:%d: expected ';' after field declaration", p.curToken.Line, p.curToken.Column))
		return nil
	}

	return field
}

func (p *Parser) isType(t lexer.TokenType) bool {
	return t == lexer.INT_TYPE || t == lexer.FLOAT_TYPE || t == lexer.STRING_TYPE ||
		t == lexer.BOOL_TYPE || t == lexer.DATE || t == lexer.DATETIME ||
		t == lexer.IDENT // For custom types
}

func (p *Parser) parsePageDecl() *ast.PageDecl {
	pageDecl := &ast.PageDecl{
		Properties: make(map[string]string),
		Body:       []ast.Node{},
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}

	pageDecl.Name = p.curToken.Literal

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken() // move into body

	// Parse page properties and body
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		// For now, just skip to the end
		p.nextToken()
	}

	return pageDecl
}

func (p *Parser) parseFunctionDecl() *ast.FunctionDecl {
	funcDecl := &ast.FunctionDecl{}

	// Parse return type
	funcDecl.ReturnType = p.curToken.Literal
	p.nextToken()

	// Parse function name
	if !p.curTokenIs(lexer.IDENT) {
		return nil
	}

	funcDecl.Name = p.curToken.Literal

	// For now, skip function body parsing
	// Just skip to the end of the function
	braceCount := 0
	for !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.LBRACE) {
			braceCount++
		} else if p.curTokenIs(lexer.RBRACE) {
			braceCount--
			if braceCount == 0 {
				break
			}
		}
		p.nextToken()
	}

	return funcDecl
}

func (p *Parser) parseHandlerDecl() *ast.HandlerDecl {
	handlerDecl := &ast.HandlerDecl{
		Parameters: []*ast.Parameter{},
		Body:       []ast.Node{},
	}

	p.nextToken() // skip 'handler'

	// Parse handler name
	if !p.curTokenIs(lexer.IDENT) {
		return nil
	}

	handlerDecl.Name = p.curToken.Literal
	p.nextToken()

	// Parse parameters
	if !p.curTokenIs(lexer.LPAREN) {
		return nil
	}

	p.nextToken() // skip (

	// Parse parameter list
	for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
		param := &ast.Parameter{}

		// Parse parameter type
		param.Type = p.curToken.Literal
		p.nextToken()

		// Parse parameter name
		if p.curTokenIs(lexer.IDENT) {
			param.Name = p.curToken.Literal
			handlerDecl.Parameters = append(handlerDecl.Parameters, param)
			p.nextToken()
		}

		if p.curTokenIs(lexer.COMMA) {
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.RPAREN) {
		return nil
	}

	p.nextToken() // skip )

	// Skip to the end of function body
	if p.curTokenIs(lexer.LBRACE) {
		braceCount := 0
		for !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.LBRACE) {
				braceCount++
			} else if p.curTokenIs(lexer.RBRACE) {
				braceCount--
				if braceCount == 0 {
					break
				}
			}
			p.nextToken()
		}
	}

	return handlerDecl
}
