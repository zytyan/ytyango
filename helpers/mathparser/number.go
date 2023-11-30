package mathparser

import (
	"math"
	"math/big"
	"strings"
)

type number struct {
	val *big.Rat
}

func (n *number) isInt() bool {
	return n.val.IsInt()
}

func (n *number) add(other *number) *number {
	return &number{val: new(big.Rat).Add(n.val, other.val)}
}

func (n *number) sub(other *number) *number {
	return &number{val: new(big.Rat).Sub(n.val, other.val)}
}

func (n *number) mul(other *number) *number {
	return &number{val: new(big.Rat).Mul(n.val, other.val)}
}

func (n *number) div(other *number) *number {
	return &number{val: new(big.Rat).Quo(n.val, other.val)}
}

// pow
func (n *number) pow(other *number) *number {
	if other.isInt() {
		if other.val.Cmp(big.NewRat(0, 1)) == 0 {
			return &number{val: big.NewRat(1, 1)}
		} else if other.val.Cmp(big.NewRat(-1, 1)) == 0 {
			return &number{val: new(big.Rat).Quo(big.NewRat(1, 1), n.val)}
		} else if other.val.Cmp(big.NewRat(1000, 1)) > 0 {
			panic("too large power")
		}
		num := n.val.Num()
		den := n.val.Denom()
		isNeg := n.val.Sign() < 0
		num = num.Abs(num)
		den = den.Abs(den)
		otNum := other.val.Num()
		otNumIsNeg := otNum.Sign() < 0
		if otNumIsNeg {
			otNum = otNum.Abs(otNum)
		}
		num = num.Exp(num, otNum, nil)
		den = den.Exp(den, otNum, nil)
		if otNumIsNeg {
			// 还原符号
			otNum = otNum.Neg(otNum)
		}

		if isNeg {
			num.Neg(num)
		}
		return &number{val: n.val}
	}
	a, _ := n.val.Float64()
	x, _ := other.val.Float64()
	return &number{val: new(big.Rat).SetFloat64(math.Pow(a, x))}
}

// powSafe if the result is too large, return nil

func (n *number) mod(other *number) *number {
	panic("not implemented")
}

func (n *number) toText() string {
	if n.val == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimRight(n.val.FloatString(4), "0"), ".")
}

//dice使用正则表达式单独实现
