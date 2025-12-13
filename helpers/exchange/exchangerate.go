package exchange

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type Rate map[string]float64

var exchangeCache = map[string]*ApiResp{}

type ApiResp struct {
	Result             string `json:"result"`
	Provider           string `json:"provider"`
	Documentation      string `json:"documentation"`
	TermsOfUse         string `json:"terms_of_use"`
	TimeLastUpdateUnix int64  `json:"time_last_update_unix"`
	TimeLastUpdateUtc  string `json:"time_last_update_utc"`
	TimeNextUpdateUnix int64  `json:"time_next_update_unix"`
	TimeNextUpdateUtc  string `json:"time_next_update_utc"`
	TimeEolUnix        int    `json:"time_eol_unix"`
	BaseCode           string `json:"base_code"`
	Rates              Rate   `json:"rates"`
}

type Resp struct {
	req      Req
	Result   float64
	UpdateAt time.Time
}

func getExchangeRateFromNet(base string) (*ApiResp, error) {
	get, err := http.Get(fmt.Sprintf("https://open.er-api.com/v6/latest/%s", base))
	if err != nil {
		return nil, err
	}
	lastReqTime = time.Now()
	defer get.Body.Close()
	var exchangeApi ApiResp
	err = jsoniter.NewDecoder(get.Body).Decode(&exchangeApi)
	if err != nil {
		return nil, err
	}
	return &exchangeApi, nil
}
func (a *ApiResp) NeedUpdate() bool {
	return time.Now().After(time.Unix(a.TimeNextUpdateUnix, 0))
}

func (a *ApiResp) Update() error {
	if !a.NeedUpdate() {
		return nil
	}
	resp, err := getExchangeRateFromNet(a.BaseCode)
	if err != nil {
		return err
	}
	*a = *resp
	return nil
}
func (a *ApiResp) LastUpdateAt() time.Time {
	return time.Unix(a.TimeLastUpdateUnix, 0)
}

var (
	ErrFromNotFound = errors.New("from currency not found")
	ErrToNotFound   = errors.New("to currency not found")
	CashNotAvail    = errors.New("currency not supported")
)

func (a *ApiResp) Exchange(req Req) (resp Resp, err error) {
	resp.req = req
	if req.From == req.To {
		resp.Result = req.Amount
		return resp, nil
	}
	rate := a.Rates
	fromRate, ok := rate[req.From]
	resp.UpdateAt = a.LastUpdateAt()
	if !ok {
		return resp, ErrFromNotFound
	}
	toRate, ok := rate[req.To]
	if !ok {
		return resp, ErrToNotFound
	}
	resp.Result = req.Amount * toRate / fromRate

	return resp, nil
}

var lastReqTime = time.Unix(0, 0)

func globalCanUpdate() bool {
	return time.Since(lastReqTime) > 1*time.Hour
}
func findLeastAvailableExchangeRate(prefer string) string {
	if prefer != "" {
		_, ok := exchangeCache[prefer]
		if ok {
			return prefer
		}
	}
	var target string
	least := int64(0)
	for k, v := range exchangeCache {
		if v.TimeLastUpdateUnix-least > 0 {
			least = v.TimeLastUpdateUnix
			target = k
		}
	}
	return target
}

func refreshCache(req Req) error {
	if !globalCanUpdate() {
		return nil
	}
	if apiResp, ok := exchangeCache[req.From]; !ok {
		resp, err := getExchangeRateFromNet(req.From)
		if err != nil {
			return err
		}
		exchangeCache[req.From] = resp
	} else {
		return apiResp.Update()
	}
	return nil
}

var exRe = regexp.MustCompile(`^(\d+(\.\d+)?)\s*([a-zA-Z]{3})\s*((to)?\s*([a-zA-Z]{3}))?$`)

type Req struct {
	Amount float64
	From   string
	To     string
}

var NotAValidExchangeReq = errors.New("not a valid exchange request")

func GetExchangeRate(req Req) (Resp, error) {
	var resp Resp
	resp.req = req
	if req.From == req.To {
		resp.UpdateAt = time.Now()
		resp.Result = req.Amount
		return resp, nil
	}
	if !IsAvailableCash(req.From) || !IsAvailableCash(req.To) {
		return resp, CashNotAvail
	}
	err := refreshCache(req)
	if err != nil {
		return resp, err
	}
	leastAvail := findLeastAvailableExchangeRate(req.From)
	e, ok := exchangeCache[leastAvail]
	if !ok {
		return resp, ErrFromNotFound
	}
	return e.Exchange(req)
}

func GetExchangeRateWithAlias(req Req, alias map[string]string) (Resp, error) {
	aliasedFrom, ok := alias[req.From]
	if ok {
		req.From = aliasedFrom
	}
	aliasedTo, ok := alias[req.To]
	if ok {
		req.To = aliasedTo
	}
	return GetExchangeRate(req)
}

func ParseExchangeRate(text string) (Req, error) {
	match := exRe.FindStringSubmatch(text)
	if match == nil {
		return Req{}, CashNotAvail
	}
	amount := match[1]
	from := strings.ToUpper(match[3])
	to := strings.ToUpper(match[6])
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return Req{}, NotAValidExchangeReq
	}
	if to == "" {
		to = "CNY"
	}
	return Req{
		Amount: amountFloat,
		From:   from,
		To:     to,
	}, nil
}
func IsExchangeRateCalc(text string) bool {
	return exRe.MatchString(text)
}
