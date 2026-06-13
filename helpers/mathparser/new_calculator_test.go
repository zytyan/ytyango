package mathparser

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// Entry point
func TestTokenize(t *testing.T) {
	as := require.New(t)
	expr := "（1.08＋0.2）×pi"
	tokens, err := tokenize(expr)
	as.NoError(err)
	as.Equal(LPAREN, tokens[0].typ)
	as.Equal(NUMBER, tokens[1].typ)
	as.Equal(big.NewRat(108, 100), tokens[1].num)
	as.Equal(PLUS, tokens[2].typ)
	as.Equal(NUMBER, tokens[3].typ)
	as.Equal(RPAREN, tokens[4].typ)
	as.Equal(MUL, tokens[5].typ)
	as.Equal(IDENT, tokens[6].typ)
}

func TestTokenizeLongestMatchAndNewOperators(t *testing.T) {
	as := require.New(t)

	toks, err := tokenize("2 ** 3 // 2 √9 |1-5|")
	as.NoError(err)
	types := []TokenType{NUMBER, POW, NUMBER, FLOORDIV, NUMBER, SQRT, NUMBER, PIPE, NUMBER, MINUS, NUMBER, PIPE, EOF}
	for i, tp := range types {
		as.Equal(tp, toks[i].typ)
	}
}

func TestParse(t *testing.T) {
	as := require.New(t)

	toks, err := tokenize("(1+2)*3")
	as.NoError(err)
	ast, err := parse(toks)
	as.NoError(err)
	res, err := ast.eval()
	as.NoError(err)
	as.Equal(big.NewRat(9, 1), res)
}

func TestEvaluate(t *testing.T) {
	as := require.New(t)

	expr := "(1 + 2) * pi ** 2 // 1 % 2"
	res, err := Evaluate(expr)
	as.NoError(err)
	// Compute expected manually: pi^2 ~ 9.8696, floor div 1 gives 9, mod2 => 1
	as.Equal(big.NewRat(1, 1), res)

	expr = "0.1 + 0.2"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(3, 10), res)

	expr = "2 ** 2049"
	res, err = Evaluate(expr)
	as.NoError(err)
	expResult, ok := new(big.Rat).SetString("6463401214262201460142975337733990392088820533943096806426069085504931" +
		"0277735781786394402823045826927377435921843796038988239118300981842190176304772896566241261754734601992183500" +
		"3955007793042135921152767681351365535844372852395123236761886769523409411632917040726100857751517830821316172" +
		"1510479824786077104382866677933668484136994957312913898971235207065264411615561131866205238541692062830051718" +
		"5728354233451887207436923714715196702304603291808807395226466574462454251369421640419450314203453862646939357" +
		"085161313395870091994536705997276431050332778874671087204270866459209290636957209904296387111707222119192461312")
	as.True(ok)
	as.Equal(expResult, res)

	expr = "2 ** 9999"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(10000, res.Num().BitLen())

	expr = "1.02 ** 2"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(10404, 10000), res)

	expr = "4 ^ 0.5"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(2, 1), res)

	expr = "0.25 ^ 0.5"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(1, 2), res)

	expr = "pi * 3"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(942477, 100000).Cmp(res), -1)
	as.Equal(big.NewRat(942480, 100000).Cmp(res), 1)

	expr = "5!"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(120, 1), res)

	expr = "10P3"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(720, 1), res)

	expr = "10C3"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(120, 1), res)

	expr = "5！"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(120, 1), res)

	expr = "10Ａ3"
	res, err = Evaluate(expr)
	as.NoError(err)
	as.Equal(big.NewRat(720, 1), res)
}

func TestEvaluateUnaryAndPrecedence(t *testing.T) {
	as := require.New(t)

	tests := map[string]*big.Rat{
		"-1+2":    big.NewRat(1, 1),
		"2*-3":    big.NewRat(-6, 1),
		"-(1+2)":  big.NewRat(-3, 1),
		"+2":      big.NewRat(2, 1),
		"2^3^2":   big.NewRat(512, 1),
		"(2^3)^2": big.NewRat(64, 1),
		"-2^2":    big.NewRat(-4, 1),
	}
	for expr, expected := range tests {
		res, err := Evaluate(expr)
		as.NoError(err, expr)
		as.Equal(expected, res, expr)
	}
}

func TestEvaluateNewOperators(t *testing.T) {
	as := require.New(t)

	tests := map[string]*big.Rat{
		"50%":      big.NewRat(1, 2),
		"100 + 5%": big.NewRat(2001, 20),
		"5 % 2":    big.NewRat(1, 1),
		"√9":       big.NewRat(3, 1),
		"√0.25":    big.NewRat(1, 2),
		"|-3|":     big.NewRat(3, 1),
		"|1-5|":    big.NewRat(4, 1),
	}
	for expr, expected := range tests {
		res, err := Evaluate(expr)
		as.NoError(err, expr)
		as.Equal(expected, res, expr)
	}
}

func TestTokenizeErrors(t *testing.T) {
	as := require.New(t)

	_, err := tokenize("1.2.3")
	as.Error(err)
	ce, ok := err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorInvalidNumber, ce.Typ)
	as.Equal(0, ce.pos)

	_, err = tokenize("1$2")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorUnknownCharacter, ce.Typ)
	as.Equal(1, ce.pos)
}

func TestEvaluateErrors(t *testing.T) {
	as := require.New(t)

	_, err := Evaluate("1+(2")
	as.Error(err)
	ce, ok := err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorMismatchedParentheses, ce.Typ)

	_, err = Evaluate("foo")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorUnknownIdentifier, ce.Typ)

	_, err = Evaluate("1/0")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorDivisionByZero, ce.Typ)

	_, err = Evaluate("1.1 % 2")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorModuloRequiresInt, ce.Typ)

	_, err = Evaluate("5 // 0")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorDivisionByZero, ce.Typ)

	_, err = Evaluate("3 % 0")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorModByZero, ce.Typ)

	_, err = Evaluate("3.5!")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorFactorialRequiresInt, ce.Typ)

	_, err = Evaluate("10P12")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorInvalidPermutation, ce.Typ)

	_, err = Evaluate("10C11")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorInvalidCombination, ce.Typ)

	_, err = Evaluate("10 ^ 5000")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorResultTooBig, ce.Typ)

	_, err = Evaluate("2 ^ 10000")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorResultTooBig, ce.Typ)

	_, err = Evaluate("1200!")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorResultTooBig, ce.Typ)

	_, err = Evaluate("√-1")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorInfiniteResult, ce.Typ)

	_, err = Evaluate("|1+2")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorMismatchedParentheses, ce.Typ)

	_, err = Evaluate("1 2")
	as.Error(err)
	ce, ok = err.(*CalcError)
	as.True(ok)
	as.Equal(ErrorUnexpectedToken, ce.Typ)
}

func TestFastCheck(t *testing.T) {
	as := require.New(t)

	as.False(FastCheck("1234567890"))
	as.False(FastCheck("154.54"))
	as.True(FastCheck("1 + 2 * 3"))
	as.False(FastCheck("hello=1+2"))
	as.True(FastCheck("3.14 ×（2＋1）"))
	as.True(FastCheck("√9 + |1-5|"))
}
