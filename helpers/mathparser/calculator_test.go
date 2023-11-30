package mathparser

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAdd(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("1+2")
	as.NoError(err)
	as.Equal("3", res.ToText())
}

func TestParens(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("(1+2) *3")
	as.NoError(err)
	as.Equal("9", res.ToText())
}

func TestAny(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("1+2*3")
	as.NoError(err)
	as.Equal("7", res.ToText())
	resAll, err := ParseString(`1+2*3**4+0+0xa`)
	as.Equal("173", resAll.ToText())
}

func TestPoint(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("0.1+0.2")
	as.NoError(err)
	as.Equal("0.3", res.ToText())
	res, err = ParseString("0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1+0.1")
	as.NoError(err)
	as.Equal("2.1", res.ToText())
}

func TestFloatPointPrec(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("1+2+3")
	as.NoError(err)
	as.Equal("6", res.ToText())
}

func TestDiv(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("1/3 *3")
	as.NoError(err)
	as.Equal("1", res.ToText())
	res, err = ParseString("(3/5)^3")
	as.NoError(err)
	as.Equal("0.216", res.ToText())
}

func TestPow(t *testing.T) {
	as := require.New(t)
	res, err := ParseString("2^3")
	as.NoError(err)
	as.Equal("8", res.ToText())
	res, err = ParseString("2^0.5")
	as.NoError(err)
	as.Equal("1.4142", res.ToText())
	res, err = ParseString("(2^0.5)^2")
	as.NoError(err)
	as.Equal(res.ToText(), "2")
}
