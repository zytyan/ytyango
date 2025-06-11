// Code generated from C:/Users/manni/GolandProjects/ytyango/helpers/mathparser/Calc.g4 by ANTLR 4.13.1. DO NOT EDIT.

package mathparser // Calc
import "github.com/antlr4-go/antlr/v4"

// BaseCalcListener is a complete listener for a parse tree produced by CalcParser.
type BaseCalcListener struct{}

var _ CalcListener = &BaseCalcListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseCalcListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseCalcListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseCalcListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseCalcListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterProg is called when production prog is entered.
func (s *BaseCalcListener) EnterProg(ctx *ProgContext) {}

// ExitProg is called when production prog is exited.
func (s *BaseCalcListener) ExitProg(ctx *ProgContext) {}

// EnterStat is called when production stat is entered.
func (s *BaseCalcListener) EnterStat(ctx *StatContext) {}

// ExitStat is called when production stat is exited.
func (s *BaseCalcListener) ExitStat(ctx *StatContext) {}

// EnterMulDivMod is called when production MulDivMod is entered.
func (s *BaseCalcListener) EnterMulDivMod(ctx *MulDivModContext) {}

// ExitMulDivMod is called when production MulDivMod is exited.
func (s *BaseCalcListener) ExitMulDivMod(ctx *MulDivModContext) {}

// EnterAddSub is called when production AddSub is entered.
func (s *BaseCalcListener) EnterAddSub(ctx *AddSubContext) {}

// ExitAddSub is called when production AddSub is exited.
func (s *BaseCalcListener) ExitAddSub(ctx *AddSubContext) {}

// EnterParens is called when production Parens is entered.
func (s *BaseCalcListener) EnterParens(ctx *ParensContext) {}

// ExitParens is called when production Parens is exited.
func (s *BaseCalcListener) ExitParens(ctx *ParensContext) {}

// EnterNum is called when production Num is entered.
func (s *BaseCalcListener) EnterNum(ctx *NumContext) {}

// ExitNum is called when production Num is exited.
func (s *BaseCalcListener) ExitNum(ctx *NumContext) {}

// EnterPow is called when production Pow is entered.
func (s *BaseCalcListener) EnterPow(ctx *PowContext) {}

// ExitPow is called when production Pow is exited.
func (s *BaseCalcListener) ExitPow(ctx *PowContext) {}

// EnterHex is called when production Hex is entered.
func (s *BaseCalcListener) EnterHex(ctx *HexContext) {}

// ExitHex is called when production Hex is exited.
func (s *BaseCalcListener) ExitHex(ctx *HexContext) {}

// EnterId is called when production Id is entered.
func (s *BaseCalcListener) EnterId(ctx *IdContext) {}

// ExitId is called when production Id is exited.
func (s *BaseCalcListener) ExitId(ctx *IdContext) {}

// EnterNumber is called when production number is entered.
func (s *BaseCalcListener) EnterNumber(ctx *NumberContext) {}

// ExitNumber is called when production number is exited.
func (s *BaseCalcListener) ExitNumber(ctx *NumberContext) {}
