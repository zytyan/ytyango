package exchange

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func useTestExchangeAPI(t *testing.T) {
	t.Helper()
	oldHTTP := exchangeHTTP
	oldAPIURLFmt := exchangeAPIURLFmt
	exchangeMu.Lock()
	oldCache := exchangeCache
	oldLastReqTime := lastReqTime
	exchangeCache = map[string]*ApiResp{}
	lastReqTime = time.Unix(0, 0)
	exchangeMu.Unlock()

	exchangeHTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		base := r.URL.Path[len("/latest/"):]
		body := `{
			"result":"success",
			"time_last_update_unix":1700000000,
			"time_next_update_unix":4102444800,
			"base_code":"` + base + `",
			"rates":{"USD":1,"CNY":7.2,"JPY":150}
		}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}
	exchangeAPIURLFmt = "https://example.test/latest/%s"
	t.Cleanup(func() {
		exchangeHTTP = oldHTTP
		exchangeAPIURLFmt = oldAPIURLFmt
		exchangeMu.Lock()
		exchangeCache = oldCache
		lastReqTime = oldLastReqTime
		exchangeMu.Unlock()
	})
}

func TestParseReq(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1.43 usd to cny")
	as.Nil(err)
	as.Equal(Req{From: "USD", To: "CNY", Amount: 1.43}, req)
}

func TestBadReq(t *testing.T) {
	as := assert.New(t)
	req, err := ParseExchangeRate("1.4.3 usd to cny")
	as.Equal(ErrCashNotAvail, err)
	as.Equal(Req{}, req)
}

func TestGetExchangeRate(t *testing.T) {
	useTestExchangeAPI(t)
	as := assert.New(t)
	req, err := ParseExchangeRate("1 usd to cny")
	as.Nil(err)
	resp, err := GetExchangeRate(req)
	as.Nil(err)
	as.Equal(7.2, resp.Result)

	req, err = ParseExchangeRate("1 cny to jpy")
	as.Nil(err)
	resp, err = GetExchangeRate(req)
	as.Nil(err)
	as.Equal(150.0/7.2, resp.Result)

	req, err = ParseExchangeRate("1 jpy to cny")
	as.Nil(err)
	resp, err = GetExchangeRate(req)
	as.Nil(err)
	as.Equal(7.2/150.0, resp.Result)
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
	useTestExchangeAPI(t)
	as := assert.New(t)
	req, err := ParseExchangeRate("1 usd")
	as.Nil(err)
	resp, err := GetExchangeRate(req)
	as.Nil(err)
	as.Equal(7.2, resp.Result)

	req, err = ParseExchangeRate("1 cny")
	as.Nil(err)
	resp, err = GetExchangeRate(req)
	as.Nil(err)
	as.Equal(1.0, resp.Result)

}

func TestGetExchangeRateConcurrent(t *testing.T) {
	useTestExchangeAPI(t)
	req, err := ParseExchangeRate("1 usd to cny")
	if err != nil {
		t.Fatalf("parse request: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 32)
	for i := 0; i < cap(errCh); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := GetExchangeRate(req)
			if err != nil {
				errCh <- err
				return
			}
			if resp.Result != 7.2 {
				errCh <- ErrNotAValidExchangeReq
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("GetExchangeRate failed: %v", err)
		}
	}
}
