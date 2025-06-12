package mathparser

import "fmt"

// CalcErrorType represents different error categories for the calculator.
type CalcErrorType int

const (
	// ErrorInvalidNumber indicates a number could not be parsed.
	ErrorInvalidNumber CalcErrorType = iota
	// ErrorUnknownCharacter is returned for an unrecognized rune in the input.
	ErrorUnknownCharacter
	// ErrorMismatchedParentheses indicates parentheses balance issues during parsing.
	ErrorMismatchedParentheses
	// ErrorUnexpectedToken covers situations where a token is not valid in context.
	ErrorUnexpectedToken
	// ErrorStackUnderflow occurs when evaluation pops from an empty stack.
	ErrorStackUnderflow
	// ErrorUnknownIdentifier denotes an identifier that isn't recognised.
	ErrorUnknownIdentifier
	// ErrorDivisionByZero is returned when attempting to divide by zero.
	ErrorDivisionByZero
	// ErrorInfiniteResult covers infinite or NaN results from floating ops.
	ErrorInfiniteResult
	// ErrorResultTooBig indicates a computed value would overflow practical limits.
	ErrorResultTooBig
	// ErrorInvalidExpression reports general expression validity failures.
	ErrorInvalidExpression
	// ErrorModuloRequiresInt when modulo is used with non-integers.
	ErrorModuloRequiresInt
	// ErrorModByZero for modulo by zero.
	ErrorModByZero
	// ErrorPermutationRequiresInt when permutation inputs are not integers.
	ErrorPermutationRequiresInt
	// ErrorInvalidPermutation when permutation parameters are out of bounds.
	ErrorInvalidPermutation
	// ErrorCombinationRequiresInt when combination inputs are not integers.
	ErrorCombinationRequiresInt
	// ErrorInvalidCombination when combination parameters are out of bounds.
	ErrorInvalidCombination
	// ErrorFactorialRequiresInt when factorial operand is not integer.
	ErrorFactorialRequiresInt
	// ErrorFactorialNegative when factorial operand is negative.
	ErrorFactorialNegative
)

// CalcError wraps an error with additional context such as the position in the input.
type CalcError struct {
	Typ CalcErrorType
	msg string
	pos int
}

// errorAt formats an error message and returns a CalcError with the given type and position.
func errorAt(pos int, typ CalcErrorType, format string, args ...interface{}) *CalcError {
	return &CalcError{
		Typ: typ,
		pos: pos,
		msg: fmt.Sprintf(format, args...),
	}
}

func (e *CalcError) Error() string {
	return e.msg
}
