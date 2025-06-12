package mathparser

// LL(1) Calculator with big.Rat support, Unicode tokens, and structured tokens

import (
	"math"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

// Token types
// EOF terminates input
// NUMBER for numeric values
// IDENT for identifiers (e.g., pi, e)
// PLUS, MINUS, MUL, DIV, POW, FLOORDIV, MOD for operators
// LPAREN, RPAREN for parentheses

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
)

// Token holds value or symbol and numeric literal

type Token struct {
	typ TokenType
	num *big.Rat // used if typ == NUMBER
	str string   // used if typ != NUMBER
}

// replacer handles Unicode variants

var replacer = strings.NewReplacer(
	"（", "(",
	"）", ")",
	"＋", "+", "－", "-",
	"×", "*", "＊", "*",
	"÷", "/", "／", "/",
	"！", "!",
	"Ａ", "A", "ａ", "a",
	"Ｃ", "C", "ｃ", "c",
	"Ｐ", "P", "ｐ", "p",
)

var e, _ = new(big.Rat).SetString(`2.718281828459`)
var pi, _ = new(big.Rat).SetString(`3.141592653589793`)

// tokenize converts input string to slice of Tokens

func tokenize(input string) ([]Token, error) {
	input = replacer.Replace(input)
	runes := []rune(input)
	i := 0
	var tokens []Token

	for i < len(runes) {
		switch c := runes[i]; {
		case unicode.IsSpace(c):
			i++
		case unicode.IsDigit(c) || c == '.':
			start := i
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			raw := string(runes[start:i])
			r := new(big.Rat)
			if _, ok := r.SetString(raw); !ok {
				return nil, errorAt(start, ErrorInvalidNumber, "invalid number: %s", raw)
			}
			tokens = append(tokens, Token{typ: NUMBER, num: r})
		case unicode.IsLetter(c):
			start := i
			for i < len(runes) && unicode.IsLetter(runes[i]) {
				i++
			}
			name := string(runes[start:i])
			if len(name) == 1 {
				switch strings.ToLower(name) {
				case "a", "p":
					tokens = append(tokens, Token{typ: PERM, str: name})
				case "c":
					tokens = append(tokens, Token{typ: COMB, str: name})
				default:
					tokens = append(tokens, Token{typ: IDENT, str: name})
				}
			} else {
				tokens = append(tokens, Token{typ: IDENT, str: name})
			}
		default:
			if i+1 < len(runes) {
				two := string(runes[i : i+2])
				switch two {
				case "**":
					tokens = append(tokens, Token{typ: POW, str: two})
					i += 2
					continue
				case "//":
					tokens = append(tokens, Token{typ: FLOORDIV, str: two})
					i += 2
					continue
				}
			}
			switch c {
			case '+':
				tokens = append(tokens, Token{typ: PLUS, str: "+"})
			case '-':
				tokens = append(tokens, Token{typ: MINUS, str: "-"})
			case '*':
				tokens = append(tokens, Token{typ: MUL, str: "*"})
			case '/':
				tokens = append(tokens, Token{typ: DIV, str: "/"})
			case '%':
				tokens = append(tokens, Token{typ: MOD, str: "%"})
			case '^':
				tokens = append(tokens, Token{typ: POW, str: "^"})
			case '!':
				tokens = append(tokens, Token{typ: FACT, str: "!"})
			case '(':
				tokens = append(tokens, Token{typ: LPAREN, str: "("})
			case ')':
				tokens = append(tokens, Token{typ: RPAREN, str: ")"})
			default:
				return nil, errorAt(i, ErrorUnknownCharacter, "unknown character: %q at %d", c, i)
			}
			i++
		}
	}
	tokens = append(tokens, Token{typ: EOF})
	return tokens, nil
}

// precedence and associativity definitions

func precedence(tok Token) int {
	switch tok.typ {
	case PLUS, MINUS:
		return 1
	case MUL, DIV, FLOORDIV, MOD:
		return 2
	case POW:
		return 3
	case FACT, PERM, COMB:
		return 4
	default:
		return 0
	}
}

func isRightAssociative(tok Token) bool {
	return tok.typ == POW
}

// shuntingYard converts infix tokens to postfix (RPN)

