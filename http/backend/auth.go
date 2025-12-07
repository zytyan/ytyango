package backend

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type webInitUser struct {
	Id              int    `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPm bool   `json:"allows_write_to_pm"`
}

type authInfo struct {
	QueryId  string      `json:"query_id"`
	User     webInitUser `json:"user"`
	AuthDate time.Time   `json:"auth_date"`
	Hash     string      `json:"hash"`
}

// checkTelegramAuth verifies Telegram WebApp init data string using the hashed bot token key.
func checkTelegramAuth(str string, verifyKey []byte) (res authInfo, err error) {
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
			err = fmt.Errorf("url unescape err %v %v", err1, err2)
			return
		}
		data = append(data, key+"="+value)
	}
	if recvHash == "" {
		err = fmt.Errorf("no hash")
		return
	}

	slices.Sort(data)
	initData := []byte(strings.Join(data, "\n"))
	mac := hmac.New(sha256.New, verifyKey)
	mac.Write(initData)
	calcHash := hex.EncodeToString(mac.Sum(nil))
	if recvHash != calcHash {
		err = fmt.Errorf("wrong recvHash calc=%s*** recv=%s", calcHash[:4], recvHash)
		return
	}
	for _, v := range data {
		key, value, _ := strings.Cut(v, "=")
		switch key {
		case "auth_date":
			parseInt, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return authInfo{}, err
			}
			res.AuthDate = time.Unix(parseInt, 0)
		case "hash":
			res.Hash = value
		case "query_id":
			res.QueryId = value
		case "user":
			var user webInitUser
			err = jsoniter.Unmarshal([]byte(value), &user)
			if err != nil {
				return
			}
			res.User = user
		}
	}
	return
}

func (h *Handler) verifyTgAuth(raw string, userID int64) error {
	info, err := checkTelegramAuth(raw, h.verifyKey)
	if err != nil {
		return err
	}
	if userID > 0 && int64(info.User.Id) != userID {
		return fmt.Errorf("tgauth user mismatch: expect %d got %d", userID, info.User.Id)
	}
	return nil
}
