package backend

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	g "main/globalcfg"
	"main/http/backend/botapi"

	jsoniter "github.com/json-iterator/go"
)

type authContextKey struct{}

type WebInitUser struct {
	Id              int    `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPm bool   `json:"allows_write_to_pm"`
}

type AuthInfo struct {
	QueryId  string
	User     WebInitUser
	AuthDate time.Time
	Hash     string
}

type securityHandler struct{}

func (s *securityHandler) HandleTelegramAuth(ctx context.Context, _ botapi.OperationName, t botapi.TelegramAuth) (context.Context, error) {
	// store raw header value; actual verification happens in handler.
	return context.WithValue(ctx, authContextKey{}, t.APIKey), nil
}

var botVerifyKey = func() []byte {
	key := g.GetConfig().BotToken
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(key))
	return mac.Sum(nil)
}()

func checkTelegramAuth(str string) (AuthInfo, error) {
	split := strings.Split(str, "&")
	const hashPrefix = "hash"
	recvHash := ""
	data := make([]string, 0, len(split))
	for _, v := range split {
		key, value, _ := strings.Cut(v, "=")
		if key == hashPrefix {
			recvHash = value
			continue
		}
		key, err1 := url.QueryUnescape(key)
		value, err2 := url.QueryUnescape(value)
		if err1 != nil || err2 != nil {
			return AuthInfo{}, fmt.Errorf("url unescape err %v %v", err1, err2)
		}
		data = append(data, key+"="+value)
	}
	if recvHash == "" {
		return AuthInfo{}, fmt.Errorf("no hash")
	}

	slices.Sort(data)
	initData := []byte(strings.Join(data, "\n"))
	mac := hmac.New(sha256.New, botVerifyKey)
	mac.Write(initData)
	calcHash := hex.EncodeToString(mac.Sum(nil))
	if recvHash != calcHash {
		return AuthInfo{}, fmt.Errorf("wrong recvHash calc=%s*** recv=%s", calcHash[:4], recvHash)
	}
	var res AuthInfo
	for _, v := range data {
		key, value, _ := strings.Cut(v, "=")
		switch key {
		case "auth_date":
			parseInt, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return AuthInfo{}, err
			}
			res.AuthDate = time.Unix(parseInt, 0)
		case "hash":
			res.Hash = value
		case "query_id":
			res.QueryId = value
		case "user":
			var user WebInitUser
			if err := jsoniter.Unmarshal([]byte(value), &user); err != nil {
				return AuthInfo{}, err
			}
			res.User = user
		}
	}
	return res, nil
}