func shuntingYard(tokens []Token) ([]Token, error) {
	var output []Token
	var ops []Token

	for _, tok := range tokens {
		switch tok.typ {
		case NUMBER, IDENT:
			output = append(output, tok)
		case PLUS, MINUS, MUL, DIV, POW, FLOORDIV, MOD, FACT, PERM, COMB:
			for len(ops) > 0 {
				top := ops[len(ops)-1]
				if (top.typ != LPAREN) && (precedence(top) > precedence(tok) || (precedence(top) == precedence(tok) && !isRightAssociative(tok))) {
					output = append(output, top)
					ops = ops[:len(ops)-1]
					continue
				}
				break
			}
			ops = append(ops, tok)
		case LPAREN:
			ops = append(ops, tok)
		case RPAREN:
			found := false
			for len(ops) > 0 {
				top := ops[len(ops)-1]
				ops = ops[:len(ops)-1]
				if top.typ == LPAREN {
					found = true
					break
				}
				output = append(output, top)
			}
			if !found {
				return nil, errorAt(-1, ErrorMismatchedParentheses, "mismatched parentheses")
			}
		case EOF:
			break
		default:
			return nil, errorAt(-1, ErrorUnexpectedToken, "unexpected token in shunting yard: %v", tok)
		}
	}
	for len(ops) > 0 {
		top := ops[len(ops)-1]
		ops = ops[:len(ops)-1]
		if top.typ == LPAREN || top.typ == RPAREN {
			return nil, errorAt(-1, ErrorMismatchedParentheses, "mismatched parentheses")
		}
		output = append(output, top)
	}
	return output, nil
}

// evalRPN evaluates a postfix token list and returns big.Rat result

func evalRPN(rpn []Token) (*big.Rat, error) {
	var stack []*big.Rat

	push := func(r *big.Rat) {
		stack = append(stack, r)
	}
	pop := func() (*big.Rat, error) {
		if len(stack) < 1 {
			return nil, errorAt(-1, ErrorStackUnderflow, "stack underflow")
		}
		r := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return r, nil
	}

	for _, tok := range rpn {
		switch tok.typ {
		case NUMBER:
			push(new(big.Rat).Set(tok.num))
		case IDENT:
			var r *big.Rat
			switch strings.ToLower(tok.str) {
			case "pi":
				r = pi
			case "e":
				r = e
			default:
				return nil, errorAt(-1, ErrorUnknownIdentifier, "unknown identifier: %s", tok.str)
			}
			push(r)
		case PLUS, MINUS, MUL, DIV, POW, FLOORDIV, MOD, PERM, COMB:
			// binary ops: pop right then left
			right, err := pop()
			if err != nil {
				return nil, err
			}
			left, err := pop()
			if err != nil {
				return nil, err
			}
			var res *big.Rat
			switch tok.typ {
			case PLUS:
				res = new(big.Rat).Add(left, right)
			case MINUS:
				res = new(big.Rat).Sub(left, right)
			case MUL:
				res = new(big.Rat).Mul(left, right)
			case DIV:
				if right.Cmp(new(big.Rat)) == 0 {
					return nil, errorAt(-1, ErrorDivisionByZero, "division by zero")
				}
				res = new(big.Rat).Quo(left, right)
			case POW:
				// exponent must be integer
				exp, _ := right.Float64()
				base, _ := left.Float64()
				if math.IsInf(exp, 0) {
					return nil, errorAt(-1, ErrorInfiniteResult, "infinite exponent")
				}
				if math.IsInf(base, 0) {
					return nil, errorAt(-1, ErrorInfiniteResult, "infinite base number")
				}
				if !right.IsInt() {
					floatRet := math.Pow(base, exp)
					if math.IsInf(floatRet, 0) || math.IsNaN(floatRet) {
						return nil, errorAt(-1, ErrorInfiniteResult, "infinite float")
					}
					res = new(big.Rat).SetFloat64(floatRet)
					break
				}
				if math.Log10(base)*exp > 3000 {
					return nil, errorAt(-1, ErrorResultTooBig, "result too big")
				}
				res = powRat(left, int(exp))
			case FLOORDIV:
				// floor division: (a/b) // (c/d) = floor((a*d)/(b*c))
				n := new(big.Int).Mul(left.Num(), right.Denom())
				d := new(big.Int).Mul(left.Denom(), right.Num())
				if d.Sign() == 0 {
					return nil, errorAt(-1, ErrorDivisionByZero, "floor division by zero")
				}
				quo := new(big.Int).Div(n, d)
				res = new(big.Rat).SetInt(quo)
			case MOD:
				if !left.IsInt() || !right.IsInt() {
					return nil, errorAt(-1, ErrorModuloRequiresInt, "modulo requires integers")
				}
				a := left.Num()
				b := right.Num()
				if b.Sign() == 0 {
					return nil, errorAt(-1, ErrorModByZero, "mod by zero")
				}
				m := new(big.Int).Mod(a, b)
				res = new(big.Rat).SetInt(m)
			case PERM:
				if !left.IsInt() || !right.IsInt() {
					return nil, errorAt(-1, ErrorPermutationRequiresInt, "permutation requires integers")
				}
				n := left.Num().Int64()
				r := right.Num().Int64()
				if n < 0 || r < 0 || r > n {
					return nil, errorAt(-1, ErrorInvalidPermutation, "invalid permutation")
				}
				if approxPermDigits(n, r) > 8000 {
					return nil, errorAt(-1, ErrorResultTooBig, "result too big")
				}
				resInt := permInt(n, r)
				res = new(big.Rat).SetInt(resInt)
			case COMB:
				if !left.IsInt() || !right.IsInt() {
					return nil, errorAt(-1, ErrorCombinationRequiresInt, "combination requires integers")
				}
				n := left.Num().Int64()
				r := right.Num().Int64()
				if n < 0 || r < 0 || r > n {
					return nil, errorAt(-1, ErrorInvalidCombination, "invalid combination")
				}
				if approxCombDigits(n, r) > 8000 {
					return nil, errorAt(-1, ErrorResultTooBig, "result too big")
				}
				resInt := combInt(n, r)
				res = new(big.Rat).SetInt(resInt)
			default:
				return nil, errorAt(-1, ErrorUnexpectedToken, "unexpected token in shunting yard: %v", tok)
			}
			push(res)
		case FACT:
			val, err := pop()
			if err != nil {
				return nil, err
			}
			if !val.IsInt() {
				return nil, errorAt(-1, ErrorFactorialRequiresInt, "factorial requires integer")
			}
			n := val.Num().Int64()
			if n < 0 {
				return nil, errorAt(-1, ErrorFactorialNegative, "factorial of negative number")
			}
			if approxFactorialDigits(n) > 8000 {
				return nil, errorAt(-1, ErrorResultTooBig, "result too big")
			}
			resInt := factorialInt(n)
			push(new(big.Rat).SetInt(resInt))
		case LPAREN, RPAREN, EOF:
		// ignore
		default:
			return nil, errorAt(-1, ErrorUnexpectedToken, "unexpected token in eval: %v", tok)
		}
	}
	if len(stack) != 1 {
		return nil, errorAt(-1, ErrorInvalidExpression, "invalid expression, stack has %d elements", len(stack))
	}
	return stack[0], nil
}

