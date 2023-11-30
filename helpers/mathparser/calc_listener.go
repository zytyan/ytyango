// Code generated from C:/Users/manni/GolandProjects/ytyango/helpers/mathparser/Calc.g4 by ANTLR 4.13.1. DO NOT EDIT.

package mathparser // Calc
import "github.com/antlr4-go/antlr/v4"

// CalcListener is a complete listener for a parse tree produced by CalcParser.
type CalcListener interface {
	antlr.ParseTreeListener

	// EnterProg is called when entering the prog production.
	EnterProg(c *ProgContext)

	// EnterStat is called when entering the stat production.
	EnterStat(c *StatContext)

	// EnterMulDivMod is called when entering the MulDivMod production.
	EnterMulDivMod(c *MulDivModContext)

	// EnterAddSub is called when entering the AddSub production.
	EnterAddSub(c *AddSubContext)

	// EnterParens is called when entering the Parens production.
	EnterParens(c *ParensContext)

	// EnterNum is called when entering the Num production.
	EnterNum(c *NumContext)

	// EnterPow is called when entering the Pow production.
	EnterPow(c *PowContext)

	// EnterHex is called when entering the Hex production.
	EnterHex(c *HexContext)

	// EnterId is called when entering the Id production.
	EnterId(c *IdContext)

	// EnterNumber is called when entering the number production.
	EnterNumber(c *NumberContext)

	// ExitProg is called when exiting the prog production.
	ExitProg(c *ProgContext)

	// ExitStat is called when exiting the stat production.
	ExitStat(c *StatContext)

	// ExitMulDivMod is called when exiting the MulDivMod production.
	ExitMulDivMod(c *MulDivModContext)

	// ExitAddSub is called when exiting the AddSub production.
	ExitAddSub(c *AddSubContext)

	// ExitParens is called when exiting the Parens production.
	ExitParens(c *ParensContext)

	// ExitNum is called when exiting the Num production.
	ExitNum(c *NumContext)

	// ExitPow is called when exiting the Pow production.
	ExitPow(c *PowContext)

	// ExitHex is called when exiting the Hex production.
	ExitHex(c *HexContext)

	// ExitId is called when exiting the Id production.
	ExitId(c *IdContext)

	// ExitNumber is called when exiting the number production.
	ExitNumber(c *NumberContext)
}
