package mathparser

import (
	"math/big"
	"regexp"
	"unicode"
)

var e, _ = new(big.Rat).SetString(`2.718281828459`)
var pi, _ = new(big.Rat).SetString(`3.141592653589793`)

// Evaluate takes an expression string, tokenizes, parses, and evaluates to big.Rat.
func Evaluate(expr string) (*big.Rat, error) {
	tokens, err := tokenize(expr)
	if err != nil {
		return nil, err
	}
	ast, err := parse(tokens)
	if err != nil {
		return nil, err
	}
	return ast.eval()
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
	case ' ', '(', ')', '（', '）', '+', '＋', '-', '－', '×', '*', '＊', '÷', '/', '／', '.', '!', '！', '^', '%', '√', '|':
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

var rePlusNum = regexp.MustCompile(`^\+\d+$`)

func FastCheck(expr string) bool {
	onlyDigits := true
	for _, c := range expr {
		if unicode.IsNumber(c) {
			continue
		}
		if c == '.' {
			continue
		}
		onlyDigits = false
		if !isAllowedFastCheckRune(c) {
			return false
		}
	}
	if onlyDigits {
		return false
	}
	return !rePlusNum.MatchString(expr)
}
