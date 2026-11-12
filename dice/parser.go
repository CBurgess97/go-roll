package dice

import (
	"fmt"
	"strconv"
)

// Result is the outcome of evaluating a dice notation string.
type Result struct {
	Total int
	Rolls []DiceRoll // every dice term, in the order it appears in the input
}

// ParseError reports a problem with the structure or values of the input.
type ParseError struct {
	Pos int // byte offset in the input
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at position %d", e.Msg, e.Pos)
}

// Parser evaluates dice notation as it parses it. Grammar:
//
//	expr     = term   { ("+" | "-") term }
//	term     = unary  { ("*" | "/") unary }
//	unary    = "-" unary | primary
//	primary  = Number [ dice ] | dice | "(" expr ")"
//	dice     = "d" ( Number | "%" ) { modifier }
//	modifier = "!" | ( "k" | "kh" | "kl" | "dh" | "dl" ) [ Number ]
//
// A dice term with no leading number rolls one die, and a keep/drop modifier
// with no number applies to one die. Division truncates toward zero.
type Parser struct {
	input  string
	tokens []Token
	pos    int
	roller *Roller
	rolls  []DiceRoll
}

// Parse tokenizes and evaluates input using the default random source.
func Parse(input string) (*Result, error) {
	return ParseWith(input, NewRoller())
}

// ParseWith is Parse with a caller-supplied Roller.
func ParseWith(input string, roller *Roller) (*Result, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}
	p := &Parser{input: input, tokens: tokens, roller: roller}

	total, err := p.expr()
	if err != nil {
		return nil, err
	}
	if tok := p.peek(); tok.Type != TokenEOF {
		return nil, p.unexpected(tok)
	}
	return &Result{Total: total, Rolls: p.rolls}, nil
}

func (p *Parser) peek() Token {
	return p.tokens[p.pos]
}

// next consumes and returns the current token. It never advances past EOF.
func (p *Parser) next() Token {
	tok := p.tokens[p.pos]
	if tok.Type != TokenEOF {
		p.pos++
	}
	return tok
}

// accept consumes the current token if it has type t.
func (p *Parser) accept(t TokenType) (Token, bool) {
	if p.peek().Type == t {
		return p.next(), true
	}
	return Token{}, false
}

func (p *Parser) unexpected(tok Token) error {
	if tok.Type == TokenEOF {
		return &ParseError{Pos: tok.Pos, Msg: "unexpected end of input"}
	}
	return &ParseError{Pos: tok.Pos, Msg: fmt.Sprintf("unexpected %q", tok.Value)}
}

func (p *Parser) expr() (int, error) {
	left, err := p.term()
	if err != nil {
		return 0, err
	}
	for {
		switch p.peek().Type {
		case TokenPlus, TokenMinus:
			op := p.next()
			right, err := p.term()
			if err != nil {
				return 0, err
			}
			if op.Type == TokenPlus {
				left += right
			} else {
				left -= right
			}
		default:
			return left, nil
		}
	}
}

func (p *Parser) term() (int, error) {
	left, err := p.unary()
	if err != nil {
		return 0, err
	}
	for {
		switch p.peek().Type {
		case TokenMultiply, TokenDivide:
			op := p.next()
			right, err := p.unary()
			if err != nil {
				return 0, err
			}
			if op.Type == TokenMultiply {
				left *= right
			} else {
				if right == 0 {
					return 0, &ParseError{Pos: op.Pos, Msg: "division by zero"}
				}
				left /= right
			}
		default:
			return left, nil
		}
	}
}

func (p *Parser) unary() (int, error) {
	if _, ok := p.accept(TokenMinus); ok {
		v, err := p.unary()
		return -v, err
	}
	return p.primary()
}

func (p *Parser) primary() (int, error) {
	tok := p.peek()
	switch tok.Type {
	case TokenNumber:
		p.next()
		n, err := p.number(tok)
		if err != nil {
			return 0, err
		}
		if p.peek().Type == TokenDice {
			return p.dice(n, tok.Pos)
		}
		return n, nil
	case TokenDice:
		return p.dice(1, tok.Pos)
	case TokenLParen:
		p.next()
		v, err := p.expr()
		if err != nil {
			return 0, err
		}
		if _, ok := p.accept(TokenRParen); !ok {
			return 0, &ParseError{Pos: p.peek().Pos, Msg: `expected ")"`}
		}
		return v, nil
	}
	return 0, p.unexpected(tok)
}

// dice parses a dice term whose "d" is the current token, rolls it, records
// the roll and returns its total. start is where the term begins in the
// input, which is before the "d" when a count was given.
func (p *Parser) dice(count, start int) (int, error) {
	p.next() // "d"

	var sides int
	switch tok := p.next(); tok.Type {
	case TokenNumber:
		n, err := p.number(tok)
		if err != nil {
			return 0, err
		}
		sides = n
	case TokenPercent:
		sides = 100
	default:
		return 0, &ParseError{Pos: tok.Pos, Msg: `expected a number of sides or "%" after "d"`}
	}

	modKinds := map[TokenType]ModifierKind{
		TokenExplode:  ModExplode,
		TokenKeep:     ModKeepHigh,
		TokenKeepHigh: ModKeepHigh,
		TokenKeepLow:  ModKeepLow,
		TokenDropHigh: ModDropHigh,
		TokenDropLow:  ModDropLow,
	}
	var mods []Modifier
	for {
		kind, ok := modKinds[p.peek().Type]
		if !ok {
			break
		}
		p.next()
		mod := Modifier{Kind: kind, N: 1}
		if kind != ModExplode {
			if tok, ok := p.accept(TokenNumber); ok {
				n, err := p.number(tok)
				if err != nil {
					return 0, err
				}
				mod.N = n
			}
		}
		mods = append(mods, mod)
	}

	roll, err := p.roller.Roll(count, sides, mods)
	if err != nil {
		return 0, &ParseError{Pos: start, Msg: err.Error()}
	}
	last := p.tokens[p.pos-1]
	roll.Notation = p.input[start : last.Pos+len(last.Value)]
	p.rolls = append(p.rolls, roll)
	return roll.Total, nil
}

func (p *Parser) number(tok Token) (int, error) {
	n, err := strconv.Atoi(tok.Value)
	if err != nil {
		return 0, &ParseError{Pos: tok.Pos, Msg: fmt.Sprintf("number %s is too large", tok.Value)}
	}
	return n, nil
}
