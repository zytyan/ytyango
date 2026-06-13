package mathparser

import "math/big"

// Token types used by the tokenizer and Pratt parser.
type TokenType int

const (
	EOF TokenType = iota
	NUMBER
	IDENT
	PLUS
	MINUS
	MUL
	DIV
	POW
	LPAREN
	RPAREN
	FLOORDIV
	MOD
	FACT
	PERM
	COMB
	SQRT
	PIPE
)

type Token struct {
	typ TokenType
	num *big.Rat
	str string
	pos int
}
