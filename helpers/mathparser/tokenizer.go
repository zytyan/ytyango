package mathparser

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxFastIntDigits  = 18
	maxFastFracDigits = 9
)

var (
	numberPattern = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)`)
	identPattern  = regexp.MustCompile(`^\p{L}+`)

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

type symbolMatch struct {
	text string
	typ  TokenType
	str  string
}

type radixNode struct {
	children map[rune]*radixNode
	match    *symbolMatch
}

func newRadixNode() *radixNode {
	return &radixNode{children: make(map[rune]*radixNode)}
}

func (n *radixNode) insert(text string, typ TokenType, str string) {
	cur := n
	for _, r := range text {
		next := cur.children[r]
		if next == nil {
			next = newRadixNode()
			cur.children[r] = next
		}
		cur = next
	}
	cur.match = &symbolMatch{text: text, typ: typ, str: str}
}

func (n *radixNode) longest(input string) (*symbolMatch, int, int) {
	cur := n
	byteLen := 0
	runeLen := 0
	best := cur.match
	bestBytes := 0
	bestRunes := 0
	for byteLen < len(input) {
		r, size := utf8.DecodeRuneInString(input[byteLen:])
		next := cur.children[r]
		if next == nil {
			break
		}
		byteLen += size
		runeLen++
		cur = next
		if cur.match != nil {
			best = cur.match
			bestBytes = byteLen
			bestRunes = runeLen
		}
	}
	return best, bestBytes, bestRunes
}

var operatorTrie = func() *radixNode {
	root := newRadixNode()
	for _, def := range []symbolMatch{
		{text: "+", typ: PLUS, str: "+"},
		{text: "＋", typ: PLUS, str: "+"},
		{text: "-", typ: MINUS, str: "-"},
		{text: "－", typ: MINUS, str: "-"},
		{text: "*", typ: MUL, str: "*"},
		{text: "＊", typ: MUL, str: "*"},
		{text: "×", typ: MUL, str: "*"},
		{text: "/", typ: DIV, str: "/"},
		{text: "／", typ: DIV, str: "/"},
		{text: "÷", typ: DIV, str: "/"},
		{text: "**", typ: POW, str: "**"},
		{text: "^", typ: POW, str: "^"},
		{text: "//", typ: FLOORDIV, str: "//"},
		{text: "%", typ: MOD, str: "%"},
		{text: "!", typ: FACT, str: "!"},
		{text: "！", typ: FACT, str: "!"},
		{text: "(", typ: LPAREN, str: "("},
		{text: "（", typ: LPAREN, str: "("},
		{text: ")", typ: RPAREN, str: ")"},
		{text: "）", typ: RPAREN, str: ")"},
		{text: "√", typ: SQRT, str: "√"},
		{text: "|", typ: PIPE, str: "|"},
		{text: "P", typ: PERM, str: "P"},
		{text: "p", typ: PERM, str: "p"},
		{text: "Ａ", typ: PERM, str: "A"},
		{text: "ａ", typ: PERM, str: "a"},
		{text: "A", typ: PERM, str: "A"},
		{text: "a", typ: PERM, str: "a"},
		{text: "C", typ: COMB, str: "C"},
		{text: "c", typ: COMB, str: "c"},
		{text: "Ｃ", typ: COMB, str: "C"},
		{text: "ｃ", typ: COMB, str: "c"},
	} {
		root.insert(def.text, def.typ, def.str)
	}
	return root
}()

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

func tokenize(input string) ([]Token, error) {
	tokens := make([]Token, 0, len(input)+1)
	bytePos := 0
	runePos := 0

	for bytePos < len(input) {
		r, size := utf8.DecodeRuneInString(input[bytePos:])
		if unicode.IsSpace(r) {
			bytePos += size
			runePos++
			continue
		}

		rest := input[bytePos:]
		if raw := numberPattern.FindString(rest); raw != "" {
			if bytePos+len(raw) < len(input) {
				next, _ := utf8.DecodeRuneInString(input[bytePos+len(raw):])
				if next == '.' {
					return nil, errorAt(runePos, ErrorInvalidNumber, "invalid number: %s", readNumberLike(rest))
				}
			}
			idx := len(tokens)
			var rat *big.Rat
			if idx < cap(tokens) {
				rat = tokens[:cap(tokens)][idx].num
			}
			if rat == nil {
				rat = new(big.Rat)
			}
			if !fastParseRat(raw, rat) {
				if _, ok := rat.SetString(raw); !ok {
					return nil, errorAt(runePos, ErrorInvalidNumber, "invalid number: %s", raw)
				}
			}
			tokens = append(tokens, Token{typ: NUMBER, num: rat, str: raw, pos: runePos})
			bytePos += len(raw)
			runePos += utf8.RuneCountInString(raw)
			continue
		}

		if raw := identPattern.FindString(rest); raw != "" {
			if utf8.RuneCountInString(raw) == 1 {
				if match, _, _ := operatorTrie.longest(raw); match != nil {
					tokens = append(tokens, Token{typ: match.typ, str: match.str, pos: runePos})
					bytePos += len(raw)
					runePos++
					continue
				}
			}
			tokens = append(tokens, Token{typ: IDENT, str: raw, pos: runePos})
			bytePos += len(raw)
			runePos += utf8.RuneCountInString(raw)
			continue
		}

		if match, byteLen, runeLen := operatorTrie.longest(rest); match != nil {
			tokens = append(tokens, Token{typ: match.typ, str: match.str, pos: runePos})
			bytePos += byteLen
			runePos += runeLen
			continue
		}

		return nil, errorAt(runePos, ErrorUnknownCharacter, "unknown character: %q at %d", r, runePos)
	}
	tokens = append(tokens, Token{typ: EOF, pos: runePos})
	return tokens, nil
}

func readNumberLike(input string) string {
	bytePos := 0
	for bytePos < len(input) {
		r, size := utf8.DecodeRuneInString(input[bytePos:])
		if !unicode.IsDigit(r) && r != '.' {
			break
		}
		bytePos += size
	}
	return input[:bytePos]
}
