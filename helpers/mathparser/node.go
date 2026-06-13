package mathparser

import (
	"math"
	"math/big"
)

const maxResultBits = 10000

type node interface {
	eval() (*big.Rat, error)
}

type numberNode struct {
	value *big.Rat
	pos   int
}

func (n *numberNode) eval() (*big.Rat, error) {
	val := new(big.Rat).Set(n.value)
	if err := checkRatBits(val, n.pos); err != nil {
		return nil, err
	}
	return val, nil
}

type identNode struct {
	name string
	pos  int
}

func (n *identNode) eval() (*big.Rat, error) {
	switch {
	case equalFoldASCII(n.name, "pi"):
		return new(big.Rat).Set(pi), nil
	case equalFoldASCII(n.name, "e"):
		return new(big.Rat).Set(e), nil
	default:
		return nil, errorAt(n.pos, ErrorUnknownIdentifier, "unknown identifier: %s", n.name)
	}
}

type unaryNode struct {
	op    TokenType
	expr  node
	pos   int
	label string
}

func (n *unaryNode) eval() (*big.Rat, error) {
	val, err := n.expr.eval()
	if err != nil {
		return nil, err
	}
	switch n.op {
	case PLUS:
		return checkRatResult(val, n.pos)
	case MINUS:
		val.Neg(val)
		return checkRatResult(val, n.pos)
	case SQRT:
		return sqrtRat(val, n.pos)
	default:
		return nil, errorAt(n.pos, ErrorUnexpectedToken, "unexpected unary operator: %s", n.label)
	}
}

type binaryNode struct {
	op          TokenType
	left, right node
	pos         int
	label       string
}

func (n *binaryNode) eval() (*big.Rat, error) {
	left, err := n.left.eval()
	if err != nil {
		return nil, err
	}
	right, err := n.right.eval()
	if err != nil {
		return nil, err
	}
	return evalBinary(n.op, left, right, n.pos, n.label)
}

type postfixNode struct {
	op    TokenType
	expr  node
	pos   int
	label string
}

func (n *postfixNode) eval() (*big.Rat, error) {
	val, err := n.expr.eval()
	if err != nil {
		return nil, err
	}
	switch n.op {
	case FACT:
		return evalFactorial(val, n.pos)
	case MOD:
		val.Quo(val, big.NewRat(100, 1))
		return checkRatResult(val, n.pos)
	default:
		return nil, errorAt(n.pos, ErrorUnexpectedToken, "unexpected postfix operator: %s", n.label)
	}
}

type absNode struct {
	expr node
	pos  int
}

func (n *absNode) eval() (*big.Rat, error) {
	val, err := n.expr.eval()
	if err != nil {
		return nil, err
	}
	val.Abs(val)
	return checkRatResult(val, n.pos)
}

func evalBinary(op TokenType, left, right *big.Rat, pos int, label string) (*big.Rat, error) {
	var tmpInt1, tmpInt2, tmpInt3 big.Int
	var tmpRat big.Rat

	switch op {
	case PLUS:
		left.Add(left, right)
	case MINUS:
		left.Sub(left, right)
	case MUL:
		left.Mul(left, right)
	case DIV:
		if right.Sign() == 0 {
			return nil, errorAt(pos, ErrorDivisionByZero, "division by zero")
		}
		left.Quo(left, right)
	case POW:
		exp, _ := right.Float64()
		base, _ := left.Float64()
		if math.IsInf(exp, 0) {
			return nil, errorAt(pos, ErrorInfiniteResult, "infinite exponent")
		}
		if math.IsInf(base, 0) {
			return nil, errorAt(pos, ErrorInfiniteResult, "infinite base number")
		}
		if !right.IsInt() {
			floatRet := math.Pow(base, exp)
			if math.IsInf(floatRet, 0) || math.IsNaN(floatRet) {
				return nil, errorAt(pos, ErrorInfiniteResult, "infinite float")
			}
			left.SetFloat64(floatRet)
			break
		}
		if right.Sign() < 0 && left.Sign() == 0 {
			return nil, errorAt(pos, ErrorDivisionByZero, "division by zero")
		}
		if right.Num().BitLen() > 62 {
			return nil, errorAt(pos, ErrorResultTooBig, "result too big")
		}
		expInt := int(right.Num().Int64())
		if estimatedPowRatBits(left, expInt) > maxResultBits {
			return nil, errorAt(pos, ErrorResultTooBig, "result too big")
		}
		tmpRat.Set(left)
		powRat(left, &tmpRat, expInt)
	case FLOORDIV:
		tmpInt1.Mul(left.Num(), right.Denom())
		tmpInt2.Mul(left.Denom(), right.Num())
		if tmpInt2.Sign() == 0 {
			return nil, errorAt(pos, ErrorDivisionByZero, "floor division by zero")
		}
		tmpInt3.Quo(&tmpInt1, &tmpInt2)
		left.SetInt(&tmpInt3)
	case MOD:
		if !left.IsInt() || !right.IsInt() {
			return nil, errorAt(pos, ErrorModuloRequiresInt, "modulo requires integers")
		}
		if right.Num().Sign() == 0 {
			return nil, errorAt(pos, ErrorModByZero, "mod by zero")
		}
		tmpInt1.Mod(left.Num(), right.Num())
		left.SetInt(&tmpInt1)
	case PERM:
		if !left.IsInt() || !right.IsInt() {
			return nil, errorAt(pos, ErrorPermutationRequiresInt, "permutation requires integers")
		}
		n := left.Num().Int64()
		r := right.Num().Int64()
		if n < 0 || r < 0 || r > n {
			return nil, errorAt(pos, ErrorInvalidPermutation, "invalid permutation")
		}
		if approxPermBits(n, r) > maxResultBits {
			return nil, errorAt(pos, ErrorResultTooBig, "result too big")
		}
		left.SetInt(permInt(&tmpInt1, n, r))
	case COMB:
		if !left.IsInt() || !right.IsInt() {
			return nil, errorAt(pos, ErrorCombinationRequiresInt, "combination requires integers")
		}
		n := left.Num().Int64()
		r := right.Num().Int64()
		if n < 0 || r < 0 || r > n {
			return nil, errorAt(pos, ErrorInvalidCombination, "invalid combination")
		}
		if approxCombBits(n, r) > maxResultBits {
			return nil, errorAt(pos, ErrorResultTooBig, "result too big")
		}
		left.SetInt(combInt(&tmpInt1, n, r))
	default:
		return nil, errorAt(pos, ErrorUnexpectedToken, "unexpected binary operator: %s", label)
	}
	return checkRatResult(left, pos)
}

