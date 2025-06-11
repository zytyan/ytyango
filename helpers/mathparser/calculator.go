package mathparser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"math/big"
	"strings"
)

type ListenerCalc struct {
	BaseCalcListener
	vm  *VmBuilder
	err error
}

func (m *ListenerCalc) ExitPow(*PowContext) {
	m.vm.addNoValOp(UnaryPow)
}

func (m *ListenerCalc) ExitMulDivMod(c *MulDivModContext) {
	if c.MUL() != nil {
		m.vm.addNoValOp(UnaryMul)
	} else if c.DIV() != nil {
		m.vm.addNoValOp(UnaryDiv)
	} else if c.MOD() != nil {
		m.vm.addNoValOp(UnaryMod)
	}
}

func (m *ListenerCalc) ExitAddSub(c *AddSubContext) {
	if c.ADD() != nil {
		m.vm.addNoValOp(UnaryAdd)
	} else if c.SUB() != nil {
		m.vm.addNoValOp(UnarySub)
	}
}

func (m *ListenerCalc) ExitNum(c *NumContext) {
	text := c.GetText()
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ToLower(text)
	if len(text) > 100 {
		m.err = fmt.Errorf("number too long")
		return
	}
	f, _ := new(big.Rat).SetString(text)
	m.vm.addConst(&Value{valType: typeNumber, Num: &number{val: f}})
}

var pi, _ = new(big.Rat).SetString("3.141592653589793238462643383279502884197169399375105820974944592307816406286")
var e, _ = new(big.Rat).SetString("2.718281828459045235360287471352662497757247093699959574966967627724076630353")

func (m *ListenerCalc) ExitId(c *IdContext) {
	t := c.GetText()
	switch t {
	case "pi":
		m.vm.addConst(&Value{valType: typeNumber, Num: &number{val: pi}})
	case "e":
		m.vm.addConst(&Value{valType: typeNumber, Num: &number{val: e}})
	default:
		m.err = fmt.Errorf("unknown id: %s", t)
	}
}
func parseHex(hex string) (*big.Rat, error) {
	strings.ToLower(hex)
	neg := false
	if strings.HasPrefix(hex, "-") {
		neg = true
		hex = hex[1:]
	}
	if len(hex) > 100 {
		return big.NewRat(0, 1), fmt.Errorf("number too long")
	}
	num, ok := new(big.Int).SetString(hex, 16)
	if !ok {
		return big.NewRat(0, 1), fmt.Errorf("invalid hex number")
	}
	r := new(big.Rat).SetInt(num)
	if !ok {
		return r, fmt.Errorf("invalid hex number")
	}
	if neg {
		r.Neg(r)
	}
	return r, nil

}
func (m *ListenerCalc) ExitHex(c *HexContext) {
	text := c.GetText()
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ToLower(text)[2:]
	if len(text) > 100 {
		m.err = fmt.Errorf("number too long")
		return
	}

	i, err := parseHex(text)
	if err != nil {
		m.err = err
	}
	m.vm.addConst(&Value{valType: typeNumber, Num: &number{val: i}})
}

type StringErrorListener struct {
	*antlr.DefaultErrorListener
	err string
}

func (s *StringErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	s.err = fmt.Sprintf("line %d:%d %s", line, column, msg)
}
func (s *StringErrorListener) HasError() bool {
	return s.err != ""
}
func (s *StringErrorListener) Error() string {
	return s.err
}
func ParseString(data string) (*Value, error) {
	input := antlr.NewInputStream(data)
	lexerListener := &StringErrorListener{}
	lexer := NewCalcLexer(input)
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerListener)

	parserListener := &StringErrorListener{}
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := NewCalcParser(stream)
	p.BuildParseTrees = true
	p.RemoveErrorListeners()
	p.AddErrorListener(parserListener)
	tree := p.Prog()
	visitor := ListenerCalc{vm: &VmBuilder{}}
	antlr.ParseTreeWalkerDefault.Walk(&visitor, tree)
	if parserListener.HasError() || lexerListener.HasError() || visitor.err != nil {
		var err error = parserListener
		if lexerListener.HasError() {
			err = lexerListener
		}
		if visitor.err != nil {
			err = visitor.err
		}
		return nil, err
	}
	vm := visitor.vm.Build()
	return vm.Run()
}