// powRat computes integer power for *big.Rat base and int exponent

func powRat(base *big.Rat, exp int) *big.Rat {
	res := big.NewRat(1, 1)
	tmp := new(big.Rat).Set(base)
	for exp > 0 {
		if exp&1 == 1 {
			res.Mul(res, tmp)
		}
		tmp.Mul(tmp, tmp)
		exp >>= 1
	}
	return res
}

func factorialInt(n int64) *big.Int {
	res := big.NewInt(1)
	for i := int64(2); i <= n; i++ {
		res.Mul(res, big.NewInt(i))
	}
	return res
}

func approxFactorialDigits(n int64) float64 {
	lg, _ := math.Lgamma(float64(n) + 1)
	return lg / math.Ln10
}

func permInt(n, r int64) *big.Int {
	res := big.NewInt(1)
	for i := int64(0); i < r; i++ {
		res.Mul(res, big.NewInt(n-i))
	}
	return res
}

func approxPermDigits(n, r int64) float64 {
	lg1, _ := math.Lgamma(float64(n) + 1)
	lg2, _ := math.Lgamma(float64(n-r) + 1)
	return (lg1 - lg2) / math.Ln10
}

func combInt(n, r int64) *big.Int {
	if r > n-r {
		r = n - r
	}
	res := big.NewInt(1)
	for i := int64(1); i <= r; i++ {
		res.Mul(res, big.NewInt(n-r+i))
		res.Div(res, big.NewInt(i))
	}
	return res
}

func approxCombDigits(n, r int64) float64 {
	lg1, _ := math.Lgamma(float64(n) + 1)
	lg2, _ := math.Lgamma(float64(r) + 1)
	lg3, _ := math.Lgamma(float64(n-r) + 1)
	return (lg1 - lg2 - lg3) / math.Ln10
}

// Evaluate takes an expression string, tokenizes, parses, and evaluates to big.Rat

func Evaluate(expr string) (*big.Rat, error) {
	tokens, err := tokenize(expr)
	if err != nil {
		return nil, err
	}
	rpn, err := shuntingYard(tokens)
	if err != nil {
		return nil, err
	}
	return evalRPN(rpn)
}

var reFastCheck = regexp.MustCompile(`^[ \depiacpACP（(）)＋+－\-×*＊÷/／.!]+$`)

func FastCheck(expr string) bool {
	for _, c := range expr {
		if !unicode.IsNumber(c) {
			goto check
		}

	}
	return false
check:
	return reFastCheck.MatchString(expr)
}