func evalFactorial(val *big.Rat, pos int) (*big.Rat, error) {
	if !val.IsInt() {
		return nil, errorAt(pos, ErrorFactorialRequiresInt, "factorial requires integer")
	}
	n := val.Num().Int64()
	if n < 0 {
		return nil, errorAt(pos, ErrorFactorialNegative, "factorial of negative number")
	}
	if approxFactorialBits(n) > maxResultBits {
		return nil, errorAt(pos, ErrorResultTooBig, "result too big")
	}
	var tmpInt big.Int
	factorialInt(&tmpInt, n)
	val.SetInt(&tmpInt)
	return checkRatResult(val, pos)
}

func sqrtRat(val *big.Rat, pos int) (*big.Rat, error) {
	if val.Sign() < 0 {
		return nil, errorAt(pos, ErrorInfiniteResult, "infinite float")
	}
	numRoot := new(big.Int).Sqrt(val.Num())
	denRoot := new(big.Int).Sqrt(val.Denom())
	var numSquared, denSquared big.Int
	numSquared.Mul(numRoot, numRoot)
	denSquared.Mul(denRoot, denRoot)
	if numSquared.Cmp(val.Num()) == 0 && denSquared.Cmp(val.Denom()) == 0 {
		return checkRatResult(new(big.Rat).SetFrac(numRoot, denRoot), pos)
	}
	floatVal, _ := val.Float64()
	floatRet := math.Sqrt(floatVal)
	if math.IsInf(floatRet, 0) || math.IsNaN(floatRet) {
		return nil, errorAt(pos, ErrorInfiniteResult, "infinite float")
	}
	return checkRatResult(new(big.Rat).SetFloat64(floatRet), pos)
}

func checkRatResult(val *big.Rat, pos int) (*big.Rat, error) {
	if err := checkRatBits(val, pos); err != nil {
		return nil, err
	}
	return val, nil
}

func checkRatBits(val *big.Rat, pos int) error {
	if val.Num().BitLen() > maxResultBits || val.Denom().BitLen() > maxResultBits {
		return errorAt(pos, ErrorResultTooBig, "result too big")
	}
	return nil
}

func estimatedPowRatBits(base *big.Rat, exp int) int {
	if exp == 0 {
		return 1
	}
	if exp < 0 {
		return max(estimatedPowIntBits(base.Num(), -exp), estimatedPowIntBits(base.Denom(), -exp))
	}
	return max(estimatedPowIntBits(base.Num(), exp), estimatedPowIntBits(base.Denom(), exp))
}

func estimatedPowIntBits(x *big.Int, exp int) int {
	if x.Sign() == 0 {
		return 0
	}
	if exp == 0 {
		return 1
	}
	return int(math.Floor(float64(exp)*approxIntLog2(x))) + 1
}

func approxIntLog2(x *big.Int) float64 {
	bits := x.BitLen()
	if bits <= 0 {
		return 0
	}
	abs := new(big.Int).Abs(x)
	shift := max(bits-53, 0)
	top := new(big.Int).Rsh(abs, uint(shift))
	return float64(shift) + math.Log2(float64(top.Uint64()))
}

// powRat computes integer power for *big.Rat base and int exponent.
// dst is used as accumulator to reduce allocations.
func powRat(dst *big.Rat, base *big.Rat, exp int) *big.Rat {
	if exp < 0 {
		var inv big.Rat
		inv.Inv(base)
		return powRat(dst, &inv, -exp)
	}
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

func approxFactorialBits(n int64) float64 {
	lg, _ := math.Lgamma(float64(n) + 1)
	return lg / math.Ln2
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

func approxPermBits(n, r int64) float64 {
	lg1, _ := math.Lgamma(float64(n) + 1)
	lg2, _ := math.Lgamma(float64(n-r) + 1)
	return (lg1 - lg2) / math.Ln2
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

func approxCombBits(n, r int64) float64 {
	lg1, _ := math.Lgamma(float64(n) + 1)
	lg2, _ := math.Lgamma(float64(r) + 1)
	lg3, _ := math.Lgamma(float64(n-r) + 1)
	return (lg1 - lg2 - lg3) / math.Ln2
}
