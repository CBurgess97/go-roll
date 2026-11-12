package dice

import "fmt"

// TokenType identifies the kind of a lexed token.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNumber
	TokenDice     // d
	TokenPercent  // % (as in d%)
	TokenExplode  // !
	TokenKeep     // k  (parser decides the default, conventionally keep highest)
	TokenKeepHigh // kh
	TokenKeepLow  // kl
	TokenDropHigh // dh
	TokenDropLow  // dl
	TokenPlus     // +
	TokenMinus    // -
	TokenMultiply // *
	TokenDivide   // /
	TokenLParen   // (
	TokenRParen   // )
)

var tokenNames = map[TokenType]string{
	TokenEOF:      "EOF",
	TokenNumber:   "Number",
	TokenDice:     "Dice",
	TokenPercent:  "Percent",
	TokenExplode:  "Explode",
	TokenKeep:     "Keep",
	TokenKeepHigh: "KeepHigh",
	TokenKeepLow:  "KeepLow",
	TokenDropHigh: "DropHigh",
	TokenDropLow:  "DropLow",
	TokenPlus:     "Plus",
	TokenMinus:    "Minus",
	TokenMultiply: "Multiply",
	TokenDivide:   "Divide",
	TokenLParen:   "LParen",
	TokenRParen:   "RParen",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TokenType(%d)", int(t))
}

// Token is a single lexical unit of a dice notation string.
type Token struct {
	Type  TokenType
	Value string // the source text of the token
	Pos   int    // byte offset of the token in the input
}

func (t Token) String() string {
	if t.Type == TokenEOF {
		return "EOF"
	}
	return fmt.Sprintf("%s(%q)", t.Type, t.Value)
}

// LexError reports a character that is not valid dice notation.
type LexError struct {
	Input string
	Pos   int
}

func (e *LexError) Error() string {
	return fmt.Sprintf("unexpected character %q at position %d in %q", e.Input[e.Pos], e.Pos, e.Input)
}

// Lexer turns a dice notation string into tokens, one at a time.
type Lexer struct {
	input string
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// Tokenize lexes the whole input. The returned slice always ends with a
// TokenEOF on success.
func Tokenize(input string) ([]Token, error) {
	l := NewLexer(input)
	var tokens []Token
	for {
		tok, err := l.Next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			return tokens, nil
		}
	}
}

// Next returns the next token, or TokenEOF once the input is exhausted.
func (l *Lexer) Next() (Token, error) {
	for l.pos < len(l.input) && isSpace(l.input[l.pos]) {
		l.pos++
	}
	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos}, nil
	}

	start := l.pos
	c := l.input[l.pos]

	if isDigit(c) {
		for l.pos < len(l.input) && isDigit(l.input[l.pos]) {
			l.pos++
		}
		return l.emit(TokenNumber, start), nil
	}

	l.pos++
	switch lower(c) {
	case 'd':
		// "dh"/"dl" are drop modifiers; a bare "d" is the dice operator.
		switch lower(l.peek()) {
		case 'h':
			l.pos++
			return l.emit(TokenDropHigh, start), nil
		case 'l':
			l.pos++
			return l.emit(TokenDropLow, start), nil
		}
		return l.emit(TokenDice, start), nil
	case 'k':
		switch lower(l.peek()) {
		case 'h':
			l.pos++
			return l.emit(TokenKeepHigh, start), nil
		case 'l':
			l.pos++
			return l.emit(TokenKeepLow, start), nil
		}
		return l.emit(TokenKeep, start), nil
	case '!':
		return l.emit(TokenExplode, start), nil
	case '%':
		return l.emit(TokenPercent, start), nil
	case '+':
		return l.emit(TokenPlus, start), nil
	case '-':
		return l.emit(TokenMinus, start), nil
	case '*':
		return l.emit(TokenMultiply, start), nil
	case '/':
		return l.emit(TokenDivide, start), nil
	case '(':
		return l.emit(TokenLParen, start), nil
	case ')':
		return l.emit(TokenRParen, start), nil
	}

	l.pos = start
	return Token{}, &LexError{Input: l.input, Pos: start}
}

func (l *Lexer) emit(t TokenType, start int) Token {
	return Token{Type: t, Value: l.input[start:l.pos], Pos: start}
}

// peek returns the next unread byte, or 0 at end of input.
func (l *Lexer) peek() byte {
	if l.pos < len(l.input) {
		return l.input[l.pos]
	}
	return 0
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
