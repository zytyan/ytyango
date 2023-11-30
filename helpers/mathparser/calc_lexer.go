// Code generated from C:/Users/manni/GolandProjects/ytyango/helpers/mathparser/Calc.g4 by ANTLR 4.13.1. DO NOT EDIT.

package mathparser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type CalcLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var CalcLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func calclexerLexerInit() {
	staticData := &CalcLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'='", "'('", "')'", "'*'", "'/'", "'+'", "'-'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "MUL", "DIV", "ADD", "SUB", "POW", "MOD", "ID", "INT",
		"FLOAT", "EXP", "HEX", "ENTER", "WS",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "MUL", "DIV", "ADD", "SUB", "POW", "MOD", "ID",
		"INT", "FLOAT", "EXP", "HEX", "ENTER", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 16, 128, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5,
		1, 5, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 3, 7, 51, 8, 7, 1, 8, 1, 8, 1, 8, 1,
		8, 3, 8, 57, 8, 8, 1, 9, 4, 9, 60, 8, 9, 11, 9, 12, 9, 61, 1, 10, 4, 10,
		65, 8, 10, 11, 10, 12, 10, 66, 1, 11, 4, 11, 70, 8, 11, 11, 11, 12, 11,
		71, 1, 11, 1, 11, 5, 11, 76, 8, 11, 10, 11, 12, 11, 79, 9, 11, 1, 11, 3,
		11, 82, 8, 11, 1, 11, 1, 11, 4, 11, 86, 8, 11, 11, 11, 12, 11, 87, 1, 11,
		3, 11, 91, 8, 11, 1, 11, 4, 11, 94, 8, 11, 11, 11, 12, 11, 95, 1, 11, 3,
		11, 99, 8, 11, 1, 12, 1, 12, 3, 12, 103, 8, 12, 1, 12, 4, 12, 106, 8, 12,
		11, 12, 12, 12, 107, 1, 13, 1, 13, 1, 13, 4, 13, 113, 8, 13, 11, 13, 12,
		13, 114, 1, 14, 3, 14, 118, 8, 14, 1, 14, 1, 14, 1, 15, 4, 15, 123, 8,
		15, 11, 15, 12, 15, 124, 1, 15, 1, 15, 0, 0, 16, 1, 1, 3, 2, 5, 3, 7, 4,
		9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14,
		29, 15, 31, 16, 1, 0, 7, 2, 0, 65, 90, 97, 122, 1, 0, 48, 57, 2, 0, 69,
		69, 101, 101, 2, 0, 43, 43, 45, 45, 2, 0, 88, 88, 120, 120, 3, 0, 48, 57,
		65, 70, 97, 102, 2, 0, 9, 9, 32, 32, 144, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0,
		0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0,
		0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1,
		0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27,
		1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 1, 33, 1, 0, 0, 0, 3,
		35, 1, 0, 0, 0, 5, 37, 1, 0, 0, 0, 7, 39, 1, 0, 0, 0, 9, 41, 1, 0, 0, 0,
		11, 43, 1, 0, 0, 0, 13, 45, 1, 0, 0, 0, 15, 50, 1, 0, 0, 0, 17, 56, 1,
		0, 0, 0, 19, 59, 1, 0, 0, 0, 21, 64, 1, 0, 0, 0, 23, 98, 1, 0, 0, 0, 25,
		100, 1, 0, 0, 0, 27, 109, 1, 0, 0, 0, 29, 117, 1, 0, 0, 0, 31, 122, 1,
		0, 0, 0, 33, 34, 5, 61, 0, 0, 34, 2, 1, 0, 0, 0, 35, 36, 5, 40, 0, 0, 36,
		4, 1, 0, 0, 0, 37, 38, 5, 41, 0, 0, 38, 6, 1, 0, 0, 0, 39, 40, 5, 42, 0,
		0, 40, 8, 1, 0, 0, 0, 41, 42, 5, 47, 0, 0, 42, 10, 1, 0, 0, 0, 43, 44,
		5, 43, 0, 0, 44, 12, 1, 0, 0, 0, 45, 46, 5, 45, 0, 0, 46, 14, 1, 0, 0,
		0, 47, 51, 5, 94, 0, 0, 48, 49, 5, 42, 0, 0, 49, 51, 5, 42, 0, 0, 50, 47,
		1, 0, 0, 0, 50, 48, 1, 0, 0, 0, 51, 16, 1, 0, 0, 0, 52, 57, 5, 37, 0, 0,
		53, 54, 5, 109, 0, 0, 54, 55, 5, 111, 0, 0, 55, 57, 5, 100, 0, 0, 56, 52,
		1, 0, 0, 0, 56, 53, 1, 0, 0, 0, 57, 18, 1, 0, 0, 0, 58, 60, 7, 0, 0, 0,
		59, 58, 1, 0, 0, 0, 60, 61, 1, 0, 0, 0, 61, 59, 1, 0, 0, 0, 61, 62, 1,
		0, 0, 0, 62, 20, 1, 0, 0, 0, 63, 65, 7, 1, 0, 0, 64, 63, 1, 0, 0, 0, 65,
		66, 1, 0, 0, 0, 66, 64, 1, 0, 0, 0, 66, 67, 1, 0, 0, 0, 67, 22, 1, 0, 0,
		0, 68, 70, 7, 1, 0, 0, 69, 68, 1, 0, 0, 0, 70, 71, 1, 0, 0, 0, 71, 69,
		1, 0, 0, 0, 71, 72, 1, 0, 0, 0, 72, 73, 1, 0, 0, 0, 73, 77, 5, 46, 0, 0,
		74, 76, 7, 1, 0, 0, 75, 74, 1, 0, 0, 0, 76, 79, 1, 0, 0, 0, 77, 75, 1,
		0, 0, 0, 77, 78, 1, 0, 0, 0, 78, 81, 1, 0, 0, 0, 79, 77, 1, 0, 0, 0, 80,
		82, 3, 25, 12, 0, 81, 80, 1, 0, 0, 0, 81, 82, 1, 0, 0, 0, 82, 99, 1, 0,
		0, 0, 83, 85, 5, 46, 0, 0, 84, 86, 7, 1, 0, 0, 85, 84, 1, 0, 0, 0, 86,
		87, 1, 0, 0, 0, 87, 85, 1, 0, 0, 0, 87, 88, 1, 0, 0, 0, 88, 90, 1, 0, 0,
		0, 89, 91, 3, 25, 12, 0, 90, 89, 1, 0, 0, 0, 90, 91, 1, 0, 0, 0, 91, 99,
		1, 0, 0, 0, 92, 94, 7, 1, 0, 0, 93, 92, 1, 0, 0, 0, 94, 95, 1, 0, 0, 0,
		95, 93, 1, 0, 0, 0, 95, 96, 1, 0, 0, 0, 96, 97, 1, 0, 0, 0, 97, 99, 3,
		25, 12, 0, 98, 69, 1, 0, 0, 0, 98, 83, 1, 0, 0, 0, 98, 93, 1, 0, 0, 0,
		99, 24, 1, 0, 0, 0, 100, 102, 7, 2, 0, 0, 101, 103, 7, 3, 0, 0, 102, 101,
		1, 0, 0, 0, 102, 103, 1, 0, 0, 0, 103, 105, 1, 0, 0, 0, 104, 106, 7, 1,
		0, 0, 105, 104, 1, 0, 0, 0, 106, 107, 1, 0, 0, 0, 107, 105, 1, 0, 0, 0,
		107, 108, 1, 0, 0, 0, 108, 26, 1, 0, 0, 0, 109, 110, 5, 48, 0, 0, 110,
		112, 7, 4, 0, 0, 111, 113, 7, 5, 0, 0, 112, 111, 1, 0, 0, 0, 113, 114,
		1, 0, 0, 0, 114, 112, 1, 0, 0, 0, 114, 115, 1, 0, 0, 0, 115, 28, 1, 0,
		0, 0, 116, 118, 5, 13, 0, 0, 117, 116, 1, 0, 0, 0, 117, 118, 1, 0, 0, 0,
		118, 119, 1, 0, 0, 0, 119, 120, 5, 10, 0, 0, 120, 30, 1, 0, 0, 0, 121,
		123, 7, 6, 0, 0, 122, 121, 1, 0, 0, 0, 123, 124, 1, 0, 0, 0, 124, 122,
		1, 0, 0, 0, 124, 125, 1, 0, 0, 0, 125, 126, 1, 0, 0, 0, 126, 127, 6, 15,
		0, 0, 127, 32, 1, 0, 0, 0, 17, 0, 50, 56, 61, 66, 71, 77, 81, 87, 90, 95,
		98, 102, 107, 114, 117, 124, 1, 6, 0, 0,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// CalcLexerInit initializes any static state used to implement CalcLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewCalcLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func CalcLexerInit() {
	staticData := &CalcLexerLexerStaticData
	staticData.once.Do(calclexerLexerInit)
}

// NewCalcLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewCalcLexer(input antlr.CharStream) *CalcLexer {
	CalcLexerInit()
	l := new(CalcLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &CalcLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "Calc.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// CalcLexer tokens.
const (
	CalcLexerT__0  = 1
	CalcLexerT__1  = 2
	CalcLexerT__2  = 3
	CalcLexerMUL   = 4
	CalcLexerDIV   = 5
	CalcLexerADD   = 6
	CalcLexerSUB   = 7
	CalcLexerPOW   = 8
	CalcLexerMOD   = 9
	CalcLexerID    = 10
	CalcLexerINT   = 11
	CalcLexerFLOAT = 12
	CalcLexerEXP   = 13
	CalcLexerHEX   = 14
	CalcLexerENTER = 15
	CalcLexerWS    = 16
)
