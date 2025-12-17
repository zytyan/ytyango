package mathparser

// LL(1) Calculator with big.Rat support, Unicode tokens, and structured tokens

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
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

const (
	maxPooledSliceCap = 256
	maxFastIntDigits  = 18
	maxFastFracDigits = 9
)

var (
	tokenPool = sync.Pool{
		New: func() any {
			return make([]Token, 0, 16)
		},
	}
	rpnPool = sync.Pool{
		New: func() any {
			return make([]Token, 0, 16)
		},
	}
	ratPool = sync.Pool{
		New: func() any {
			return make([]big.Rat, 0, 16)
		},
	}
	pow10Int64 = [...]int64{
		1,
		10,
		100,
		1000,
		10000,
		100000,
		1000000,
		10000000,
		100000000,
		1000000000,
	}
)

func getTokenSlice(need int) []Token {
	buf := tokenPool.Get().([]Token)
	if cap(buf) < need {
		return make([]Token, 0, need)
	}
	return buf[:0]
}

func putTokenSlice(buf []Token) {
	if cap(buf) > maxPooledSliceCap {
		return
	}
	tokenPool.Put(buf[:0])
}

func getRpnSlice(need int) []Token {
	buf := rpnPool.Get().([]Token)
	if cap(buf) < need {
		return make([]Token, 0, need)
	}
	return buf[:0]
}

func putRpnSlice(buf []Token) {
	if cap(buf) > maxPooledSliceCap {
		return
	}
	rpnPool.Put(buf[:0])
}

func getRatSlice(need int) []big.Rat {
	buf := ratPool.Get().([]big.Rat)
	if cap(buf) < need {
		return make([]big.Rat, need)
	}
	return buf[:need]
}

func putRatSlice(buf []big.Rat) {
	if cap(buf) > maxPooledSliceCap {
		return
	}
	for i := range buf {
		buf[i].SetInt64(0)
	}
	ratPool.Put(buf[:0])
}

func fastParseRat(raw string, dst *big.Rat) bool {
	if raw == "" {
		return false
	}
	dot := strings.IndexByte(raw, '.')
	if dot == -1 {
		if len(raw) > maxFastIntDigits {
			return false
		}
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			dst.SetInt64(v)
			return true
		}
		return false
	}
	intPart := raw[:dot]
	fracPart := raw[dot+1:]
	if fracPart == "" || len(fracPart) > maxFastFracDigits || len(intPart) > maxFastFracDigits {
		return false
	}
	intVal := int64(0)
	if intPart != "" {
		v, err := strconv.ParseInt(intPart, 10, 64)
		if err != nil {
			return false
		}
		intVal = v
	}
	fracVal, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return false
	}
	denom := pow10Int64[len(fracPart)]
	num := fracVal + intVal*denom
	dst.SetFrac64(num, denom)
	return true
}

func needsReplace(s string) bool {
	for _, r := range s {
		switch r {
		case '（', '）', '＋', '－', '×', '＊', '÷', '／', '！', 'Ａ', 'ａ', 'Ｃ', 'ｃ', 'Ｐ', 'ｐ':
			return true
		}
	}
	return false
}

// tokenize converts input string to slice of Tokens

