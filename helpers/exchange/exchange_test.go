package exchange

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseReq(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1.43 usd to cny")
	as.Nil(err)
	as.Equal(Req{From: "USD", To: "CNY", Amount: 1.43}, req)
}

func TestBadReq(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1.4.3 usd to cny")
	as.Equal(CashNotAvail, err)
	as.Equal(Req{}, req)
}

func TestGetExchangeRate(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1 usd to cny")
	as.Nil(err)
	resp, err := GetExchangeRate(req)
	as.Nil(err)
	as.NotEqual(0, resp)
	fmt.Println(resp)

	req, err = ParseExchangeRate("1 cny to jpy")
	as.Nil(err)
	resp, err = GetExchangeRate(req)
	as.Nil(err)
	as.NotEqual(0, resp)
	fmt.Println(resp)

	req, err = ParseExchangeRate("1 jpy to cny")
	as.Nil(err)
	resp, err = GetExchangeRate(req)
	as.Nil(err)
	as.NotEqual(0, resp)
	fmt.Println(resp)
}

func TestBadCash(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1 usd to cny")
	as.Nil(err)
	req.From = "ABC"
	_, err = GetExchangeRate(req)
	as.NotNil(err)
}

func TestOnlyFrom(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1 usd")
	as.Nil(err)
	resp, err := GetExchangeRate(req)
	as.Nil(err)
	as.NotEqual(0, resp)
	fmt.Println(resp)

	req, err = ParseExchangeRate("1 cny")
	as.Nil(err)
	resp, err = GetExchangeRate(req)
	as.Nil(err)
	as.NotEqual(0, resp)
	fmt.Println(resp)

}
