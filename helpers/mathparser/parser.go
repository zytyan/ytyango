package mathparser

const (
	precLowest = iota
	precAdd
	precMul
	precUnary
	precPow
	precPostfix
)

type nodeBuilder func(tok Token, left node, right node) node

type Op struct {
	leftBp         int
	rightBp        int
	hasRightOprand bool
	ExpectTokens   []TokenType
	build          nodeBuilder
}

func buildPassNode(_ Token, _ node, right node) node {
	return right
}

func buildAbsNode(tok Token, _ node, right node) node {
	return &absNode{expr: right, pos: tok.pos}
}

func buildBinaryNode(tok Token, left node, right node) node {
	return &binaryNode{op: tok.typ, left: left, right: right, pos: tok.pos, label: tok.str}
}

func buildUnaryNode(op TokenType) nodeBuilder {
	return func(tok Token, _ node, right node) node {
		return &unaryNode{op: op, expr: right, pos: tok.pos, label: tok.str}
	}
}

func buildPostfixNode(op TokenType) nodeBuilder {
	return func(tok Token, left node, _ node) node {
		return &postfixNode{op: op, expr: left, pos: tok.pos, label: tok.str}
	}
}

var prefixOps = map[TokenType]Op{
	LPAREN: {
		rightBp:        precLowest,
		hasRightOprand: true,
		ExpectTokens:   []TokenType{RPAREN},
		build:          buildPassNode,
	},
	PIPE: {
		rightBp:        precLowest,
		hasRightOprand: true,
		ExpectTokens:   []TokenType{PIPE},
		build:          buildAbsNode,
	},
	PLUS: {
		rightBp:        precUnary,
		hasRightOprand: true,
		build:          buildUnaryNode(PLUS),
	},
	MINUS: {
		rightBp:        precUnary,
		hasRightOprand: true,
		build:          buildUnaryNode(MINUS),
	},
	SQRT: {
		rightBp:        precUnary,
		hasRightOprand: true,
		build:          buildUnaryNode(SQRT),
	},
}

var infixOps = map[TokenType]Op{
	PLUS:     binaryOp(precAdd, precAdd),
	MINUS:    binaryOp(precAdd, precAdd),
	MUL:      binaryOp(precMul, precMul),
	DIV:      binaryOp(precMul, precMul),
	FLOORDIV: binaryOp(precMul, precMul),
	MOD:      binaryOp(precMul, precMul),
	POW:      binaryOp(precPow, precPow-1),
	PERM:     binaryOp(precPostfix, precPostfix),
	COMB:     binaryOp(precPostfix, precPostfix),
	FACT: {
		leftBp:         precPostfix,
		hasRightOprand: false,
		build:          buildPostfixNode(FACT),
	},
}

var percentPostfixOp = Op{
	leftBp:         precPostfix,
	hasRightOprand: false,
	build:          buildPostfixNode(MOD),
}

func binaryOp(leftBp, rightBp int) Op {
	return Op{
		leftBp:         leftBp,
		rightBp:        rightBp,
		hasRightOprand: true,
		build:          buildBinaryNode,
	}
}

type parser struct {
	tokens []Token
	pos    int
}

func parse(tokens []Token) (node, error) {
	p := &parser{tokens: tokens}
	expr, err := p.parseExpression(precLowest)
	if err != nil {
		return nil, err
	}
	if tok := p.peek(); tok.typ != EOF {
		return nil, errorAt(tok.pos, ErrorUnexpectedToken, "unexpected token: %s", tok.str)
	}
	return expr, nil
}

func (p *parser) parseExpression(minBP int) (node, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.peek()
		if tok.typ == EOF || tok.typ == RPAREN || tok.typ == PIPE {
			break
		}

		op, ok := p.infixOp(tok)
		if !ok || op.leftBp <= minBP {
			break
		}

		p.next()
		var right node
		if op.hasRightOprand {
			right, err = p.parseExpression(op.rightBp)
			if err != nil {
				return nil, err
			}
		}
		if err := p.consumeExpectedTokens(tok, op); err != nil {
			return nil, err
		}
		left = op.build(tok, left, right)
	}
	return left, nil
}

func (p *parser) parsePrefix() (node, error) {
	tok := p.next()
	switch tok.typ {
	case NUMBER:
		return &numberNode{value: tok.num, pos: tok.pos}, nil
	case IDENT:
		return &identNode{name: tok.str, pos: tok.pos}, nil
	case EOF:
		return nil, errorAt(tok.pos, ErrorInvalidExpression, "empty expression")
	}

	op, ok := prefixOps[tok.typ]
	if !ok {
		return nil, errorAt(tok.pos, ErrorUnexpectedToken, "unexpected token: %s", tok.str)
	}
	right, err := p.parseExpression(op.rightBp)
	if err != nil {
		return nil, err
	}
	if err := p.consumeExpectedTokens(tok, op); err != nil {
		return nil, err
	}
	return op.build(tok, nil, right), nil
}

func (p *parser) consumeExpectedTokens(start Token, op Op) error {
	for _, expected := range op.ExpectTokens {
		if got := p.peek(); got.typ != expected {
			return expectedTokenError(start, expected)
		}
		p.next()
	}
	return nil
}

func expectedTokenError(start Token, expected TokenType) error {
	switch expected {
	case RPAREN:
		return errorAt(start.pos, ErrorMismatchedParentheses, "mismatched parentheses")
	case PIPE:
		return errorAt(start.pos, ErrorMismatchedParentheses, "mismatched absolute value")
	default:
		return errorAt(start.pos, ErrorUnexpectedToken, "unexpected token: %s", start.str)
	}
}

func (p *parser) infixOp(tok Token) (Op, bool) {
	if tok.typ == MOD && !canStartExpression(p.peekN(1).typ) {
		return percentPostfixOp, true
	}
	op, ok := infixOps[tok.typ]
	return op, ok
}

func (p *parser) peek() Token {
	return p.peekN(0)
}

func (p *parser) peekN(n int) Token {
	idx := p.pos + n
	if idx >= len(p.tokens) {
		return Token{typ: EOF}
	}
	return p.tokens[idx]
}

func (p *parser) next() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func canStartExpression(typ TokenType) bool {
	switch typ {
	case NUMBER, IDENT:
		return true
	default:
		_, ok := prefixOps[typ]
		return ok
	}
}