func tokenize(input string) ([]Token, error) {
	if needsReplace(input) {
		input = replacer.Replace(input)
	}
	tokens := getTokenSlice(len(input) + 1)
	bytePos := 0
	runePos := 0

	for bytePos < len(input) {
		r, size := utf8.DecodeRuneInString(input[bytePos:])
		switch {
		case unicode.IsSpace(r):
			bytePos += size
			runePos++
		case unicode.IsDigit(r) || r == '.':
			startByte := bytePos
			startRune := runePos
			bytePos += size
			runePos++
			for bytePos < len(input) {
				nextRune, nextSize := utf8.DecodeRuneInString(input[bytePos:])
				if unicode.IsDigit(nextRune) || nextRune == '.' {
					bytePos += nextSize
					runePos++
					continue
				}
				break
			}
			raw := input[startByte:bytePos]
			idx := len(tokens)
			var rat *big.Rat
			if idx < cap(tokens) {
				rat = tokens[:cap(tokens)][idx].num
			}
			if rat == nil {
				rat = new(big.Rat)
			}
			if fastParseRat(raw, rat) {
				tokens = append(tokens, Token{typ: NUMBER, num: rat})
				continue
			}
			if _, ok := rat.SetString(raw); !ok {
				return nil, errorAt(startRune, ErrorInvalidNumber, "invalid number: %s", raw)
			}
			tokens = append(tokens, Token{typ: NUMBER, num: rat})
		case unicode.IsLetter(r):
			startByte := bytePos
			bytePos += size
			runePos++
			for bytePos < len(input) {
				nextRune, nextSize := utf8.DecodeRuneInString(input[bytePos:])
				if unicode.IsLetter(nextRune) {
					bytePos += nextSize
					runePos++
					continue
				}
				break
			}
			name := input[startByte:bytePos]
			if size == len(name) && r <= unicode.MaxASCII {
				switch toLowerASCII(byte(r)) {
				case 'a', 'p':
					tokens = append(tokens, Token{typ: PERM, str: name})
				case 'c':
					tokens = append(tokens, Token{typ: COMB, str: name})
				default:
					tokens = append(tokens, Token{typ: IDENT, str: name})
				}
			} else {
				tokens = append(tokens, Token{typ: IDENT, str: name})
			}
		default:
			if r == '*' && bytePos+1 < len(input) && input[bytePos+1] == '*' {
				tokens = append(tokens, Token{typ: POW, str: "**"})
				bytePos += 2
				runePos += 2
				continue
			}
			if r == '/' && bytePos+1 < len(input) && input[bytePos+1] == '/' {
				tokens = append(tokens, Token{typ: FLOORDIV, str: "//"})
				bytePos += 2
				runePos += 2
				continue
			}
			switch r {
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
				return nil, errorAt(runePos, ErrorUnknownCharacter, "unknown character: %q at %d", r, runePos)
			}
			bytePos += size
			runePos++
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
	output := getRpnSlice(len(tokens))
	ops := getTokenSlice(len(tokens))
	defer putTokenSlice(ops)
	fail := func(err error) ([]Token, error) {
		putRpnSlice(output)
		return nil, err
	}

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
				return fail(errorAt(-1, ErrorMismatchedParentheses, "mismatched parentheses"))
			}
		case EOF:
			// do nothing
		default:
			return fail(errorAt(-1, ErrorUnexpectedToken, "unexpected token in shunting yard: %v", tok))
		}
	}
	for len(ops) > 0 {
		top := ops[len(ops)-1]
		ops = ops[:len(ops)-1]
		if top.typ == LPAREN || top.typ == RPAREN {
			return fail(errorAt(-1, ErrorMismatchedParentheses, "mismatched parentheses"))
		}
		output = append(output, top)
	}
	return output, nil
}

type ratStack struct {
	data []big.Rat
	top  int
}

func newRatStack(size int) ratStack {
	return ratStack{
		data: getRatSlice(size),
	}
}

func (s *ratStack) push(src *big.Rat) {
	s.data[s.top].Set(src)
	s.top++
}

func (s *ratStack) pop() (*big.Rat, error) {
	if s.top == 0 {
		return nil, errorAt(-1, ErrorStackUnderflow, "stack underflow")
	}
	s.top--
	return &s.data[s.top], nil
}

func (s *ratStack) popBinary() (*big.Rat, *big.Rat, error) {
	if s.top < 2 {
		return nil, nil, errorAt(-1, ErrorStackUnderflow, "stack underflow")
	}
	s.top--
	right := &s.data[s.top]
	s.top--
	left := &s.data[s.top]
	return left, right, nil
}

func (s *ratStack) release() {
	putRatSlice(s.data)
	s.data = nil
	s.top = 0
}

// evalRPN evaluates a postfix token list and returns big.Rat result

func evalRPN(rpn []Token) (*big.Rat, error) {
	stack := newRatStack(len(rpn))
	defer stack.release()
	var tmpInt1, tmpInt2, tmpInt3 big.Int
	var tmpRat big.Rat

	for _, tok := range rpn {
		switch tok.typ {
		case NUMBER:
			stack.push(tok.num)
		case IDENT:
			switch {
			case equalFoldASCII(tok.str, "pi"):
				stack.push(pi)
			case equalFoldASCII(tok.str, "e"):
				stack.push(e)
			default:
				return nil, errorAt(-1, ErrorUnknownIdentifier, "unknown identifier: %s", tok.str)
			}
		case PLUS, MINUS, MUL, DIV, POW, FLOORDIV, MOD, PERM, COMB:
			left, right, err := stack.popBinary()
			if err != nil {
				return nil, err
			}
			switch tok.typ {
			case PLUS:
				left.Add(left, right)
			case MINUS:
				left.Sub(left, right)
			case MUL:
				left.Mul(left, right)
			case DIV:
				if right.Sign() == 0 {
					return nil, errorAt(-1, ErrorDivisionByZero, "division by zero")
				}
				left.Quo(left, right)
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
					left.SetFloat64(floatRet)
					break
				}
				if math.Log10(base)*exp > 3000 {
					return nil, errorAt(-1, ErrorResultTooBig, "result too big")
				}
				tmpRat.Set(left)
				powRat(left, &tmpRat, int(exp))
			case FLOORDIV:
				// floor division: (a/b) // (c/d) = floor((a*d)/(b*c))
				tmpInt1.Mul(left.Num(), right.Denom())
				tmpInt2.Mul(left.Denom(), right.Num())
				if tmpInt2.Sign() == 0 {
					return nil, errorAt(-1, ErrorDivisionByZero, "floor division by zero")
				}
				tmpInt3.Quo(&tmpInt1, &tmpInt2)
				left.SetInt(&tmpInt3)
			case MOD:
				if !left.IsInt() || !right.IsInt() {
					return nil, errorAt(-1, ErrorModuloRequiresInt, "modulo requires integers")
				}
				if right.Num().Sign() == 0 {
					return nil, errorAt(-1, ErrorModByZero, "mod by zero")
				}
				tmpInt1.Mod(left.Num(), right.Num())
				left.SetInt(&tmpInt1)
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
				left.SetInt(permInt(&tmpInt1, n, r))
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
				left.SetInt(combInt(&tmpInt1, n, r))
			default:
				return nil, errorAt(-1, ErrorUnexpectedToken, "unexpected token in shunting yard: %v", tok)
			}
			stack.top++
		case FACT:
			val, err := stack.pop()
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
			factorialInt(&tmpInt1, n)
			val.SetInt(&tmpInt1)
			stack.top++
		case LPAREN, RPAREN, EOF:
		default:
			return nil, errorAt(-1, ErrorUnexpectedToken, "unexpected token in eval: %v", tok)
		}
	}
	if stack.top != 1 {
		return nil, errorAt(-1, ErrorInvalidExpression, "invalid expression, stack has %d elements", stack.top)
	}
	result := new(big.Rat).Set(&stack.data[0])
	return result, nil
}

// powRat computes integer power for *big.Rat base and int exponent.
// dst is used as accumulator to reduce allocations.
func powRat(dst *big.Rat, base *big.Rat, exp int) *big.Rat {
	dst.SetInt64(1)
	var tmp big.Rat
	tmp.Set(base)
	for exp > 0 {
		if exp&1 == 1 {
			dst.Mul(dst, &tmp)
		}
		tmp.Mul(&tmp, &tmp)
		exp >>= 1
	}
	return dst
}

func factorialInt(dst *big.Int, n int64) *big.Int {
	dst.SetInt64(1)
	var step big.Int
	for i := int64(2); i <= n; i++ {
		step.SetInt64(i)
		dst.Mul(dst, &step)
	}
	return dst
}

func approxFactorialDigits(n int64) float64 {
	lg, _ := math.Lgamma(float64(n) + 1)
	return lg / math.Ln10
}

func permInt(dst *big.Int, n, r int64) *big.Int {
	dst.SetInt64(1)
	var step big.Int
	for i := int64(0); i < r; i++ {
		step.SetInt64(n - i)
		dst.Mul(dst, &step)
	}
	return dst
}

func approxPermDigits(n, r int64) float64 {
	lg1, _ := math.Lgamma(float64(n) + 1)
	lg2, _ := math.Lgamma(float64(n-r) + 1)
	return (lg1 - lg2) / math.Ln10
}

func combInt(dst *big.Int, n, r int64) *big.Int {
	if r > n-r {
		r = n - r
	}
	dst.SetInt64(1)
	var step big.Int
	for i := int64(1); i <= r; i++ {
		step.SetInt64(n - r + i)
		dst.Mul(dst, &step)
		step.SetInt64(i)
		dst.Div(dst, &step)
	}
	return dst
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
	defer putTokenSlice(tokens)
	rpn, err := shuntingYard(tokens)
	if err != nil {
		return nil, err
	}
	defer putRpnSlice(rpn)
	return evalRPN(rpn)
}

func toLowerASCII(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if toLowerASCII(a[i]) != toLowerASCII(b[i]) {
			return false
		}
	}
	return true
}

func isAllowedFastCheckRune(r rune) bool {
	switch r {
	case ' ', '(', ')', '（', '）', '+', '＋', '-', '－', '×', '*', '＊', '÷', '/', '／', '.', '!', '^':
		return true
	}
	if unicode.IsNumber(r) {
		return true
	}
	if r <= unicode.MaxASCII {
		switch toLowerASCII(byte(r)) {
		case 'e', 'p', 'i', 'a', 'c':
			return true
		}
	}
	return false
}

func FastCheck(expr string) bool {
	onlyDigits := true
	for _, c := range expr {
		if unicode.IsNumber(c) {
			continue
		}
		onlyDigits = false
		if !isAllowedFastCheckRune(c) {
			return false
		}
	}
	return !onlyDigits
}
